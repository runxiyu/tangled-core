pub mod login {
    use crate::AppState;
    use atrium_api::{
        agent::{AtpAgent, store::MemorySessionStore},
        types::string::AtIdentifier,
    };
    use atrium_xrpc_client::isahc::IsahcClient;
    use axum::{
        extract::{Form, State},
        http::StatusCode,
        response::IntoResponse,
    };
    use serde::Deserialize;

    pub async fn get() -> String {
        "hello world".to_owned()
    }

    #[derive(Deserialize, Debug)]
    pub struct Req {
        handle: AtIdentifier,
        app_password: String,
    }

    pub async fn post(
        State(state): State<AppState>,
        session: tower_sessions::Session,
        Form(req): Form<Req>,
    ) -> impl IntoResponse {
        let did_doc = state.resolve_did_document(&req.handle).await.unwrap();
        let agent = AtpAgent::new(
            IsahcClient::new(did_doc.get_pds_endpoint().unwrap()),
            MemorySessionStore::default(),
        );
        let res = agent.login(req.handle, req.app_password).await.unwrap();
        println!("logged in as {:?} ({:?})", res.handle, res.did);
        session.insert("bild_session", res).await.unwrap();
        println!("stored session");
        StatusCode::OK
    }
}

pub mod index {
    use crate::AppState;
    use atrium_api::agent::{AtpAgent, store::MemorySessionStore};
    use atrium_api::types::string::AtIdentifier;
    use atrium_xrpc_client::isahc::IsahcClient;
    use axum::extract::State;

    pub async fn get(
        session: tower_sessions::Session,
        State(state): State<AppState>,
    ) -> &'static str {
        match session
            .get::<atrium_api::agent::Session>("bild_session")
            .await
            .unwrap()
        {
            None => "no session",
            Some(s) => {
                let did_doc = state
                    .resolve_did_document(&AtIdentifier::Did(s.did.clone()))
                    .await
                    .unwrap();
                let agent = AtpAgent::new(
                    IsahcClient::new(did_doc.get_pds_endpoint().unwrap()),
                    MemorySessionStore::default(),
                );
                println!("resuming session of {:?} ({:?})", s.handle, s.did);
                agent.resume_session(s).await.unwrap();
                "resuming session"
            }
        }
    }
}

pub mod keys {
    use crate::{AppState, db};
    use atrium_api::types::string::AtIdentifier;
    use axum::http::StatusCode;
    use axum::{
        extract::{Extension, Json, Query, State},
        response::IntoResponse,
    };
    use serde::{Deserialize, Serialize};
    use std::sync::Arc;

    #[derive(Deserialize)]
    pub struct GetReq {
        handle: AtIdentifier,
    }

    #[derive(Serialize)]
    pub struct GetRes {
        id: AtIdentifier,
        keys: Vec<String>,
    }

    pub async fn get(
        Extension(db): Extension<Arc<db::Db>>,
        State(state): State<AppState>,
        Query(req): Query<GetReq>,
    ) -> impl IntoResponse {
        let did_doc = state.resolve_did_document(&req.handle).await.unwrap();
        let keys = db
            .conn
            .call(|c| {
                let mut stmt = c.prepare("SELECT key FROM keys WHERE did = ?1")?;
                let keys = stmt
                    .query_map([did_doc.id], |row| Ok(row.get::<usize, String>(0)?))?
                    .filter_map(|r| r.ok())
                    .collect::<Vec<_>>();
                Ok(keys)
            })
            .await
            .unwrap();
        Json(GetRes {
            id: req.handle.clone(),
            keys,
        })
    }

    #[derive(Deserialize)]
    pub struct PutReq {
        key: String,
        name: String,
    }

    pub async fn put(
        Extension(db): Extension<Arc<db::Db>>,
        session: tower_sessions::Session,
        Json(req): Json<PutReq>,
    ) -> impl IntoResponse {
        if openssh_keys::PublicKey::parse(&req.key).is_err() {
            return StatusCode::BAD_REQUEST;
        }
        match session
            .get::<atrium_api::agent::Session>("bild_session")
            .await
            .unwrap()
        {
            None => StatusCode::UNAUTHORIZED,
            Some(sess) => {
                let did = sess.did.clone().into();
                let handle = sess.handle.clone().into();
                db.conn
                    .call(|c| {
                        c.execute(
                            "INSERT INTO keys (did, handle, key, name) VALUES (?1, ?2, ?3, ?4)",
                            [did, handle, req.key, req.name],
                        )?;
                        Ok(())
                    })
                    .await
                    .unwrap();
                StatusCode::OK
            }
        }
    }

    pub async fn keys_file(Extension(db): Extension<Arc<db::Db>>) -> impl IntoResponse {
        authorized_keys_file(db).await
    }

    // TODO: use this
    async fn write_authorized_keys_file(db: Arc<db::Db>) {
        let file = authorized_keys_file(db).await;
        tokio::fs::write("/home/git/.ssh/authorized_keys", &file)
            .await
            .unwrap();
    }

    // TODO: use this
    async fn authorized_keys_file(db: Arc<db::Db>) -> String {
        db.conn
            .call(|c| {
                let mut stmt = c.prepare("SELECT handle, key FROM keys")?;
                let file = stmt
                    .query_map([], |row| {
                        let handle = row.get::<usize, String>(0)?;
                        let key = row.get::<usize, String>(1)?;
                        Ok(format!(r##"command="/home/git/repoguard -base-dir /home/git -user {handle} -log-path /home/git/log ",no-port-forwarding,no-X11-forwarding,no-agent-forwarding,no-pty {key}"##))
                    })?
                    .filter_map(|r| r.ok())
                    .collect::<Vec<_>>()
                    .join("\n");
                Ok(file)
            })
            .await
            .unwrap()
    }
}

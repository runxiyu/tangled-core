package state

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/atproto/syntax"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/jetstream/pkg/models"
	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/go-chi/chi/v5"
	tangled "github.com/sotangled/tangled/api/tangled"
	"github.com/sotangled/tangled/appview"
	"github.com/sotangled/tangled/appview/auth"
	"github.com/sotangled/tangled/appview/db"
	"github.com/sotangled/tangled/appview/pages"
	"github.com/sotangled/tangled/jetstream"
	"github.com/sotangled/tangled/rbac"
)

type State struct {
	db       *db.DB
	auth     *auth.Auth
	enforcer *rbac.Enforcer
	tidClock *syntax.TIDClock
	pages    *pages.Pages
	resolver *appview.Resolver
	jc       *jetstream.JetstreamClient
}

func Make() (*State, error) {
	db, err := db.Make(appview.SqliteDbPath)
	if err != nil {
		return nil, err
	}

	auth, err := auth.Make()
	if err != nil {
		return nil, err
	}

	enforcer, err := rbac.NewEnforcer(appview.SqliteDbPath)
	if err != nil {
		return nil, err
	}

	clock := syntax.NewTIDClock(0)

	pgs := pages.NewPages()

	resolver := appview.NewResolver()

	jc, err := jetstream.NewJetstreamClient("appview", []string{tangled.GraphFollowNSID}, nil, slog.Default(), db, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create jetstream client: %w", err)
	}
	err = jc.StartJetstream(context.Background(), func(ctx context.Context, e *models.Event) error {
		if e.Kind != models.EventKindCommit {
			return nil
		}

		did := e.Did
		fmt.Println("got event", e.Commit.Collection, e.Commit.RKey, e.Commit.Record)
		raw := json.RawMessage(e.Commit.Record)

		switch e.Commit.Collection {
		case tangled.GraphFollowNSID:
			record := tangled.GraphFollow{}
			err := json.Unmarshal(raw, &record)
			if err != nil {
				log.Println("invalid record")
				return err
			}
			err = db.AddFollow(did, record.Subject, e.Commit.RKey)
			if err != nil {
				return fmt.Errorf("failed to add follow to db: %w", err)
			}
			return db.UpdateLastTimeUs(e.TimeUS)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start jetstream watcher: %w", err)
	}

	state := &State{
		db,
		auth,
		enforcer,
		clock,
		pgs,
		resolver,
		jc,
	}

	return state, nil
}

func (s *State) TID() string {
	return s.tidClock.Next().String()
}

func (s *State) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		err := s.pages.Login(w, pages.LoginParams{})
		if err != nil {
			log.Printf("rendering login page: %s", err)
		}
		return
	case http.MethodPost:
		handle := strings.TrimPrefix(r.FormValue("handle"), "@")
		appPassword := r.FormValue("app_password")

		resolved, err := s.resolver.ResolveIdent(ctx, handle)
		if err != nil {
			log.Println("failed to resolve handle:", err)
			s.pages.Notice(w, "login-msg", fmt.Sprintf("\"%s\" is an invalid handle.", handle))
			return
		}

		atSession, err := s.auth.CreateInitialSession(ctx, resolved, appPassword)
		if err != nil {
			s.pages.Notice(w, "login-msg", "Invalid handle or password.")
			return
		}
		sessionish := auth.CreateSessionWrapper{ServerCreateSession_Output: atSession}

		err = s.auth.StoreSession(r, w, &sessionish, resolved.PDSEndpoint())
		if err != nil {
			s.pages.Notice(w, "login-msg", "Failed to login, try again later.")
			return
		}

		log.Printf("successfully saved session for %s (%s)", atSession.Handle, atSession.Did)
		s.pages.HxRedirect(w, "/")
		return
	}
}

func (s *State) Logout(w http.ResponseWriter, r *http.Request) {
	s.auth.ClearSession(r, w)
	s.pages.HxRedirect(w, "/")
}

func (s *State) Timeline(w http.ResponseWriter, r *http.Request) {
	user := s.auth.GetUser(r)

	timeline, err := s.db.MakeTimeline()
	if err != nil {
		log.Println(err)
		s.pages.Notice(w, "timeline", "Uh oh! Failed to load timeline.")
	}

	var didsToResolve []string
	for _, ev := range timeline {
		if ev.Repo != nil {
			didsToResolve = append(didsToResolve, ev.Repo.Did)
		}
		if ev.Follow != nil {
			didsToResolve = append(didsToResolve, ev.Follow.UserDid)
			didsToResolve = append(didsToResolve, ev.Follow.SubjectDid)
		}
	}

	resolvedIds := s.resolver.ResolveIdents(r.Context(), didsToResolve)
	didHandleMap := make(map[string]string)
	for _, identity := range resolvedIds {
		if !identity.Handle.IsInvalidHandle() {
			didHandleMap[identity.DID.String()] = fmt.Sprintf("@%s", identity.Handle.String())
		} else {
			didHandleMap[identity.DID.String()] = identity.DID.String()
		}
	}

	s.pages.Timeline(w, pages.TimelineParams{
		LoggedInUser: user,
		Timeline:     timeline,
		DidHandleMap: didHandleMap,
	})

	return
}

// requires auth
func (s *State) RegistrationKey(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// list open registrations under this did

		return
	case http.MethodPost:
		session, err := s.auth.Store.Get(r, appview.SessionName)
		if err != nil || session.IsNew {
			log.Println("unauthorized attempt to generate registration key")
			http.Error(w, "Forbidden", http.StatusUnauthorized)
			return
		}

		did := session.Values[appview.SessionDid].(string)

		// check if domain is valid url, and strip extra bits down to just host
		domain := r.FormValue("domain")
		if domain == "" {
			http.Error(w, "Invalid form", http.StatusBadRequest)
			return
		}

		key, err := s.db.GenerateRegistrationKey(domain, did)

		if err != nil {
			log.Println(err)
			http.Error(w, "unable to register this domain", http.StatusNotAcceptable)
			return
		}

		w.Write([]byte(key))
	}
}

func (s *State) Keys(w http.ResponseWriter, r *http.Request) {
	user := chi.URLParam(r, "user")
	user = strings.TrimPrefix(user, "@")

	if user == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id, err := s.resolver.ResolveIdent(r.Context(), user)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	pubKeys, err := s.db.GetPublicKeys(id.DID.String())
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if len(pubKeys) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	for _, k := range pubKeys {
		key := strings.TrimRight(k.Key, "\n")
		w.Write([]byte(fmt.Sprintln(key)))
	}
}

// create a signed request and check if a node responds to that
func (s *State) InitKnotServer(w http.ResponseWriter, r *http.Request) {
	user := s.auth.GetUser(r)

	domain := chi.URLParam(r, "domain")
	if domain == "" {
		http.Error(w, "malformed url", http.StatusBadRequest)
		return
	}
	log.Println("checking ", domain)

	secret, err := s.db.GetRegistrationKey(domain)
	if err != nil {
		log.Printf("no key found for domain %s: %s\n", domain, err)
		return
	}

	client, err := NewSignedClient(domain, secret)
	if err != nil {
		log.Println("failed to create client to ", domain)
	}

	resp, err := client.Init(user.Did)
	if err != nil {
		w.Write([]byte("no dice"))
		log.Println("domain was unreachable after 5 seconds")
		return
	}

	if resp.StatusCode == http.StatusConflict {
		log.Println("status conflict", resp.StatusCode)
		w.Write([]byte("already registered, sorry!"))
		return
	}

	if resp.StatusCode != http.StatusNoContent {
		log.Println("status nok", resp.StatusCode)
		w.Write([]byte("no dice"))
		return
	}

	// verify response mac
	signature := resp.Header.Get("X-Signature")
	signatureBytes, err := hex.DecodeString(signature)
	if err != nil {
		return
	}

	expectedMac := hmac.New(sha256.New, []byte(secret))
	expectedMac.Write([]byte("ok"))

	if !hmac.Equal(expectedMac.Sum(nil), signatureBytes) {
		log.Printf("response body signature mismatch: %x\n", signatureBytes)
		return
	}

	// mark as registered
	err = s.db.Register(domain)
	if err != nil {
		log.Println("failed to register domain", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// set permissions for this did as owner
	reg, err := s.db.RegistrationByDomain(domain)
	if err != nil {
		log.Println("failed to register domain", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// add basic acls for this domain
	err = s.enforcer.AddDomain(domain)
	if err != nil {
		log.Println("failed to setup owner of domain", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// add this did as owner of this domain
	err = s.enforcer.AddOwner(domain, reg.ByDid)
	if err != nil {
		log.Println("failed to setup owner of domain", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("check success"))
}

func (s *State) KnotServerInfo(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")
	if domain == "" {
		http.Error(w, "malformed url", http.StatusBadRequest)
		return
	}

	user := s.auth.GetUser(r)
	reg, err := s.db.RegistrationByDomain(domain)
	if err != nil {
		w.Write([]byte("failed to pull up registration info"))
		return
	}

	var members []string
	if reg.Registered != nil {
		members, err = s.enforcer.GetUserByRole("server:member", domain)
		if err != nil {
			w.Write([]byte("failed to fetch member list"))
			return
		}
	}

	ok, err := s.enforcer.IsServerOwner(user.Did, domain)
	isOwner := err == nil && ok

	p := pages.KnotParams{
		LoggedInUser: user,
		Registration: reg,
		Members:      members,
		IsOwner:      isOwner,
	}

	s.pages.Knot(w, p)
}

// get knots registered by this user
func (s *State) Knots(w http.ResponseWriter, r *http.Request) {
	// for now, this is just pubkeys
	user := s.auth.GetUser(r)
	registrations, err := s.db.RegistrationsByDid(user.Did)
	if err != nil {
		log.Println(err)
	}

	s.pages.Knots(w, pages.KnotsParams{
		LoggedInUser:  user,
		Registrations: registrations,
	})
}

// list members of domain, requires auth and requires owner status
func (s *State) ListMembers(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")
	if domain == "" {
		http.Error(w, "malformed url", http.StatusBadRequest)
		return
	}

	// list all members for this domain
	memberDids, err := s.enforcer.GetUserByRole("server:member", domain)
	if err != nil {
		w.Write([]byte("failed to fetch member list"))
		return
	}

	w.Write([]byte(strings.Join(memberDids, "\n")))
	return
}

// add member to domain, requires auth and requires invite access
func (s *State) AddMember(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")
	if domain == "" {
		http.Error(w, "malformed url", http.StatusBadRequest)
		return
	}

	memberDid := r.FormValue("member")
	if memberDid == "" {
		http.Error(w, "malformed form", http.StatusBadRequest)
		return
	}

	memberIdent, err := s.resolver.ResolveIdent(r.Context(), memberDid)
	if err != nil {
		w.Write([]byte("failed to resolve member did to a handle"))
		return
	}
	log.Printf("adding %s to %s\n", memberIdent.Handle.String(), domain)

	// announce this relation into the firehose, store into owners' pds
	client, _ := s.auth.AuthorizedClient(r)
	currentUser := s.auth.GetUser(r)
	addedAt := time.Now().Format(time.RFC3339)
	resp, err := comatproto.RepoPutRecord(r.Context(), client, &comatproto.RepoPutRecord_Input{
		Collection: tangled.KnotMemberNSID,
		Repo:       currentUser.Did,
		Rkey:       s.TID(),
		Record: &lexutil.LexiconTypeDecoder{
			Val: &tangled.KnotMember{
				Member:  memberIdent.DID.String(),
				Domain:  domain,
				AddedAt: &addedAt,
			}},
	})

	// invalid record
	if err != nil {
		log.Printf("failed to create record: %s", err)
		return
	}
	log.Println("created atproto record: ", resp.Uri)

	secret, err := s.db.GetRegistrationKey(domain)
	if err != nil {
		log.Printf("no key found for domain %s: %s\n", domain, err)
		return
	}

	ksClient, err := NewSignedClient(domain, secret)
	if err != nil {
		log.Println("failed to create client to ", domain)
		return
	}

	ksResp, err := ksClient.AddMember(memberIdent.DID.String())
	if err != nil {
		log.Printf("failed to make request to %s: %s", domain, err)
		return
	}

	if ksResp.StatusCode != http.StatusNoContent {
		w.Write([]byte(fmt.Sprint("knotserver failed to add member: ", err)))
		return
	}

	err = s.enforcer.AddMember(domain, memberIdent.DID.String())
	if err != nil {
		w.Write([]byte(fmt.Sprint("failed to add member: ", err)))
		return
	}

	w.Write([]byte(fmt.Sprint("added member: ", memberIdent.Handle.String())))
}

func (s *State) RemoveMember(w http.ResponseWriter, r *http.Request) {
}

func (s *State) AddRepo(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		user := s.auth.GetUser(r)
		knots, err := s.enforcer.GetDomainsForUser(user.Did)

		if err != nil {
			s.pages.Notice(w, "repo", "Invalid user account.")
			return
		}

		s.pages.NewRepo(w, pages.NewRepoParams{
			LoggedInUser: user,
			Knots:        knots,
		})
	case http.MethodPost:
		user := s.auth.GetUser(r)

		domain := r.FormValue("domain")
		if domain == "" {
			s.pages.Notice(w, "repo", "Invalid form submission&mdash;missing knot domain.")
			return
		}

		repoName := r.FormValue("name")
		if repoName == "" {
			s.pages.Notice(w, "repo", "Invalid repo name.")
			return
		}

		ok, err := s.enforcer.E.Enforce(user.Did, domain, domain, "repo:create")
		if err != nil || !ok {
			s.pages.Notice(w, "repo", "You do not have permission to create a repo in this knot.")
			return
		}

		secret, err := s.db.GetRegistrationKey(domain)
		if err != nil {
			s.pages.Notice(w, "repo", fmt.Sprintf("No registration key found for knot %s.", domain))
			return
		}

		client, err := NewSignedClient(domain, secret)
		if err != nil {
			s.pages.Notice(w, "repo", "Failed to connect to knot server.")
			return
		}

		resp, err := client.NewRepo(user.Did, repoName)
		if err != nil {
			s.pages.Notice(w, "repo", "Failed to create repository on knot server.")
			return
		}

		switch resp.StatusCode {
		case http.StatusConflict:
			s.pages.Notice(w, "repo", "A repository with that name already exists.")
			return
		case http.StatusInternalServerError:
			s.pages.Notice(w, "repo", "Failed to create repository on knot. Try again later.")
		case http.StatusNoContent:
			// continue
		}

		rkey := s.TID()
		repo := &db.Repo{
			Did:  user.Did,
			Name: repoName,
			Knot: domain,
			Rkey: rkey,
		}

		xrpcClient, _ := s.auth.AuthorizedClient(r)

		addedAt := time.Now().Format(time.RFC3339)
		atresp, err := comatproto.RepoPutRecord(r.Context(), xrpcClient, &comatproto.RepoPutRecord_Input{
			Collection: tangled.RepoNSID,
			Repo:       user.Did,
			Rkey:       rkey,
			Record: &lexutil.LexiconTypeDecoder{
				Val: &tangled.Repo{
					Knot:    repo.Knot,
					Name:    repoName,
					AddedAt: &addedAt,
					Owner:   user.Did,
				}},
		})
		if err != nil {
			log.Printf("failed to create record: %s", err)
			s.pages.Notice(w, "repo", "Failed to announce repository creation.")
			return
		}
		log.Println("created repo record: ", atresp.Uri)

		err = s.db.AddRepo(repo)
		if err != nil {
			log.Println(err)
			s.pages.Notice(w, "repo", "Failed to save repository information.")
			return
		}

		// acls
		p, _ := securejoin.SecureJoin(user.Did, repoName)
		err = s.enforcer.AddRepo(user.Did, domain, p)
		if err != nil {
			log.Println(err)
			s.pages.Notice(w, "repo", "Failed to set up repository permissions.")
			return
		}

		s.pages.HxLocation(w, fmt.Sprintf("/@%s/%s", user.Handle, repoName))
		return
	}
}

func (s *State) ProfilePage(w http.ResponseWriter, r *http.Request) {
	didOrHandle := chi.URLParam(r, "user")
	if didOrHandle == "" {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	ident, err := s.resolver.ResolveIdent(r.Context(), didOrHandle)
	if err != nil {
		log.Printf("resolving identity: %s", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	repos, err := s.db.GetAllReposByDid(ident.DID.String())
	if err != nil {
		log.Printf("getting repos for %s: %s", ident.DID.String(), err)
	}

	collaboratingRepos, err := s.db.CollaboratingIn(ident.DID.String())
	if err != nil {
		log.Printf("getting collaborating repos for %s: %s", ident.DID.String(), err)
	}

	followers, following, err := s.db.GetFollowerFollowing(ident.DID.String())
	if err != nil {
		log.Printf("getting follow stats repos for %s: %s", ident.DID.String(), err)
	}

	loggedInUser := s.auth.GetUser(r)
	followStatus := db.IsNotFollowing
	if loggedInUser != nil {
		followStatus = s.db.GetFollowStatus(loggedInUser.Did, ident.DID.String())
	}

	s.pages.ProfilePage(w, pages.ProfilePageParams{
		LoggedInUser:       loggedInUser,
		UserDid:            ident.DID.String(),
		UserHandle:         ident.Handle.String(),
		Repos:              repos,
		CollaboratingRepos: collaboratingRepos,
		ProfileStats: pages.ProfileStats{
			Followers: followers,
			Following: following,
		},
		FollowStatus: db.FollowStatus(followStatus),
	})
}

func (s *State) Follow(w http.ResponseWriter, r *http.Request) {
	currentUser := s.auth.GetUser(r)

	subject := r.URL.Query().Get("subject")
	if subject == "" {
		log.Println("invalid form")
		return
	}

	subjectIdent, err := s.resolver.ResolveIdent(r.Context(), subject)
	if err != nil {
		log.Println("failed to follow, invalid did")
	}

	if currentUser.Did == subjectIdent.DID.String() {
		log.Println("cant follow or unfollow yourself")
		return
	}

	client, _ := s.auth.AuthorizedClient(r)

	switch r.Method {
	case http.MethodPost:
		createdAt := time.Now().Format(time.RFC3339)
		rkey := s.TID()
		resp, err := comatproto.RepoPutRecord(r.Context(), client, &comatproto.RepoPutRecord_Input{
			Collection: tangled.GraphFollowNSID,
			Repo:       currentUser.Did,
			Rkey:       rkey,
			Record: &lexutil.LexiconTypeDecoder{
				Val: &tangled.GraphFollow{
					Subject:   subjectIdent.DID.String(),
					CreatedAt: createdAt,
				}},
		})
		if err != nil {
			log.Println("failed to create atproto record", err)
			return
		}

		err = s.db.AddFollow(currentUser.Did, subjectIdent.DID.String(), rkey)
		if err != nil {
			log.Println("failed to follow", err)
			return
		}

		log.Println("created atproto record: ", resp.Uri)

		w.Write([]byte(fmt.Sprintf(`
			<button id="followBtn"
				class="btn mt-2"
				hx-delete="/follow?subject=%s"
				hx-trigger="click"
				hx-target="#followBtn"
				hx-swap="outerHTML">
				Unfollow
			</button>
		`, subjectIdent.DID.String())))

		return
	case http.MethodDelete:
		// find the record in the db
		follow, err := s.db.GetFollow(currentUser.Did, subjectIdent.DID.String())
		if err != nil {
			log.Println("failed to get follow relationship")
			return
		}

		_, err = comatproto.RepoDeleteRecord(r.Context(), client, &comatproto.RepoDeleteRecord_Input{
			Collection: tangled.GraphFollowNSID,
			Repo:       currentUser.Did,
			Rkey:       follow.RKey,
		})

		if err != nil {
			log.Println("failed to unfollow")
			return
		}

		err = s.db.DeleteFollow(currentUser.Did, subjectIdent.DID.String())
		if err != nil {
			log.Println("failed to delete follow from DB")
			// this is not an issue, the firehose event might have already done this
		}

		w.Write([]byte(fmt.Sprintf(`
			<button id="followBtn"
				class="btn mt-2"
				hx-post="/follow?subject=%s"
				hx-trigger="click"
				hx-target="#followBtn"
				hx-swap="outerHTML">
				Follow
			</button>
		`, subjectIdent.DID.String())))
		return
	}

}

func (s *State) Router() http.Handler {
	router := chi.NewRouter()

	router.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		pat := chi.URLParam(r, "*")
		if strings.HasPrefix(pat, "did:") || strings.HasPrefix(pat, "@") {
			s.UserRouter().ServeHTTP(w, r)
		} else {
			s.StandardRouter().ServeHTTP(w, r)
		}
	})

	return router
}

func (s *State) UserRouter() http.Handler {
	r := chi.NewRouter()

	// strip @ from user
	r.Use(StripLeadingAt)

	r.With(ResolveIdent(s)).Route("/{user}", func(r chi.Router) {
		r.Get("/", s.ProfilePage)
		r.With(ResolveRepoKnot(s)).Route("/{repo}", func(r chi.Router) {
			r.Get("/", s.RepoIndex)
			r.Get("/log/{ref}", s.RepoLog)
			r.Route("/tree/{ref}", func(r chi.Router) {
				r.Get("/", s.RepoIndex)
				r.Get("/*", s.RepoTree)
			})
			r.Get("/commit/{ref}", s.RepoCommit)
			r.Get("/branches", s.RepoBranches)
			r.Get("/tags", s.RepoTags)
			r.Get("/blob/{ref}/*", s.RepoBlob)

			// These routes get proxied to the knot
			r.Get("/info/refs", s.InfoRefs)
			r.Post("/git-upload-pack", s.UploadPack)

			// settings routes, needs auth
			r.Group(func(r chi.Router) {
				r.With(RepoPermissionMiddleware(s, "repo:settings")).Route("/settings", func(r chi.Router) {
					r.Get("/", s.RepoSettings)
					r.With(RepoPermissionMiddleware(s, "repo:invite")).Put("/collaborator", s.AddCollaborator)
				})
			})
		})
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		s.pages.Error404(w)
	})

	return r
}

func (s *State) StandardRouter() http.Handler {
	r := chi.NewRouter()

	r.Handle("/static/*", s.pages.Static())

	r.Get("/", s.Timeline)

	r.Get("/logout", s.Logout)

	r.Get("/login", s.Login)
	r.Post("/login", s.Login)

	r.Route("/knots", func(r chi.Router) {
		r.Use(AuthMiddleware(s))
		r.Get("/", s.Knots)
		r.Post("/key", s.RegistrationKey)

		r.Route("/{domain}", func(r chi.Router) {
			r.Post("/init", s.InitKnotServer)
			r.Get("/", s.KnotServerInfo)
			r.Route("/member", func(r chi.Router) {
				r.Use(RoleMiddleware(s, "server:owner"))
				r.Get("/", s.ListMembers)
				r.Put("/", s.AddMember)
				r.Delete("/", s.RemoveMember)
			})
		})
	})

	r.Route("/repo", func(r chi.Router) {
		r.Route("/new", func(r chi.Router) {
			r.Get("/", s.AddRepo)
			r.Post("/", s.AddRepo)
		})
		// r.Post("/import", s.ImportRepo)
	})

	r.With(AuthMiddleware(s)).Route("/follow", func(r chi.Router) {
		r.Post("/", s.Follow)
		r.Delete("/", s.Follow)
	})

	r.Route("/settings", func(r chi.Router) {
		r.Use(AuthMiddleware(s))
		r.Get("/", s.Settings)
		r.Put("/keys", s.SettingsKeys)
	})

	r.Get("/keys/{user}", s.Keys)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		s.pages.Error404(w)
	})
	return r
}

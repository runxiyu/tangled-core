package state

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/icyphox/bild/appview"
	"github.com/icyphox/bild/appview/auth"
	"github.com/icyphox/bild/appview/db"
)

type State struct {
	Db   *db.DB
	Auth *auth.Auth
}

func Make() (*State, error) {
	db, err := db.Make("appview.db")
	if err != nil {
		return nil, err
	}

	auth, err := auth.Make()
	if err != nil {
		return nil, err
	}

	return &State{db, auth}, nil
}

func (s *State) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		log.Println("unimplemented")
		return
	case http.MethodPost:
		username := r.FormValue("username")
		appPassword := r.FormValue("password")

		atSession, err := s.Auth.CreateInitialSession(ctx, username, appPassword)
		if err != nil {
			log.Printf("creating initial session: %s", err)
			return
		}
		sessionish := auth.CreateSessionWrapper{atSession}

		err = s.Auth.StoreSession(r, w, &sessionish)
		if err != nil {
			log.Printf("storing session: %s", err)
			return
		}

		log.Printf("successfully saved session for %s (%s)", atSession.Handle, atSession.Did)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
}

// requires auth
func (s *State) Key(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// list open registrations under this did

		return
	case http.MethodPost:
		session, err := s.Auth.Store.Get(r, appview.SESSION_NAME)
		if err != nil || session.IsNew {
			log.Println("unauthorized attempt to generate registration key")
			http.Error(w, "Forbidden", http.StatusUnauthorized)
			return
		}

		did := session.Values[appview.SESSION_DID].(string)

		// check if domain is valid url, and strip extra bits down to just host
		domain := r.FormValue("domain")
		if domain == "" || err != nil {
			log.Println(err)
			http.Error(w, "Invalid form", http.StatusBadRequest)
			return
		}

		key, err := s.Db.GenerateRegistrationKey(domain, did)

		if err != nil {
			log.Println(err)
			http.Error(w, "unable to register this domain", http.StatusNotAcceptable)
			return
		}

		w.Write([]byte(key))
		return
	}
}

// create a signed request and check if a node responds to that
//
// we should also rate limit these checks to avoid ddosing knotservers
func (s *State) Check(w http.ResponseWriter, r *http.Request) {
	domain := r.FormValue("domain")
	if domain == "" {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	log.Println("checking ", domain)

	secret, err := s.Db.GetRegistrationKey(domain)
	if err != nil {
		log.Printf("no key found for domain %s: %s\n", domain, err)
		return
	}
	log.Println("has secret ", secret)

	// make a request do the knotserver with an empty body and above signature
	url := fmt.Sprintf("http://%s/health", domain)

	pingRequest, err := buildPingRequest(url, secret)
	if err != nil {
		log.Println("failed to build ping request", err)
		return
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Do(pingRequest)
	if err != nil {
		w.Write([]byte("no dice"))
		log.Println("domain was unreachable after 5 seconds")
		return
	}

	if resp.StatusCode != http.StatusOK {
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

	w.Write([]byte("check success"))

	// mark as registered
	err = s.Db.Register(domain)
	if err != nil {
		log.Println("failed to register domain", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	return
}

func buildPingRequest(url, secret string) (*http.Request, error) {
	pingRequest, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().Format(time.RFC3339)
	mac := hmac.New(sha256.New, []byte(secret))
	message := pingRequest.Method + pingRequest.URL.Path + timestamp
	mac.Write([]byte(message))
	signature := hex.EncodeToString(mac.Sum(nil))

	pingRequest.Header.Set("X-Signature", signature)
	pingRequest.Header.Set("X-Timestamp", timestamp)

	return pingRequest, nil
}

func (s *State) Router() http.Handler {
	r := chi.NewRouter()

	r.Post("/login", s.Login)

	r.Route("/node", func(r chi.Router) {
		r.Post("/check", s.Check)

		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware(s))
			r.Post("/key", s.Key)
		})
	})

	return r
}

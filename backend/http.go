package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	qv "github.com/jackc/quo_vadis"
	log "gopkg.in/inconshreveable/log15.v2"
	"net/http"
	"time"
)

type EnvHandlerFunc func(w http.ResponseWriter, req *http.Request, env *environment)

func EnvHandler(userRepo UserRepository, sessionRepo SessionRepository, logger log.Logger, f EnvHandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		userID, err := getUserIDFromSession(req, sessionRepo)
		if err == ErrNotFound {
			// TODO
		}
		env := &environment{userID: userID, userRepo: userRepo, sessionRepo: sessionRepo, logger: logger}
		f(w, req, env)
	})
}

func AuthenticatedHandler(f EnvHandlerFunc) EnvHandlerFunc {
	return EnvHandlerFunc(func(w http.ResponseWriter, req *http.Request, env *environment) {
		if env.userID == 0 {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, "Bad or missing X-Authentication header")
			return
		}
		f(w, req, env)
	})
}

type environment struct {
	userID      int32
	userRepo    UserRepository
	sessionRepo SessionRepository
	logger      log.Logger
}

func NewAPIHandler(userRepo UserRepository, sessionRepo SessionRepository, logger log.Logger) http.Handler {
	router := qv.NewRouter()

	router.Post("/register", EnvHandler(userRepo, sessionRepo, logger, RegisterHandler))
	router.Post("/sessions", EnvHandler(userRepo, sessionRepo, logger, CreateSessionHandler))
	router.Delete("/sessions/:id", EnvHandler(userRepo, sessionRepo, logger, AuthenticatedHandler(DeleteSessionHandler)))

	return router
}

func getUserIDFromSession(req *http.Request, sessionRepo SessionRepository) (userID int32, err error) {
	token := req.Header.Get("X-Authentication")

	sessionID, err := hex.DecodeString(token)
	if err != nil {
		return 0, err
	}

	session, err := sessionRepo.GetSession(sessionID)
	if err != nil {
		return 0, err
	}

	return session.UserID, nil
}

func RegisterHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	var registration struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&registration); err != nil {
		w.WriteHeader(422)
		fmt.Fprintf(w, "Error decoding request: %v", err)
		return
	}

	if registration.Name == "" {
		w.WriteHeader(422)
		fmt.Fprintln(w, `Request must include the attribute "name"`)
		return
	}

	if len(registration.Name) > 30 {
		w.WriteHeader(422)
		fmt.Fprintln(w, `"name" must be less than 30 characters`)
		return
	}

	err := ValidatePassword(registration.Password)
	if err != nil {
		w.WriteHeader(422)
		fmt.Fprintln(w, err)
		return
	}

	userID, err := env.userRepo.Create(registration.Name, registration.Email, registration.Password)
	if err != nil {
		if err, ok := err.(DuplicationError); ok {
			w.WriteHeader(422)
			fmt.Fprintf(w, `"%s" is already taken`, err.Field)
			return
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	sessionID, err := env.sessionRepo.Create(Session{UserID: userID})
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	var response struct {
		Name      string `json:"name"`
		SessionID string `json:"sessionID"`
	}

	response.Name = registration.Name
	response.SessionID = hex.EncodeToString(sessionID)

	encoder := json.NewEncoder(w)
	encoder.Encode(response)
}

func CreateSessionHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	var credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&credentials); err != nil {
		w.WriteHeader(422)
		fmt.Fprintf(w, "Error decoding request: %v", err)
		return
	}

	if credentials.Email == "" {
		w.WriteHeader(422)
		fmt.Fprintln(w, `Request must include the attribute "email"`)
		return
	}

	if credentials.Password == "" {
		w.WriteHeader(422)
		fmt.Fprintln(w, `Request must include the attribute "password"`)
		return
	}

	userID, err := env.userRepo.Login(credentials.Email, credentials.Password)
	if err != nil {
		w.WriteHeader(422)
		fmt.Fprintln(w, "Bad email or password")
		return
	}

	sessionID, err := env.sessionRepo.Create(Session{UserID: userID})
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	var response struct {
		Email     string `json:"email"`
		SessionID string `json:"sessionID"`
	}

	response.Email = credentials.Email
	response.SessionID = hex.EncodeToString(sessionID)

	encoder := json.NewEncoder(w)
	encoder.Encode(response)

}

func DeleteSessionHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	sessionID, err := hex.DecodeString(req.FormValue("id"))
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	err = env.sessionRepo.Delete(sessionID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	cookie := &http.Cookie{Name: "sessionId", Value: "logged out", Expires: time.Unix(0, 0)}
	http.SetCookie(w, cookie)
}

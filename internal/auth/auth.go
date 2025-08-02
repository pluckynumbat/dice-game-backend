// Package auth: service which deals with the authenticating the player and managing user sessions
package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"example.com/dice-game-backend/internal/constants"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// session sweeper related constants
const sessionSweepPeriod time.Duration = 6 * time.Hour
const sessionExpirySeconds int64 = 24 * 60 * 60 // 1 day

// Auth Specific Errors:
var serverNilError = fmt.Errorf("provided auth server pointer is nil")
var missingSessionIDError = fmt.Errorf("no session id header in the request")
var invalidSessionError = fmt.Errorf("invalid session in request")

type LoginRequestBody struct {
	IsNewUser     bool   `json:"IsNewUser"`
	ServerVersion string `json:"serverVersion"`
}

type LoginResponse struct {
	PlayerID      string `json:"playerID"`
	ServerVersion string `json:"serverVersion"`
}

type SessionData struct {
	PlayerID       string
	SessionID      string
	LastActionTime int64
}

// Server is the core auth service provider
type Server struct {
	credentials map[string]string

	sessions map[string]*SessionData

	// like a reverse map to the one above it, keyed by player id, values are session ids,
	// used to prevent multiple sessions by the same player
	activePlayerIDs map[string]string

	authMutex sync.Mutex

	serverVersion string
}

// NewAuthServer returns an initialized pointer to the auth server
func NewAuthServer() *Server {
	return &Server{
		credentials:     map[string]string{},
		sessions:        map[string]*SessionData{},
		activePlayerIDs: map[string]string{},

		authMutex: sync.Mutex{},

		serverVersion: strconv.FormatInt(time.Now().UTC().Unix(), 10),
	}
}

// RunAuthServer runs a given auth server on the given port
func (as *Server) RunAuthServer(port string) {

	if as == nil {
		fmt.Println(serverNilError)
	}

	as.StartPeriodicSessionSweep(sessionSweepPeriod, sessionExpirySeconds)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /auth/login", as.HandleLoginRequest)
	mux.HandleFunc("DELETE /auth/logout", as.HandleLogoutRequest)

	mux.HandleFunc("POST /auth/validation-internal", as.HandleValidateRequest)

	addr := constants.CommonHost + ":" + port
	log.Fatal(http.ListenAndServe(addr, mux))
}

// HandleLoginRequest responds with a player id if successful
func (as *Server) HandleLoginRequest(w http.ResponseWriter, r *http.Request) {

	if as == nil {
		http.Error(w, serverNilError.Error(), http.StatusInternalServerError)
		return
	}

	// check if the required header is present
	authHeader := r.Header["Authorization"]
	if authHeader == nil {
		http.Error(w, "received login request without the required header", http.StatusBadRequest)
		return
	}

	// get the username and password from the base 64 encoded data in the auth header
	usr, pwd, err := as.decodeAuthHeaderPayload(authHeader[0])
	if err != nil {
		http.Error(w, "cannot decode the given credentials", http.StatusBadRequest)
		return
	}

	// decode the request
	lrb := &LoginRequestBody{}
	err = json.NewDecoder(r.Body).Decode(lrb)
	if err != nil {
		http.Error(w, "could not decode request body", http.StatusBadRequest)
		return
	}

	// check if it is a new user request VS an existing user request
	// first check the server version, if it does not match with our version,
	// the request will be considered a new user request
	// otherwise, check the 'IsNewUser' flag from the request

	var isNewUser bool
	requestServerVersion := lrb.ServerVersion
	if requestServerVersion != as.serverVersion {
		isNewUser = true
	} else {
		isNewUser = lrb.IsNewUser
	}

	fmt.Printf("received auth login request at: %v , for new user? %v \n", time.Now().UTC(), isNewUser)

	as.authMutex.Lock()
	defer as.authMutex.Unlock()

	if isNewUser {

		// username should not exist in credentials already
		_, exists := as.credentials[usr]
		if exists {
			http.Error(w, "username already exists, cannot create new user", http.StatusBadRequest)
			return
		}

		// add a new entry in the credentials map
		as.credentials[usr] = pwd

	} else {

		// username should exist in credentials already, and passwords should match
		password, ok := as.credentials[usr]
		if !ok || password != pwd {
			http.Error(w, "invalid credentials", http.StatusBadRequest)
			return
		}
	}

	// generate the player id
	pID, err := as.generatePlayerID(usr)
	if err != nil {
		http.Error(w, "could not generate player id", http.StatusInternalServerError)
		return
	}

	// generate a new session id from current unix epoch in microseconds
	sID := strconv.FormatInt(time.Now().UTC().UnixMicro(), 10)

	// check that player id doesn't have an already existing session
	otherSession, exists := as.activePlayerIDs[pID]
	if exists {
		// if they do, delete that session,
		fmt.Printf("found an already existing session for the player id %v, deleting it \n", pID)
		delete(as.sessions, otherSession)
	}

	// add a new entry to the sessions map
	as.sessions[sID] = &SessionData{pID, sID, time.Now().UTC().Unix()}

	// and tie this new session to the player id
	as.activePlayerIDs[pID] = sID

	// provide the session id in the response header
	w.Header().Set("Session-Id", sID)

	w.Header().Set("Content-Type", "application/json")

	// provide the player id and server version in the response body
	err = json.NewEncoder(w).Encode(&LoginResponse{pID, as.serverVersion})
	if err != nil {
		http.Error(w, "could not create response", http.StatusInternalServerError)
		return
	}
}

// HandleLogoutRequest deletes the session if successful
func (as *Server) HandleLogoutRequest(w http.ResponseWriter, r *http.Request) {

	if as == nil {
		http.Error(w, serverNilError.Error(), http.StatusInternalServerError)
		return
	}

	// session based validation
	err := as.ValidateRequest(r)
	if err != nil {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"User Visible Realm\"")
		http.Error(w, "session error: "+err.Error(), http.StatusUnauthorized)
		return
	}

	fmt.Printf("received auth logout request at: %v \n", time.Now().UTC())

	// the above validation guarantees that we have an active session which matches the Session-Id header
	// so we can just delete the required entry
	sIDHeader := r.Header["Session-Id"]
	sID := sIDHeader[0]

	err = as.deleteSession(sID)
	if err != nil {
		http.Error(w, "could not delete session: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = fmt.Fprint(w, "success")
	if err != nil {
		http.Error(w, "could not write response", http.StatusInternalServerError)
		return
	}
}

// decodeAuthHeaderPayload will take the authorization header and return a username and password if successful
// reference: https://en.wikipedia.org/wiki/Basic_access_authentication
func (as *Server) decodeAuthHeaderPayload(encodedCred string) (string, string, error) {

	if as == nil {
		return "", "", serverNilError
	}

	// trim away the first 6 elements which are the prefix 'Basic '
	encodedCred = encodedCred[6:]

	// decode the base64 data
	decodedCred, err := base64.StdEncoding.DecodeString(encodedCred)

	if err != nil {
		return "", "", fmt.Errorf("cannot decode the given credentials")
	}

	// separate the username and password
	decodedStrings := strings.Split(string(decodedCred), ":")

	return decodedStrings[0], decodedStrings[1], nil
}

// generatePlayerID generates a sha 256 hash from the username,
// and returns the first few elements of it as the new player id
func (as *Server) generatePlayerID(input string) (string, error) {

	if input == "" {
		return "", fmt.Errorf("input is empty")
	}

	hash := sha256.New()
	hash.Write([]byte(input))
	hashBytes := hash.Sum(nil)

	resultString := hex.EncodeToString(hashBytes[:4])

	return resultString, nil
}

// ValidateRequest checks for the session id header in other requests, and the validity of the session if present
func (as *Server) ValidateRequest(req *http.Request) error {

	if as == nil {
		return serverNilError
	}

	sessionIdHeader := req.Header["Session-Id"]

	if sessionIdHeader == nil {
		return missingSessionIDError
	}

	// get the session id from the header
	sID := sessionIdHeader[0]

	as.authMutex.Lock()
	defer as.authMutex.Unlock()

	// check for an active session
	activeSession, ok := as.sessions[sID]
	if !ok || sID != activeSession.SessionID {
		return invalidSessionError
	}

	// update the last action time for that session
	as.sessions[sID] = &SessionData{
		activeSession.PlayerID,
		activeSession.SessionID,
		time.Now().UTC().Unix(),
	}

	return nil
}

// HandleValidateRequest is a wrapper around the above method, used when the server is fielding
// internal requests for session validation from other servers
func (as *Server) HandleValidateRequest(w http.ResponseWriter, r *http.Request) {

	if as == nil {
		http.Error(w, serverNilError.Error(), http.StatusInternalServerError)
		return
	}

	err := as.ValidateRequest(r)
	if err != nil {
		http.Error(w, serverNilError.Error(), http.StatusUnauthorized)
		return
	}

	// provide the success response, if the status is 200, validation will be considered to be successful
	_, err = fmt.Fprint(w, "success")
	if err != nil {
		http.Error(w, "could not write response", http.StatusInternalServerError)
		return
	}
}

// deleteSession deletes the session from the session map, and the player ID entry from the active player ID map
func (as *Server) deleteSession(sessionID string) error {

	as.authMutex.Lock()
	defer as.authMutex.Unlock()

	session, ok := as.sessions[sessionID]
	if !ok {
		return invalidSessionError
	}

	delete(as.activePlayerIDs, session.PlayerID) // delete the association between the player id and the session
	delete(as.sessions, sessionID)               // delete the session

	return nil
}

// deleteAllStaleSessions deletes stale sessions based on their last action time
func (as *Server) deleteAllStaleSessions(timeNow time.Time, expirySeconds int64) error {

	unixNow := timeNow.UTC().Unix()

	for sID, session := range as.sessions {

		stale := (unixNow - session.LastActionTime) > expirySeconds

		if stale {
			fmt.Printf("found an old session for player id: %v, deleting it \n", session.PlayerID)
			err := as.deleteSession(sID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// StartPeriodicSessionSweep creates a ticker that will periodically check for stale sessions and delete them
func (as *Server) StartPeriodicSessionSweep(sweepPeriod time.Duration, sessionExpirySeconds int64) {

	if as == nil {
		return
	}

	ticker := time.NewTicker(sweepPeriod)

	go func() {
		for {
			timeNow := <-ticker.C
			fmt.Printf("periodic session sweep tick at %v \n", timeNow.UTC())
			err := as.deleteAllStaleSessions(timeNow, sessionExpirySeconds)
			if err != nil {
				fmt.Printf("error in the periodic session sweep, abort")
				return
			}
		}
	}()
}

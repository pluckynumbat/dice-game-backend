// Package auth: service which deals with the authenticating the player and managing user sessions
package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type LoginRequestBody struct {
	IsNewUser bool `json:"IsNewUser"`
}

type LoginResponse struct {
	PlayerID string `json:"playerID"`
}

type SessionData struct {
	PlayerID       string
	SessionID      string
	LastActionTime int64
}

type Server struct {
	credentials map[string]string
	credMutex   sync.Mutex

	sessions  map[string]*SessionData
	sessMutex sync.Mutex
}

func NewAuthServer() *Server {
	return &Server{
		credentials: map[string]string{},
		credMutex:   sync.Mutex{},

		sessions:  map[string]*SessionData{},
		sessMutex: sync.Mutex{},
	}
}

// HandleLoginRequest responds with a player id (if the player already exists)
func (as *Server) HandleLoginRequest(w http.ResponseWriter, r *http.Request) {

	if as == nil {
		http.Error(w, "provided auth server pointer is nil", http.StatusInternalServerError)
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

	// check if it is a new user request VS an existing user request
	lrb := &LoginRequestBody{}
	err = json.NewDecoder(r.Body).Decode(lrb)
	if err != nil {
		http.Error(w, "could not decode request body", http.StatusBadRequest)
		return
	}

	isNewUser := lrb.IsNewUser
	fmt.Printf("received auth login request at: %v , for new user? %v \n", time.Now().UTC(), isNewUser)

	as.credMutex.Lock()
	defer as.credMutex.Unlock()

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

	as.sessMutex.Lock()
	defer as.sessMutex.Unlock()

	// TODO: handle this differently?
	// check that player id doesn't have an already existing session, and if so, delete it
	for key, val := range as.sessions {
		if val.PlayerID == pID {
			fmt.Printf("found an already existing session for the player id %v, deleting it \n", pID)
			delete(as.sessions, key)
		}
	}

	// add a new entry to the sessions map
	as.sessions[sID] = &SessionData{pID, sID, time.Now().UTC().Unix()}

	// provide the session id in the response header
	w.Header().Set("Session-Id", sID)

	w.Header().Set("Content-Type", "application/json")

	// provide the player id in the response body
	err = json.NewEncoder(w).Encode(&LoginResponse{pID})
	if err != nil {
		http.Error(w, "could not create response", http.StatusInternalServerError)
		return
	}
}

// decodeAuthHeaderPayload will take the authorization header and return a username and password if successful
// reference: https://en.wikipedia.org/wiki/Basic_access_authentication
func (as *Server) decodeAuthHeaderPayload(encodedCred string) (string, string, error) {

	if as == nil {
		return "", "", fmt.Errorf("provided auth server pointer is nil")
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

	sessionIdHeader := req.Header["Session-Id"]

	if sessionIdHeader == nil {
		return fmt.Errorf("no session id header in the request")
	}

	// get the session id from the header
	sID := sessionIdHeader[0]

	as.sessMutex.Lock()
	defer as.sessMutex.Unlock()

	// check for an active session
	activeSession, ok := as.sessions[sID]
	if !ok || sID != activeSession.SessionID {
		return fmt.Errorf("invalid session in request")
	}

	// TODO: also check session expiry and do something about it?

	// update the last action time for that session
	as.sessions[sID] = &SessionData{
		activeSession.PlayerID,
		activeSession.SessionID,
		time.Now().UTC().Unix(),
	}

	return nil
}

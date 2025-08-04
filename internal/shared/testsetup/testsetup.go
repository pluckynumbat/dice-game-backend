// Package testsetup contains common setup functionality used by the different packages in our module
package testsetup

import (
	"bytes"
	"encoding/json"
	"example.com/dice-game-backend/internal/auth"
	"net/http"
	"net/http/httptest"
)

// SetupTestAuth is used to procure a session id that is
// used for validation in the other request handlers
func SetupTestAuth() (*auth.Server, string, error) {
	buf := &bytes.Buffer{}
	reqBody := &auth.LoginRequestBody{IsNewUser: true, ServerVersion: "0"}
	err := json.NewEncoder(buf).Encode(reqBody)
	if err != nil {
		return nil, "", err
	}

	newAuthReq := httptest.NewRequest(http.MethodPost, "/auth/login", buf)
	newAuthReq.SetBasicAuth("user1", "pass1")
	authRespRec := httptest.NewRecorder()

	as := auth.NewServer()
	as.HandleLoginRequest(authRespRec, newAuthReq)
	sID := authRespRec.Header().Get("Session-Id")

	return as, sID, nil
}

// SetupTestAuthWithInput is used to procure a session id from
// the given auth server, using the given credentials
func SetupTestAuthWithInput(as *auth.Server, usr string, pwd string) (string, error) {
	buf := &bytes.Buffer{}
	reqBody := &auth.LoginRequestBody{IsNewUser: false, ServerVersion: "0"}
	err := json.NewEncoder(buf).Encode(reqBody)
	if err != nil {
		return "", err
	}

	newAuthReq := httptest.NewRequest(http.MethodPost, "/auth/login", buf)
	newAuthReq.SetBasicAuth(usr, pwd)
	authRespRec := httptest.NewRecorder()

	as.HandleLoginRequest(authRespRec, newAuthReq)
	sID := authRespRec.Header().Get("Session-Id")

	return sID, nil
}

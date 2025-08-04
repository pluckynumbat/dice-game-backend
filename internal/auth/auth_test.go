package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestNewAuthServer(t *testing.T) {
	authServer := NewServer()

	if authServer == nil {
		t.Fatal("new auth server should not return a nil server pointer")
	}

	if authServer.credentials == nil {
		t.Fatal("new auth server should not contain a nil credentials pointer")
	}

	if authServer.sessions == nil {
		t.Fatal("new auth server should not contain a nil credentials pointer")
	}

	if authServer.activePlayerIDs == nil {
		t.Fatal("new auth server should not contain a nil active player IDs pointer")
	}

	if authServer.serverVersion != strconv.FormatInt(time.Now().UTC().Unix(), 10) {
		t.Error("new auth server's server version should be the current UTC unix timestamp in seconds")
	}
}

func TestServer_HandleLoginRequest(t *testing.T) {

	as := NewServer()

	as.credentials["test2"] = "pass2"
	as.credentials["test3"] = "pass3"

	unixMicroString := strconv.FormatInt(time.Now().UTC().Unix(), 10)
	as.sessions[unixMicroString] = &SessionData{"fd61a03a", unixMicroString, time.Now().UTC().Unix() - 60}
	as.activePlayerIDs["fd61a03a"] = unixMicroString

	tests := []struct {
		name             string
		server           *Server
		addAuthHeader    bool
		username         string
		password         string
		requestBody      *LoginRequestBody
		wantStatus       int
		wantContentType  string
		wantResponseBody *LoginResponse
	}{
		{"nil server", nil, true, "", "", nil, http.StatusInternalServerError, "", nil},
		{"no auth header", as, false, "", "", &LoginRequestBody{IsNewUser: true, ServerVersion: "0"}, http.StatusBadRequest, "", nil},
		{"blank credentials", as, true, "", "", &LoginRequestBody{IsNewUser: true, ServerVersion: "0"}, http.StatusInternalServerError, "", nil},
		{"invalid credentials", as, true, "test0", "pass0", &LoginRequestBody{IsNewUser: false, ServerVersion: as.serverVersion}, http.StatusBadRequest, "", nil},
		{"used credentials", as, true, "test2", "pass2", &LoginRequestBody{IsNewUser: true, ServerVersion: as.serverVersion}, http.StatusBadRequest, "", nil},

		{"new user", as, true, "test1", "pass1", &LoginRequestBody{IsNewUser: true, ServerVersion: "0"}, http.StatusOK, "application/json", &LoginResponse{
			PlayerID:      "1b4f0e98",
			ServerVersion: strconv.FormatInt(time.Now().UTC().Unix(), 10),
		}},
		{"existing user", as, true, "test2", "pass2", &LoginRequestBody{IsNewUser: false, ServerVersion: as.serverVersion}, http.StatusOK, "application/json", &LoginResponse{
			PlayerID:      "60303ae2",
			ServerVersion: strconv.FormatInt(time.Now().UTC().Unix(), 10),
		}},
		{"existing user, existing session", as, true, "test3", "pass3", &LoginRequestBody{IsNewUser: false, ServerVersion: as.serverVersion}, http.StatusOK, "application/json", &LoginResponse{
			PlayerID:      "fd61a03a",
			ServerVersion: strconv.FormatInt(time.Now().UTC().Unix(), 10),
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			buf := &bytes.Buffer{}
			err := json.NewEncoder(buf).Encode(test.requestBody)
			if err != nil {
				t.Fatal("could not encode request body")
			}

			newAuthReq := httptest.NewRequest(http.MethodPost, "/auth/login", buf)
			if test.addAuthHeader {
				newAuthReq.SetBasicAuth(test.username, test.password)
			}
			authRespRec := httptest.NewRecorder()

			authServer := test.server
			authServer.HandleLoginRequest(authRespRec, newAuthReq)

			gotStatus := authRespRec.Result().StatusCode

			if gotStatus != test.wantStatus {
				t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantStatus, gotStatus)
			}

			if gotStatus == http.StatusOK {
				gotContentType := authRespRec.Result().Header.Get("Content-Type")

				if gotContentType != test.wantContentType {
					t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantContentType, gotContentType)
				}

				gotResponseBody := &LoginResponse{}
				err = json.NewDecoder(authRespRec.Result().Body).Decode(gotResponseBody)
				if err != nil {
					t.Fatal("could not decode the response body")
				}

				if !reflect.DeepEqual(gotResponseBody, test.wantResponseBody) {
					t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantResponseBody, gotResponseBody)
				}
			}
		})
	}
}

func TestServer_HandleLogoutRequest(t *testing.T) {

	as, sID, err := setupTestAuth()
	if err != nil {
		t.Fatal("auth setup error: " + err.Error())
	}

	tests := []struct {
		name            string
		server          *Server
		sessionID       string
		wantStatus      int
		wantContentType string
	}{
		{"nil server", nil, "", http.StatusInternalServerError, ""},
		{"blank session id", as, "", http.StatusUnauthorized, "application/json"},
		{"invalid session id", as, "testSessionID", http.StatusUnauthorized, "application/json"},
		{"success", as, sID, http.StatusOK, "text/plain; charset=utf-8"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			newReq := httptest.NewRequest(http.MethodDelete, "/auth/logout/", nil)
			newReq.Header.Set("Session-Id", test.sessionID)
			respRec := httptest.NewRecorder()

			authServer := test.server
			authServer.HandleLogoutRequest(respRec, newReq)

			gotStatus := respRec.Result().StatusCode

			if gotStatus != test.wantStatus {
				t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantStatus, gotStatus)
			}

			if gotStatus == http.StatusOK {
				gotContentType := respRec.Result().Header.Get("Content-Type")

				if gotContentType != test.wantContentType {
					t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantContentType, gotContentType)
				}
			}
		})
	}
}

func TestServer_ValidateRequest(t *testing.T) {

	as := NewServer()
	as.sessions["testsessionid3"] = &SessionData{
		PlayerID:       "",
		SessionID:      "testsessionid3",
		LastActionTime: 0,
	}

	newAuthReq := httptest.NewRequest(http.MethodPost, "/test/", nil)

	newAuthReq2 := httptest.NewRequest(http.MethodPost, "/test/", nil)
	newAuthReq2.Header.Set("Session-Id", "test")

	newAuthReq3 := httptest.NewRequest(http.MethodPost, "/test/", nil)
	newAuthReq3.Header.Set("Session-Id", "testsessionid3")

	tests := []struct {
		name        string
		server      *Server
		httpRequest *http.Request
		expError    error
	}{
		{"nil server", nil, nil, serverNilError},
		{"blank session id", as, newAuthReq, missingSessionIDError},
		{"invalid session", as, newAuthReq2, invalidSessionError},
		{"valid session", as, newAuthReq3, nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			gotErr := test.server.ValidateRequest(test.httpRequest)
			if gotErr != nil {
				if errors.Is(gotErr, test.expError) {
					fmt.Println(gotErr)
				} else {
					t.Fatalf("ValidateRequest() failed with an unexpected error, %v", gotErr)
				}
			}
		})
	}
}

func TestServer_ValidateRequestHandler(t *testing.T) {
	as := NewServer()
	as.sessions["testsessionid3"] = &SessionData{
		PlayerID:       "",
		SessionID:      "testsessionid3",
		LastActionTime: 0,
	}

	newAuthReq := httptest.NewRequest(http.MethodPost, "/auth/validate-internal/", nil)

	newAuthReq2 := httptest.NewRequest(http.MethodPost, "/auth/validate-internal/", nil)
	newAuthReq2.Header.Set("Session-Id", "test")

	newAuthReq3 := httptest.NewRequest(http.MethodPost, "/auth/validate-internal/", nil)
	newAuthReq3.Header.Set("Session-Id", "testsessionid3")

	tests := []struct {
		name            string
		server          *Server
		httpRequest     *http.Request
		wantStatus      int
		wantContentType string
	}{
		{"nil server", nil, nil, http.StatusInternalServerError, ""},
		{"blank session id", as, newAuthReq, http.StatusUnauthorized, ""},
		{"invalid session", as, newAuthReq2, http.StatusUnauthorized, ""},
		{"valid session", as, newAuthReq3, http.StatusOK, "text/plain; charset=utf-8"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			respRec := httptest.NewRecorder()

			authServer := test.server
			authServer.HandleValidateRequest(respRec, test.httpRequest)

			gotStatus := respRec.Result().StatusCode

			if gotStatus != test.wantStatus {
				t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantStatus, gotStatus)
			}

			if gotStatus == http.StatusOK {
				gotContentType := respRec.Result().Header.Get("Content-Type")

				if gotContentType != test.wantContentType {
					t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantContentType, gotContentType)
				}
			}
		})
	}
}

func TestServer_StartPeriodicSessionSweep(t *testing.T) {

	as1 := NewServer()
	as1.sessions["sessionID1"] = &SessionData{
		PlayerID:       "playerID1",
		SessionID:      "sessionID1",
		LastActionTime: time.Now().UTC().Unix() - 10,
	}
	as1.activePlayerIDs["playerID1"] = "sessionID1"

	as2 := NewServer()
	as2.sessions["sessionID2"] = &SessionData{
		PlayerID:       "playerID2",
		SessionID:      "sessionID2",
		LastActionTime: time.Now().UTC().Unix() - 10,
	}
	as2.activePlayerIDs["playerID2"] = "sessionID2"

	tests := []struct {
		name                string
		server              *Server
		period              time.Duration
		expirySeconds       int64
		wantSessions        map[string]*SessionData
		wantActivePlayerIDs map[string]string
	}{
		{"stale session", as1, 25 * time.Millisecond, 5, map[string]*SessionData{}, map[string]string{}},
		{"active session", as2, 25 * time.Millisecond, 20, map[string]*SessionData{"sessionID2": {"playerID2", "sessionID2", time.Now().UTC().Unix() - 10}}, map[string]string{"playerID2": "sessionID2"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.server.StartPeriodicSessionSweep(test.period, test.expirySeconds)
			time.Sleep(test.period + 10*time.Millisecond)

			if !reflect.DeepEqual(test.server.sessions, test.wantSessions) {
				t.Errorf("StartPeriodicSessionSweep() gave incorrect results, want: %v, got: %v", test.wantSessions, test.server.sessions)
			}

			if !reflect.DeepEqual(test.server.activePlayerIDs, test.wantActivePlayerIDs) {
				t.Errorf("StartPeriodicSessionSweep() gave incorrect results, want: %v, got: %v", test.wantActivePlayerIDs, test.server.activePlayerIDs)
			}
		})
	}
}

func setupTestAuth() (*Server, string, error) {
	buf := &bytes.Buffer{}
	reqBody := &LoginRequestBody{IsNewUser: true, ServerVersion: "0"}
	err := json.NewEncoder(buf).Encode(reqBody)
	if err != nil {
		return nil, "", err
	}

	newAuthReq := httptest.NewRequest(http.MethodPost, "/auth/login", buf)
	newAuthReq.SetBasicAuth("user1", "pass1")
	authRespRec := httptest.NewRecorder()

	as := NewServer()
	as.HandleLoginRequest(authRespRec, newAuthReq)
	sID := authRespRec.Header().Get("Session-Id")

	return as, sID, nil
}

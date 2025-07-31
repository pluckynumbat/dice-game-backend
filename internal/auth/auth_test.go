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
	authServer := NewAuthServer()

	if authServer == nil {
		t.Fatal("new profile server should not return a nil server pointer")
	}

	if authServer.credentials == nil {
		t.Fatal("new profile server should not contain a nil credentials pointer")
	}

	if authServer.sessions == nil {
		t.Fatal("new profile server should not contain a nil credentials pointer")
	}

	if authServer.activePlayerIDs == nil {
		t.Fatal("new profile server should not contain a nil active player IDs pointer")
	}

	if authServer.serverVersion != strconv.FormatInt(time.Now().UTC().Unix(), 10) {
		t.Error("new profile server's server version should be the current UTC unix timestamp in seconds")
	}
}

func TestServer_HandleLoginRequest(t *testing.T) {

	as := NewAuthServer()

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

func TestServer_ValidateRequest(t *testing.T) {

	as := NewAuthServer()
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

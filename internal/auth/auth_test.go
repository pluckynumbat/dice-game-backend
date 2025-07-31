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

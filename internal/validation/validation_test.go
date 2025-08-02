package validation

import (
	"bytes"
	"encoding/json"
	"example.com/dice-game-backend/internal/auth"
	"example.com/dice-game-backend/internal/constants"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

var authServer *auth.Server

func TestMain(m *testing.M) {

	authServer = auth.NewAuthServer()
	go authServer.RunAuthServer(constants.AuthServerPort)

	code := m.Run()

	os.Exit(code)
}

func TestValidateRequest(t *testing.T) {

	buf := &bytes.Buffer{}
	reqBody := &auth.LoginRequestBody{IsNewUser: true, ServerVersion: "0"}
	err := json.NewEncoder(buf).Encode(reqBody)
	if err != nil {
		t.Fatal(err)
	}

	newAuthReq := httptest.NewRequest(http.MethodPost, "/auth/login", buf)
	newAuthReq.SetBasicAuth("user1", "pass1")
	authRespRec := httptest.NewRecorder()

	authServer.HandleLoginRequest(authRespRec, newAuthReq)
	sID := authRespRec.Header().Get("Session-Id")

	newReq := httptest.NewRequest(http.MethodPost, "/test/", nil)

	newReq2 := httptest.NewRequest(http.MethodPost, "/test/", nil)
	newReq2.Header.Set("Session-Id", "test")

	newReq3 := httptest.NewRequest(http.MethodPost, "/test/", nil)
	newReq3.Header.Set("Session-Id", sID)

	tests := []struct {
		name        string
		httpRequest *http.Request
		shouldFail  bool
	}{
		{"blank session id", newReq, true},
		{"invalid session", newReq2, true},
		{"valid session", newReq3, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			gotErr := ValidateRequest(test.httpRequest)
			if gotErr != nil && !test.shouldFail {
				t.Fatalf("ValidateRequest() failed with an unexpected error, %v", gotErr)
			} else if gotErr == nil && test.shouldFail {
				t.Fatalf("ValidateRequest() should have failed but it did not")
			}
		})
	}

}

package config

import (
	"example.com/dice-game-backend/internal/auth"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewConfigServer(t *testing.T) {

	configServer := NewConfigServer(auth.NewAuthServer())

	if configServer == nil {
		t.Fatal("new config server should not return a nil server pointer")
	}

	if configServer.gameConfig == nil {
		t.Fatal("new config server should not contain a nil game config")
	}

	if configServer.gameConfig.Levels == nil {
		t.Fatal("new config server should not contain a game config with nil levels")
	}

	if len(configServer.gameConfig.Levels) == 0 {
		t.Fatal("new config server should not contain a game config with empty levels")
	}
}

func TestHandleConfigRequest(t *testing.T) {

	var cs1, cs2 *Server

	buf := &bytes.Buffer{}
	reqBody := &auth.LoginRequestBody{IsNewUser: true, ServerVersion: "0"}
	err := json.NewEncoder(buf).Encode(reqBody)
	if err != nil {
		t.Fatal("could not encode the request body: " + err.Error())
	}

	newAuthReq := httptest.NewRequest(http.MethodPost, "/auth/login", buf)
	newAuthReq.SetBasicAuth("testuser4", "pass4")
	authRespRec := httptest.NewRecorder()

	as := auth.NewAuthServer()
	as.HandleLoginRequest(authRespRec, newAuthReq)
	sID := authRespRec.Header().Get("Session-Id")

	cs2 = NewConfigServer(as)

	tests := []struct {
		name            string
		server          *Server
		sessionID       string
		wantStatus      int
		wantContentType string
	}{
		{"nil server", cs1, "", http.StatusInternalServerError, ""},
		{"valid server, blank session id", cs2, "", http.StatusUnauthorized, "application/json"},
		{"valid server, valid session id", cs2, sID, http.StatusOK, "application/json"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			newReq := httptest.NewRequest(http.MethodGet, "/config/game-config", nil)
			newReq.Header.Set("Session-Id", test.sessionID)
			respRec := httptest.NewRecorder()

			configServer := test.server
			configServer.HandleConfigRequest(respRec, newReq)

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

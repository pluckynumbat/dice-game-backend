package config

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewConfigServer(t *testing.T) {

	configServer := NewConfigServer()

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

	cs2 = NewConfigServer()

	tests := []struct {
		name            string
		server          *Server
		wantStatus      int
		wantContentType string
	}{
		{"nil server", cs1, http.StatusInternalServerError, ""},
		{"valid server", cs2, http.StatusOK, "application/json"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			newReq := httptest.NewRequest(http.MethodGet, "/config/game-config", nil)
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

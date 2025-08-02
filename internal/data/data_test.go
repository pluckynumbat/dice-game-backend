package data

import (
	"encoding/json"
	"example.com/dice-game-backend/internal/profile"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestNewDataServer(t *testing.T) {
	dataServer := NewDataServer()

	if dataServer == nil {
		t.Fatal("new data server should not return a nil server pointer")
	}

	if dataServer.playersDB == nil {
		t.Fatal("new data server should not contain a nil playersDB pointer")
	}

	if dataServer.statsDB == nil {
		t.Fatal("new profile server should not contain a nil statsDB pointer")
	}
}

func TestServer_HandleReadPlayerDataRequest(t *testing.T) {

	ds := NewDataServer()
	ds.playersDB["player2"] = profile.PlayerData{PlayerID: "player2", Level: 1, Energy: 20, LastUpdateTime: time.Now().UTC().Unix()}

	tests := []struct {
		name             string
		server           *Server
		playerID         string
		wantStatus       int
		wantContentType  string
		wantResponseBody *profile.PlayerData
	}{
		{"invalid player", ds, "player1", http.StatusBadRequest, "application/json", nil},
		{name: "existing player", server: ds, playerID: "player2", wantStatus: http.StatusOK, wantContentType: "application/json", wantResponseBody: &profile.PlayerData{PlayerID: "player2", Level: 1, Energy: 20, LastUpdateTime: time.Now().UTC().Unix()}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			newReq := httptest.NewRequest(http.MethodGet, "/data/player-internal/", nil)
			newReq.SetPathValue("id", test.playerID)
			respRec := httptest.NewRecorder()

			dataServer := test.server
			dataServer.HandleReadPlayerDataRequest(respRec, newReq)

			gotStatus := respRec.Result().StatusCode

			if gotStatus != test.wantStatus {
				t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantStatus, gotStatus)
			}

			if gotStatus == http.StatusOK {
				gotContentType := respRec.Result().Header.Get("Content-Type")

				if gotContentType != test.wantContentType {
					t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantContentType, gotContentType)
				}

				gotResponseBody := &profile.PlayerData{}
				err := json.NewDecoder(respRec.Result().Body).Decode(gotResponseBody)
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

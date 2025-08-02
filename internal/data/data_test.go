package data

import (
	"bytes"
	"encoding/json"
	"example.com/dice-game-backend/internal/types"
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
	ds.playersDB["player2"] = types.PlayerData{PlayerID: "player2", Level: 1, Energy: 20, LastUpdateTime: time.Now().UTC().Unix()}

	tests := []struct {
		name             string
		server           *Server
		playerID         string
		wantStatus       int
		wantContentType  string
		wantResponseBody *types.PlayerData
	}{
		{"nil server", nil, "player1", http.StatusInternalServerError, "application/json", nil},
		{"invalid player", ds, "player1", http.StatusBadRequest, "application/json", nil},
		{"existing player", ds, "player2", http.StatusOK, "application/json", &types.PlayerData{PlayerID: "player2", Level: 1, Energy: 20, LastUpdateTime: time.Now().UTC().Unix()}},
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

				gotResponseBody := &types.PlayerData{}
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

func TestServer_HandleWritePlayerDataRequest(t *testing.T) {

	ds := NewDataServer()
	ds.playersDB["player2"] = types.PlayerData{PlayerID: "player2", Level: 1, Energy: 20, LastUpdateTime: time.Now().UTC().Unix()}

	tests := []struct {
		name            string
		server          *Server
		requestPlayer   *types.PlayerData
		wantStatus      int
		wantContentType string
	}{
		{"nil server", nil, nil, http.StatusInternalServerError, "text/plain"},
		{"nil player", ds, nil, http.StatusBadRequest, "text/plain"},
		{"valid player", ds, &types.PlayerData{PlayerID: "player2", Level: 1, Energy: 20, LastUpdateTime: time.Now().UTC().Unix()}, http.StatusOK, "text/plain"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			buf := &bytes.Buffer{}
			reqBody := test.requestPlayer
			err2 := json.NewEncoder(buf).Encode(reqBody)
			if err2 != nil {
				t.Fatal("could not encode the request body: " + err2.Error())
			}

			newReq := httptest.NewRequest(http.MethodPost, "/data/player-internal", buf)
			respRec := httptest.NewRecorder()

			dataServer := test.server
			dataServer.HandleWritePlayerDataRequest(respRec, newReq)

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

func TestServer_HandleReadPlayerStatsRequest(t *testing.T) {

	ds := NewDataServer()

	ds.statsDB["player2"] = types.PlayerStats{
		LevelStats: []types.PlayerLevelStats{
			{1, 2, 3, 1},
			{2, 1, 4, 2},
			{3, 0, 1, 99},
		},
	}

	tests := []struct {
		name             string
		server           *Server
		playerID         string
		wantStatus       int
		wantContentType  string
		wantResponseBody *types.PlayerStats
	}{
		{"nil server", nil, "", http.StatusInternalServerError, "", nil},
		{"new user", ds, "player1", http.StatusBadRequest, "application/json", &types.PlayerStats{}},
		{"existing user", ds, "player2", http.StatusOK, "application/json", &types.PlayerStats{LevelStats: []types.PlayerLevelStats{
			{1, 2, 3, 1},
			{2, 1, 4, 2},
			{3, 0, 1, 99},
		}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			newReq := httptest.NewRequest(http.MethodGet, "/data/stats-internal/", nil)
			newReq.SetPathValue("id", test.playerID)
			respRec := httptest.NewRecorder()

			dataServer := test.server
			dataServer.HandleReadPlayerStatsRequest(respRec, newReq)

			gotStatus := respRec.Result().StatusCode

			if gotStatus != test.wantStatus {
				t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantStatus, gotStatus)
			}

			if gotStatus == http.StatusOK {
				gotContentType := respRec.Result().Header.Get("Content-Type")

				if gotContentType != test.wantContentType {
					t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantContentType, gotContentType)
				}

				gotResponseBody := &types.PlayerStats{}
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

func TestServer_HandleWritePlayerStatsRequest(t *testing.T) {

	ds := NewDataServer()
	ds.statsDB["player2"] = types.PlayerStats{
		LevelStats: []types.PlayerLevelStats{
			{1, 2, 3, 1},
			{2, 1, 4, 2},
			{3, 0, 1, 99},
		},
	}
	tests := []struct {
		name            string
		server          *Server
		requestStats    *types.PlayerStatsWithID
		wantStatus      int
		wantContentType string
	}{
		{"nil server", nil, nil, http.StatusInternalServerError, "text/plain"},
		{"nil player", ds, nil, http.StatusBadRequest, "text/plain"},
		{"new player", ds, &types.PlayerStatsWithID{PlayerID: "player1", PlayerStats: types.PlayerStats{}}, http.StatusOK, "text/plain"},
		{"existing player", ds, &types.PlayerStatsWithID{PlayerID: "player2", PlayerStats: types.PlayerStats{LevelStats: []types.PlayerLevelStats{{1, 2, 3, 1}, {2, 1, 4, 2}, {3, 1, 1, 2}, {4, 0, 1, 99}}}}, http.StatusOK, "text/plain"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			buf := &bytes.Buffer{}
			reqBody := test.requestStats
			err2 := json.NewEncoder(buf).Encode(reqBody)
			if err2 != nil {
				t.Fatal("could not encode the request body: " + err2.Error())
			}

			newReq := httptest.NewRequest(http.MethodPost, "/data/player-internal", buf)
			respRec := httptest.NewRecorder()

			dataServer := test.server
			dataServer.HandleWritePlayerStatsRequest(respRec, newReq)

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

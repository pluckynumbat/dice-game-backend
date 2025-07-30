package stats

import (
	"bytes"
	"encoding/json"
	"errors"
	"example.com/dice-game-backend/internal/auth"
	"example.com/dice-game-backend/internal/config"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestNewStatsServer(t *testing.T) {

	authServer := auth.NewAuthServer()
	statsServer := NewStatsServer(authServer, config.NewConfigServer(authServer).GameConfig)

	if statsServer == nil {
		t.Fatal("new stats server should not return a nil server pointer")
	}

	if statsServer.allStats == nil {
		t.Fatal("new stats server should not contain a nil all stats pointer")
	}
}

func TestServer_ReturnUpdatedPlayerStats(t *testing.T) {

	var s1, s2 *Server

	authServer := auth.NewAuthServer()
	s2 = NewStatsServer(authServer, config.NewConfigServer(authServer).GameConfig)
	s2.allStats["player2"] = PlayerStats{
		nil,
	}
	s2.allStats["player3"] = PlayerStats{
		LevelStats: []PlayerLevelStats{
			{1, 2, 3, 1},
			{2, 1, 4, 2},
			{3, 0, 1, 99},
		},
	}

	tests := []struct {
		name      string
		server    *Server
		playerID  string
		lvlStats  *PlayerLevelStats
		wantStats *PlayerStats
		expError  error
	}{
		{"nil server", s1, "player1", &PlayerLevelStats{}, &PlayerStats{}, serverNilError},
		{"invalid player", s2, "player1", &PlayerLevelStats{5, 1, 0, 4}, nil, playerStatsNotFoundErr{"player1", 5}},
		{"valid new player", s2, "player2", &PlayerLevelStats{1, 0, 1, 99}, &PlayerStats{
			LevelStats: []PlayerLevelStats{
				{1, 0, 1, 99},
			},
		}, nil},
		{"valid existing player", s2, "player3", &PlayerLevelStats{3, 1, 0, 3}, &PlayerStats{
			LevelStats: []PlayerLevelStats{
				{1, 2, 3, 1},
				{2, 1, 4, 2},
				{3, 1, 1, 3},
			},
		}, nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			gotStats, gotErr := test.server.ReturnUpdatedPlayerStats(test.playerID, test.lvlStats)
			if gotErr != nil {
				if errors.Is(gotErr, test.expError) {
					fmt.Println(gotErr)
				} else {
					t.Fatalf("ReturnUpdatedPlayerStats() failed with an unexpected error, %v", gotErr)
				}
			} else {
				if !reflect.DeepEqual(gotStats, test.wantStats) {
					t.Errorf("ReturnUpdatedPlayerStats() gave incorrect results, want: %v, got: %v", test.wantStats, gotStats)
				}
			}
		})
	}
}

func TestServer_HandlePlayerStatsRequest(t *testing.T) {

	var s1, s2 *Server

	as, sID, err := setupTestAuth()
	if err != nil {
		t.Fatal("auth setup error: " + err.Error())
	}

	s2 = NewStatsServer(as, config.NewConfigServer(as).GameConfig)
	s2.allStats["player2"] = PlayerStats{
		LevelStats: []PlayerLevelStats{
			{1, 2, 3, 1},
			{2, 1, 4, 2},
			{3, 0, 1, 99},
		},
	}

	tests := []struct {
		name             string
		server           *Server
		sessionID        string
		playerID         string
		wantStatus       int
		wantContentType  string
		wantResponseBody *PlayerStatsResponse
	}{
		{"nil server", s1, "", "", http.StatusInternalServerError, "", nil},
		{"valid server, blank session id", s2, "", "", http.StatusUnauthorized, "application/json", nil},
		{"valid server, valid session id, new user", s2, sID, "player1", http.StatusOK, "application/json", &PlayerStatsResponse{"player1", PlayerStats{}}},
		{"valid server, valid session id, existing user", s2, sID, "player2", http.StatusOK, "application/json", &PlayerStatsResponse{"player2", PlayerStats{[]PlayerLevelStats{
			{1, 2, 3, 1},
			{2, 1, 4, 2},
			{3, 0, 1, 99},
		}}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			newReq := httptest.NewRequest(http.MethodGet, "/stats/player-stats/", nil)
			newReq.SetPathValue("id", test.playerID)
			newReq.Header.Set("Session-Id", test.sessionID)
			respRec := httptest.NewRecorder()

			statsServer := test.server
			statsServer.HandlePlayerStatsRequest(respRec, newReq)

			gotStatus := respRec.Result().StatusCode

			if gotStatus != test.wantStatus {
				t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantStatus, gotStatus)
			}

			if gotStatus == http.StatusOK {
				gotContentType := respRec.Result().Header.Get("Content-Type")

				if gotContentType != test.wantContentType {
					t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantContentType, gotContentType)
				}

				gotResponseBody := &PlayerStatsResponse{}
				err = json.NewDecoder(respRec.Result().Body).Decode(gotResponseBody)
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

func setupTestAuth() (*auth.Server, string, error) {
	buf := &bytes.Buffer{}
	reqBody := &auth.LoginRequestBody{IsNewUser: true, ServerVersion: "0"}
	err := json.NewEncoder(buf).Encode(reqBody)
	if err != nil {
		return nil, "", err
	}

	newAuthReq := httptest.NewRequest(http.MethodPost, "/auth/login", buf)
	newAuthReq.SetBasicAuth("user1", "pass1")
	authRespRec := httptest.NewRecorder()

	as := auth.NewAuthServer()
	as.HandleLoginRequest(authRespRec, newAuthReq)
	sID := authRespRec.Header().Get("Session-Id")

	return as, sID, nil
}

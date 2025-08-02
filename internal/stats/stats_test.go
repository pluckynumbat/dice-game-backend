package stats

import (
	"encoding/json"
	"errors"
	"example.com/dice-game-backend/internal/auth"
	"example.com/dice-game-backend/internal/config"
	"example.com/dice-game-backend/internal/constants"
	"example.com/dice-game-backend/internal/data"
	"example.com/dice-game-backend/internal/testsetup"
	"example.com/dice-game-backend/internal/types"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
)

func TestMain(m *testing.M) {

	dataServer := data.NewDataServer()
	go dataServer.RunDataServer(constants.DataServerPort)

	code := m.Run()

	os.Exit(code)
}

func TestNewStatsServer(t *testing.T) {

	authServer := auth.NewAuthServer()
	statsServer := NewStatsServer(authServer, config.NewConfigServer(authServer).GameConfig)

	if statsServer == nil {
		t.Fatal("new stats server should not return a nil server pointer")
	}
}

func TestServer_ReturnUpdatedPlayerStats(t *testing.T) {

	var s1, s2 *Server

	authServer := auth.NewAuthServer()
	s2 = NewStatsServer(authServer, config.NewConfigServer(authServer).GameConfig)

	err := s2.writeStatsToDB(&types.PlayerStatsWithID{"player2", types.PlayerStats{nil}})
	if err != nil {
		t.Fatalf("%v \n", err.Error())
	}

	err = s2.writeStatsToDB(&types.PlayerStatsWithID{"player3", types.PlayerStats{[]types.PlayerLevelStats{
		{1, 2, 3, 1},
		{2, 1, 4, 2},
		{3, 0, 1, 99},
	}}})
	if err != nil {
		t.Fatalf("%v \n", err.Error())
	}

	tests := []struct {
		name      string
		server    *Server
		playerID  string
		lvlStats  *types.PlayerLevelStats
		wantStats *types.PlayerStats
		expError  error
	}{
		{"nil server", s1, "player1", &types.PlayerLevelStats{}, &types.PlayerStats{}, serverNilError},
		{"invalid player", s2, "player1", &types.PlayerLevelStats{5, 1, 0, 4}, nil, playerStatsNotFoundErr{"player1", 5}},
		{"valid new player", s2, "player2", &types.PlayerLevelStats{1, 0, 1, 99}, &types.PlayerStats{
			[]types.PlayerLevelStats{
				{1, 0, 1, 99},
			},
		}, nil},
		{"valid existing player", s2, "player3", &types.PlayerLevelStats{3, 1, 0, 3}, &types.PlayerStats{
			[]types.PlayerLevelStats{
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

	as, sID, err := testsetup.SetupTestAuth()
	if err != nil {
		t.Fatal("auth setup error: " + err.Error())
	}

	s2 = NewStatsServer(as, config.NewConfigServer(as).GameConfig)

	err = s2.writeStatsToDB(&types.PlayerStatsWithID{"player2", types.PlayerStats{[]types.PlayerLevelStats{
		{1, 2, 3, 1},
		{2, 1, 4, 2},
		{3, 0, 1, 99},
	}}})
	if err != nil {
		t.Fatalf("%v \n", err.Error())
	}

	tests := []struct {
		name             string
		server           *Server
		sessionID        string
		playerID         string
		wantStatus       int
		wantContentType  string
		wantResponseBody *types.PlayerStatsWithID
	}{
		{"nil server", s1, "", "", http.StatusInternalServerError, "", nil},
		{"valid server, blank session id", s2, "", "", http.StatusUnauthorized, "application/json", nil},
		{"valid server, valid session id, new user", s2, sID, "player1", http.StatusOK, "application/json", &types.PlayerStatsWithID{"player1", types.PlayerStats{}}},
		{"valid server, valid session id, existing user", s2, sID, "player2", http.StatusOK, "application/json", &types.PlayerStatsWithID{"player2", types.PlayerStats{[]types.PlayerLevelStats{
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

				gotResponseBody := &types.PlayerStatsWithID{}
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

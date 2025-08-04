package stats

import (
	"bytes"
	"encoding/json"
	"errors"
	"example.com/dice-game-backend/internal/auth"
	"example.com/dice-game-backend/internal/constants"
	"example.com/dice-game-backend/internal/data"
	"example.com/dice-game-backend/internal/testsetup"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
)

func TestMain(m *testing.M) {

	dataServer := data.NewDataServer()
	go dataServer.Run(constants.DataServerPort)

	code := m.Run()

	os.Exit(code)
}

func TestNewStatsServer(t *testing.T) {

	authServer := auth.NewAuthServer()
	statsServer := NewStatsServer(authServer)

	if statsServer == nil {
		t.Fatal("new stats server should not return a nil server pointer")
	}
}

func TestServer_ReturnUpdatedPlayerStats(t *testing.T) {

	var s1, s2 *Server

	authServer := auth.NewAuthServer()
	s2 = NewStatsServer(authServer)

	err := s2.writeStatsToDB(&data.PlayerStatsWithID{"data", data.PlayerStats{nil}})
	if err != nil {
		t.Fatalf("%v \n", err.Error())
	}

	err = s2.writeStatsToDB(&data.PlayerStatsWithID{"player3", data.PlayerStats{[]data.PlayerLevelStats{
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
		lvlStats  *data.PlayerLevelStats
		wantStats *data.PlayerStats
		expError  error
	}{
		{"nil server", s1, "player1", &data.PlayerLevelStats{}, &data.PlayerStats{}, serverNilError},
		{"invalid player", s2, "player1", &data.PlayerLevelStats{5, 1, 0, 4}, nil, playerStatsNotFoundErr{"player1", 5}},
		{"valid new player", s2, "player2", &data.PlayerLevelStats{1, 0, 1, 99}, &data.PlayerStats{
			[]data.PlayerLevelStats{
				{1, 0, 1, 99},
			},
		}, nil},
		{"valid existing player", s2, "player3", &data.PlayerLevelStats{3, 1, 0, 3}, &data.PlayerStats{
			[]data.PlayerLevelStats{
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

	s2 = NewStatsServer(as)

	err = s2.writeStatsToDB(&data.PlayerStatsWithID{"player2", data.PlayerStats{[]data.PlayerLevelStats{
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
		wantResponseBody *data.PlayerStatsWithID
	}{
		{"nil server", s1, "", "", http.StatusInternalServerError, "", nil},
		{"valid server, blank session id", s2, "", "", http.StatusUnauthorized, "application/json", nil},
		{"valid server, valid session id, new user", s2, sID, "player1", http.StatusOK, "application/json", &data.PlayerStatsWithID{"player1", data.PlayerStats{}}},
		{"valid server, valid session id, existing user", s2, sID, "player2", http.StatusOK, "application/json", &data.PlayerStatsWithID{"player2", data.PlayerStats{[]data.PlayerLevelStats{
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

				gotResponseBody := &data.PlayerStatsWithID{}
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

func TestServer_HandleUpdatePlayerStatsRequest(t *testing.T) {

	s2 := NewStatsServer(auth.NewAuthServer())

	err := s2.writeStatsToDB(&data.PlayerStatsWithID{"player4", data.PlayerStats{nil}})
	if err != nil {
		t.Fatalf("%v \n", err.Error())
	}

	err = s2.writeStatsToDB(&data.PlayerStatsWithID{"player5", data.PlayerStats{[]data.PlayerLevelStats{
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
		playerID         string
		lvlStats         *data.PlayerLevelStats
		wantStatus       int
		wantContentType  string
		wantResponseBody *data.PlayerStats
	}{
		{"nil server", nil, "player1", &data.PlayerLevelStats{}, http.StatusInternalServerError, "", &data.PlayerStats{}},
		{"invalid player", s2, "player1", &data.PlayerLevelStats{5, 1, 0, 4}, http.StatusBadRequest, "", nil},
		{"valid new player", s2, "player4", &data.PlayerLevelStats{1, 0, 1, 99}, http.StatusOK, "application/json", &data.PlayerStats{
			[]data.PlayerLevelStats{
				{1, 0, 1, 99},
			},
		}},
		{"valid existing player", s2, "player5", &data.PlayerLevelStats{3, 1, 0, 3}, http.StatusOK, "application/json", &data.PlayerStats{
			[]data.PlayerLevelStats{
				{1, 2, 3, 1},
				{2, 1, 4, 2},
				{3, 1, 1, 3},
			},
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			buf := &bytes.Buffer{}
			reqBody := &PlayerIDLevelStats{PlayerID: test.playerID, LevelStatsDelta: *test.lvlStats}
			err2 := json.NewEncoder(buf).Encode(reqBody)
			if err2 != nil {
				t.Fatal("could not encode the request body: " + err2.Error())
			}
			newReq := httptest.NewRequest(http.MethodPost, "/stats/player-stats-internal", buf)
			respRec := httptest.NewRecorder()

			statsServer := test.server
			statsServer.HandleUpdatePlayerStatsRequest(respRec, newReq)

			gotStatus := respRec.Result().StatusCode

			if gotStatus != test.wantStatus {
				t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantStatus, gotStatus)
			}

			if gotStatus == http.StatusOK {
				gotContentType := respRec.Result().Header.Get("Content-Type")

				if gotContentType != test.wantContentType {
					t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantContentType, gotContentType)
				}

				gotResponseBody := &data.PlayerStats{}
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

package gameplay

import (
	"bytes"
	"encoding/json"
	"example.com/dice-game-backend/internal/auth"
	"example.com/dice-game-backend/internal/config"
	"example.com/dice-game-backend/internal/data"
	"example.com/dice-game-backend/internal/profile"
	"example.com/dice-game-backend/internal/shared/constants"
	"example.com/dice-game-backend/internal/shared/testsetup"
	"example.com/dice-game-backend/internal/stats"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
)

var authServer *auth.Server
var profileServer *profile.Server

func TestMain(m *testing.M) {

	authServer = auth.NewServer()
	go authServer.Run(constants.AuthServerPort)

	dataServer := data.NewServer()
	go dataServer.Run(constants.DataServerPort)

	profileServer = profile.NewServer(authServer)
	go profileServer.Run(constants.ProfileServerPort)

	statsServer := stats.NewServer(authServer)
	go statsServer.Run(constants.StatsServerPort)

	code := m.Run()

	os.Exit(code)
}

func TestNewGameplayServer(t *testing.T) {

	as := auth.NewServer()

	gs := NewServer(as)

	if gs == nil {
		t.Fatal("new profile server should not return a nil server pointer")
	}
}

func TestServer_HandleEnterLevelRequest(t *testing.T) {

	sID, err := testsetup.SetupTestAuthWithInput(authServer, "user1", "pass1")
	if err != nil {
		t.Fatal("auth setup error: " + err.Error())
	}

	newPlayerData, err := setupTestProfile("player2", sID, profileServer)
	if err != nil {
		t.Fatal("profile setup error: " + err.Error())
	}
	energyCost := config.Config.Levels[0].EnergyCost

	gs := NewServer(authServer)

	tests := []struct {
		name             string
		server           *Server
		sessionID        string
		requestBody      *EnterLevelRequestBody
		wantStatus       int
		wantContentType  string
		wantResponseBody *EnterLevelResponse
	}{
		{"nil server", nil, "", nil, http.StatusInternalServerError, "", nil},
		{"blank session id", gs, "", nil, http.StatusUnauthorized, "application/json", nil},
		{"invalid session id", gs, "testSessionID", nil, http.StatusUnauthorized, "application/json", nil},
		{"invalid player", gs, sID, &EnterLevelRequestBody{"player1", 1}, http.StatusInternalServerError, "application/json", nil},
		{"invalid level 0", gs, sID, &EnterLevelRequestBody{"player2", 0}, http.StatusBadRequest, "application/json", nil},
		{"invalid level 50", gs, sID, &EnterLevelRequestBody{"player2", 50}, http.StatusBadRequest, "application/json", nil},
		{"locked level", gs, sID, &EnterLevelRequestBody{"player2", 5}, http.StatusOK, "application/json", &EnterLevelResponse{false, *newPlayerData}},
		{name: "valid level", server: gs, sessionID: sID, requestBody: &EnterLevelRequestBody{"player2", 1}, wantStatus: http.StatusOK, wantContentType: "application/json", wantResponseBody: &EnterLevelResponse{
			AccessGranted: true,
			Player: data.PlayerData{
				PlayerID:       newPlayerData.PlayerID,
				Level:          newPlayerData.Level,
				Energy:         newPlayerData.Energy - energyCost,
				LastUpdateTime: newPlayerData.LastUpdateTime,
			}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			buf := &bytes.Buffer{}
			err2 := json.NewEncoder(buf).Encode(test.requestBody)
			if err2 != nil {
				t.Fatal("could not encode the request body: " + err2.Error())
			}

			newReq := httptest.NewRequest(http.MethodPost, "/gameplay/entry/", buf)
			newReq.Header.Set("Session-Id", test.sessionID)
			respRec := httptest.NewRecorder()

			gameplayServer := test.server
			gameplayServer.HandleEnterLevelRequest(respRec, newReq)

			gotStatus := respRec.Result().StatusCode

			if gotStatus != test.wantStatus {
				t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantStatus, gotStatus)
			}

			if gotStatus == http.StatusOK {
				gotContentType := respRec.Result().Header.Get("Content-Type")

				if gotContentType != test.wantContentType {
					t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantContentType, gotContentType)
				}

				gotResponseBody := &EnterLevelResponse{}
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

func TestServer_HandleLevelResultRequest(t *testing.T) {

	sID, err := testsetup.SetupTestAuthWithInput(authServer, "user2", "pass2")
	if err != nil {
		t.Fatal("auth setup error: " + err.Error())
	}

	newPlayer3, err := setupTestProfile("player3", sID, profileServer)
	if err != nil {
		t.Fatal("profile setup error: " + err.Error())
	}

	energyReward := config.Config.Levels[0].EnergyReward

	gs := NewServer(authServer)

	tests := []struct {
		name             string
		server           *Server
		sessionID        string
		requestBody      *LevelResultRequestBody
		wantStatus       int
		wantContentType  string
		wantResponseBody *LevelResultResponse
	}{
		{"nil server", nil, "", nil, http.StatusInternalServerError, "", nil},
		{"blank session id", gs, "", nil, http.StatusUnauthorized, "application/json", nil},
		{"invalid session id", gs, "testSessionID", nil, http.StatusUnauthorized, "application/json", nil},
		{"invalid player", gs, sID, &LevelResultRequestBody{"player1", 1, nil}, http.StatusInternalServerError, "application/json", nil},
		{"invalid level 0", gs, sID, &LevelResultRequestBody{"player3", 0, nil}, http.StatusBadRequest, "application/json", nil},
		{"invalid level 50", gs, sID, &LevelResultRequestBody{"player3", 50, nil}, http.StatusBadRequest, "application/json", nil},
		{"locked level", gs, sID, &LevelResultRequestBody{"player3", 5, nil}, http.StatusBadRequest, "application/json", &LevelResultResponse{}},
		{"nil rolls", gs, sID, &LevelResultRequestBody{"player3", 5, nil}, http.StatusBadRequest, "application/json", &LevelResultResponse{}},
		{"invalid rolls", gs, sID, &LevelResultRequestBody{"player3", 1, []int32{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}}, http.StatusBadRequest, "application/json", &LevelResultResponse{}},

		{name: "level loss", server: gs, sessionID: sID, requestBody: &LevelResultRequestBody{"player3", 1, []int32{1, 1}}, wantStatus: http.StatusOK, wantContentType: "application/json", wantResponseBody: &LevelResultResponse{
			LevelResult: LevelResult{false, 0, false},
			Player:      *newPlayer3,
			Stats:       data.PlayerStats{LevelStats: []data.PlayerLevelStats{{1, 0, 1, 99}}},
		}},
		{name: "level win", server: gs, sessionID: sID, requestBody: &LevelResultRequestBody{"player3", 1, []int32{1, 6}}, wantStatus: http.StatusOK, wantContentType: "application/json", wantResponseBody: &LevelResultResponse{
			LevelResult: LevelResult{true, energyReward, true},
			Player:      data.PlayerData{PlayerID: newPlayer3.PlayerID, Level: newPlayer3.Level + 1, Energy: 50, LastUpdateTime: newPlayer3.LastUpdateTime},
			Stats:       data.PlayerStats{LevelStats: []data.PlayerLevelStats{{1, 1, 1, 2}}},
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			buf := &bytes.Buffer{}
			err2 := json.NewEncoder(buf).Encode(test.requestBody)
			if err2 != nil {
				t.Fatal("could not encode the request body: " + err2.Error())
			}

			newReq := httptest.NewRequest(http.MethodPost, "/gameplay/result/", buf)
			newReq.Header.Set("Session-Id", test.sessionID)
			respRec := httptest.NewRecorder()

			gameplayServer := test.server
			gameplayServer.HandleLevelResultRequest(respRec, newReq)

			gotStatus := respRec.Result().StatusCode

			if gotStatus != test.wantStatus {
				t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantStatus, gotStatus)
			}

			if gotStatus == http.StatusOK {
				gotContentType := respRec.Result().Header.Get("Content-Type")

				if gotContentType != test.wantContentType {
					t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantContentType, gotContentType)
				}

				gotResponseBody := &LevelResultResponse{}
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

func setupTestProfile(playerID string, sessionID string, profileServer *profile.Server) (*data.PlayerData, error) {
	buf := &bytes.Buffer{}
	reqBody := &profile.NewPlayerRequestBody{PlayerID: playerID}
	err := json.NewEncoder(buf).Encode(reqBody)
	if err != nil {
		return nil, err
	}

	newReq := httptest.NewRequest(http.MethodPost, "/profile/new-player", buf)
	newReq.Header.Set("Session-Id", sessionID)
	respRec := httptest.NewRecorder()

	profileServer.HandleNewPlayerRequest(respRec, newReq)

	newPlayerData := &data.PlayerData{}
	err = json.NewDecoder(respRec.Result().Body).Decode(newPlayerData)
	if err != nil {
		return nil, err
	}

	return newPlayerData, nil
}

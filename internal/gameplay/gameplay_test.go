package gameplay

import (
	"bytes"
	"encoding/json"
	"example.com/dice-game-backend/internal/auth"
	"example.com/dice-game-backend/internal/config"
	"example.com/dice-game-backend/internal/profile"
	"example.com/dice-game-backend/internal/stats"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestNewGameplayServer(t *testing.T) {

	as := auth.NewAuthServer()
	cs := config.NewConfigServer(as)
	ps := profile.NewProfileServer(as, cs.GameConfig)
	ss := stats.NewStatsServer(as, cs.GameConfig)

	gs := NewGameplayServer(as, ps, ss, cs.GameConfig)

	if gs == nil {
		t.Fatal("new profile server should not return a nil server pointer")
	}
}

func TestServer_HandleEnterLevelRequest(t *testing.T) {

	as, sID, err := setupTestAuth()
	if err != nil {
		t.Fatal("auth setup error: " + err.Error())
	}

	cs := config.NewConfigServer(as)
	ps := profile.NewProfileServer(as, cs.GameConfig)

	newPlayerData, err := setupTestProfile("player2", sID, ps)
	if err != nil {
		t.Fatal("profile setup error: " + err.Error())
	}
	energyCost := cs.GameConfig.Levels[0].EnergyCost

	ss := stats.NewStatsServer(as, cs.GameConfig)

	gs := NewGameplayServer(as, ps, ss, cs.GameConfig)

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
		{"invalid player", gs, sID, &EnterLevelRequestBody{"player1", 1}, http.StatusBadRequest, "application/json", nil},
		{"invalid level", gs, sID, &EnterLevelRequestBody{"player2", 50}, http.StatusBadRequest, "application/json", nil},
		{"locked level", gs, sID, &EnterLevelRequestBody{"player2", 5}, http.StatusOK, "application/json", &EnterLevelResponse{false, *newPlayerData}},
		{"valid level", gs, sID, &EnterLevelRequestBody{"player2", 1}, http.StatusOK, "application/json", &EnterLevelResponse{true, profile.PlayerData{newPlayerData.PlayerID, newPlayerData.Level, newPlayerData.Energy - energyCost, newPlayerData.LastUpdateTime}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			buf := &bytes.Buffer{}
			err2 := json.NewEncoder(buf).Encode(test.requestBody)
			if err2 != nil {
				t.Fatal("could not encode the request body: " + err2.Error())
			}

			newReq := httptest.NewRequest(http.MethodPost, "/profile/new-player/", buf)
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

func setupTestProfile(playerID string, sessionID string, profileServer *profile.Server) (*profile.PlayerData, error) {
	buf := &bytes.Buffer{}
	reqBody := &profile.NewPlayerRequestBody{PlayerID: playerID}
	err := json.NewEncoder(buf).Encode(reqBody)
	if err != nil {
		return nil, err
	}

	newReq := httptest.NewRequest(http.MethodPost, "/profile/new-player/", buf)
	newReq.Header.Set("Session-Id", sessionID)
	respRec := httptest.NewRecorder()

	profileServer.HandleNewPlayerRequest(respRec, newReq)

	newPlayerData := &profile.PlayerData{}
	err = json.NewDecoder(respRec.Result().Body).Decode(newPlayerData)
	if err != nil {
		return nil, err
	}

	return newPlayerData, nil
}
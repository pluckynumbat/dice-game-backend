package profile

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
	"time"
)

func TestNewProfileServer(t *testing.T) {

	authServer := auth.NewAuthServer()
	profileServer := NewProfileServer(authServer, config.NewConfigServer(authServer).GameConfig)

	if profileServer == nil {
		t.Fatal("new profile server should not return a nil server pointer")
	}

	if profileServer.players == nil {
		t.Fatal("new profile server should not contain a nil players pointer")
	}
}

func TestServer_GetPlayer(t *testing.T) {

	authServer := auth.NewAuthServer()
	ps := NewProfileServer(authServer, config.NewConfigServer(authServer).GameConfig)
	ps.players["player2"] = PlayerData{"player2", 1, 50, time.Now().UTC().Unix()}
	ps.players["player3"] = PlayerData{"player3", 1, 20, time.Now().UTC().Unix() - 100}

	tests := []struct {
		name       string
		server     *Server
		playerID   string
		wantPlayer *PlayerData
		expError   error
	}{
		{"nil server", nil, "", nil, serverNilError},
		{"invalid player", ps, "player1", nil, playerNotFoundErr{"player1"}},
		{"valid player", ps, "player2", &PlayerData{"player2", 1, 50, time.Now().UTC().Unix()}, nil},
		{"valid player, restore energy", ps, "player2", &PlayerData{"player2", 1, 50, time.Now().UTC().Unix()}, nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			gotPlayer, gotErr := test.server.GetPlayer(test.playerID)
			if gotErr != nil {
				if errors.Is(gotErr, test.expError) {
					fmt.Println(gotErr)
				} else {
					t.Fatalf("GetPlayer() failed with an unexpected error, %v", gotErr)
				}
			} else {
				if !reflect.DeepEqual(gotPlayer, test.wantPlayer) {
					t.Errorf("GetPlayer() gave incorrect results, want: %v, got: %v", test.wantPlayer, gotPlayer)
				}
			}
		})
	}
}

func TestServer_UpdatePlayerData(t *testing.T) {

	authServer := auth.NewAuthServer()
	ps := NewProfileServer(authServer, config.NewConfigServer(authServer).GameConfig)
	ps.players["player2"] = PlayerData{"player2", 1, 20, time.Now().UTC().Unix()}
	ps.players["player3"] = PlayerData{"player3", 2, 20, time.Now().UTC().Unix()}
	ps.players["player4"] = PlayerData{"player4", 10, 50, time.Now().UTC().Unix()}

	tests := []struct {
		name        string
		server      *Server
		playerID    string
		energyDelta int32
		newLevel    int32
		wantPlayer  *PlayerData
		expError    error
	}{
		{"nil server", nil, "", 0, 0, nil, serverNilError},
		{"invalid player", ps, "player1", 0, 0, nil, playerNotFoundErr{"player1"}},
		{"valid player, more energy", ps, "player2", 20, 1, &PlayerData{"player2", 1, 40, time.Now().UTC().Unix()}, nil},
		{"valid player, new level", ps, "player3", 10, 3, &PlayerData{"player3", 3, 30, time.Now().UTC().Unix()}, nil},
		{"valid player, max energy, max level, ", ps, "player4", 100, 100, &PlayerData{"player4", 10, 50, time.Now().UTC().Unix()}, nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			gotPlayer, gotErr := test.server.UpdatePlayerData(test.playerID, test.energyDelta, test.newLevel)
			if gotErr != nil {
				if errors.Is(gotErr, test.expError) {
					fmt.Println(gotErr)
				} else {
					t.Fatalf("UpdatePlayerData() failed with an unexpected error, %v", gotErr)
				}
			} else {
				if !reflect.DeepEqual(gotPlayer, test.wantPlayer) {
					t.Errorf("UpdatePlayerData() gave incorrect results, want: %v, got: %v", test.wantPlayer, gotPlayer)
				}
			}
		})
	}
}

func TestServer_HandleNewPlayerRequest(t *testing.T) {

	as, sID, err := setupTestAuth()
	if err != nil {
		t.Fatal("auth setup error: " + err.Error())
	}

	ps := NewProfileServer(as, config.NewConfigServer(as).GameConfig)
	ps.players["player2"] = PlayerData{"player2", 1, 20, time.Now().UTC().Unix()}

	tests := []struct {
		name             string
		server           *Server
		sessionID        string
		playerID         string
		wantStatus       int
		wantContentType  string
		wantResponseBody *PlayerData
	}{
		{"nil server", nil, "", "", http.StatusInternalServerError, "", nil},
		{"blank session id", ps, "", "", http.StatusUnauthorized, "application/json", nil},
		{"invalid session id", ps, "testSessionID", "", http.StatusUnauthorized, "application/json", nil},
		{"new player", ps, sID, "player1", http.StatusOK, "application/json", &PlayerData{"player1", 1, 50, time.Now().UTC().Unix()}},
		{"existing player", ps, sID, "player2", http.StatusBadRequest, "application/json", nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			buf := &bytes.Buffer{}
			reqBody := &NewPlayerRequestBody{test.playerID}
			err2 := json.NewEncoder(buf).Encode(reqBody)
			if err2 != nil {
				t.Fatal("could not encode the request body: " + err2.Error())
			}

			newReq := httptest.NewRequest(http.MethodPost, "/profile/new-player/", buf)
			newReq.Header.Set("Session-Id", test.sessionID)
			respRec := httptest.NewRecorder()

			statsServer := test.server
			statsServer.HandleNewPlayerRequest(respRec, newReq)

			gotStatus := respRec.Result().StatusCode

			if gotStatus != test.wantStatus {
				t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantStatus, gotStatus)
			}

			if gotStatus == http.StatusOK {
				gotContentType := respRec.Result().Header.Get("Content-Type")

				if gotContentType != test.wantContentType {
					t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantContentType, gotContentType)
				}

				gotResponseBody := &PlayerData{}
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

func TestServer_HandlePlayerDataRequest(t *testing.T) {

	as, sID, err := setupTestAuth()
	if err != nil {
		t.Fatal("auth setup error: " + err.Error())
	}

	ps := NewProfileServer(as, config.NewConfigServer(as).GameConfig)
	ps.players["player2"] = PlayerData{"player2", 1, 20, time.Now().UTC().Unix()}

	tests := []struct {
		name             string
		server           *Server
		sessionID        string
		playerID         string
		wantStatus       int
		wantContentType  string
		wantResponseBody *PlayerData
	}{
		{"nil server", nil, "", "", http.StatusInternalServerError, "", nil},
		{"blank session id", ps, "", "", http.StatusUnauthorized, "application/json", nil},
		{"invalid session id", ps, "testSessionID", "", http.StatusUnauthorized, "application/json", nil},
		{"new player", ps, sID, "player1", http.StatusBadRequest, "application/json", nil},
		{"existing player", ps, sID, "player2", http.StatusOK, "application/json", &PlayerData{"player2", 1, 20, time.Now().UTC().Unix()}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			newReq := httptest.NewRequest(http.MethodGet, "/profile/player-data/", nil)
			newReq.SetPathValue("id", test.playerID)
			newReq.Header.Set("Session-Id", test.sessionID)
			respRec := httptest.NewRecorder()

			statsServer := test.server
			statsServer.HandlePlayerDataRequest(respRec, newReq)

			gotStatus := respRec.Result().StatusCode

			if gotStatus != test.wantStatus {
				t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantStatus, gotStatus)
			}

			if gotStatus == http.StatusOK {
				gotContentType := respRec.Result().Header.Get("Content-Type")

				if gotContentType != test.wantContentType {
					t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantContentType, gotContentType)
				}

				gotResponseBody := &PlayerData{}
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

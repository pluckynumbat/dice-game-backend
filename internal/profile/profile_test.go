package profile

import (
	"bytes"
	"encoding/json"
	"errors"
	"example.com/dice-game-backend/internal/auth"
	"example.com/dice-game-backend/internal/data"
	"example.com/dice-game-backend/internal/shared/constants"
	"example.com/dice-game-backend/internal/shared/testsetup"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestMain(m *testing.M) {

	dataServer := data.NewDataServer()
	go dataServer.Run(constants.DataServerPort)

	code := m.Run()

	os.Exit(code)
}

func TestNewProfileServer(t *testing.T) {

	authServer := auth.NewServer()
	profileServer := NewProfileServer(authServer)

	if profileServer == nil {
		t.Fatal("new profile server should not return a nil server pointer")
	}
}

func TestServer_GetPlayer(t *testing.T) {

	authServer := auth.NewServer()
	ps := NewProfileServer(authServer)

	err := ps.writePlayerToDB(&data.PlayerData{"player2", 1, 50, time.Now().UTC().Unix()})
	if err != nil {
		t.Fatalf("%v \n", err.Error())
	}

	tests := []struct {
		name       string
		server     *Server
		playerID   string
		wantPlayer *data.PlayerData
		expError   error
	}{
		{"nil server", nil, "", nil, serverNilError},
		{"invalid player", ps, "player1", nil, playerNotFoundErr{"player1"}},
		{"valid player", ps, "player2", &data.PlayerData{"player2", 1, 50, time.Now().UTC().Unix()}, nil},
		{"valid player, restore energy", ps, "player2", &data.PlayerData{"player2", 1, 50, time.Now().UTC().Unix()}, nil},
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

	authServer := auth.NewServer()
	ps := NewProfileServer(authServer)

	err := ps.writePlayerToDB(&data.PlayerData{"player2", 1, 20, time.Now().UTC().Unix()})
	if err != nil {
		t.Fatalf("%v \n", err.Error())
	}
	err = ps.writePlayerToDB(&data.PlayerData{"player3", 2, 20, time.Now().UTC().Unix()})
	if err != nil {
		t.Fatalf("%v \n", err.Error())
	}
	err = ps.writePlayerToDB(&data.PlayerData{"player4", 10, 50, time.Now().UTC().Unix()})
	if err != nil {
		t.Fatalf("%v \n", err.Error())
	}

	tests := []struct {
		name        string
		server      *Server
		playerID    string
		energyDelta int32
		newLevel    int32
		wantPlayer  *data.PlayerData
		expError    error
	}{
		{"nil server", nil, "", 0, 0, nil, serverNilError},
		{"invalid player", ps, "player1", 0, 0, nil, playerNotFoundErr{"player1"}},
		{"valid player, more energy", ps, "player2", 20, 1, &data.PlayerData{"player2", 1, 40, time.Now().UTC().Unix()}, nil},
		{"valid player, new level", ps, "player3", 10, 3, &data.PlayerData{"player3", 3, 30, time.Now().UTC().Unix()}, nil},
		{"valid player, max energy, max level, ", ps, "player4", 100, 100, &data.PlayerData{"player4", 10, 50, time.Now().UTC().Unix()}, nil},
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

	as, sID, err := testsetup.SetupTestAuth()
	if err != nil {
		t.Fatal("auth setup error: " + err.Error())
	}

	ps := NewProfileServer(as)

	err = ps.writePlayerToDB(&data.PlayerData{"player2", 1, 20, time.Now().UTC().Unix()})
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
		wantResponseBody *data.PlayerData
	}{
		{"nil server", nil, "", "", http.StatusInternalServerError, "", nil},
		{"blank session id", ps, "", "", http.StatusUnauthorized, "application/json", nil},
		{"invalid session id", ps, "testSessionID", "", http.StatusUnauthorized, "application/json", nil},
		{"new player", ps, sID, "player1", http.StatusOK, "application/json", &data.PlayerData{"player1", 1, 50, time.Now().UTC().Unix()}},
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

			newReq := httptest.NewRequest(http.MethodPost, "/profile/new-player", buf)
			newReq.Header.Set("Session-Id", test.sessionID)
			respRec := httptest.NewRecorder()

			profileServer := test.server
			profileServer.HandleNewPlayerRequest(respRec, newReq)

			gotStatus := respRec.Result().StatusCode

			if gotStatus != test.wantStatus {
				t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantStatus, gotStatus)
			}

			if gotStatus == http.StatusOK {
				gotContentType := respRec.Result().Header.Get("Content-Type")

				if gotContentType != test.wantContentType {
					t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantContentType, gotContentType)
				}

				gotResponseBody := &data.PlayerData{}
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

	as, sID, err := testsetup.SetupTestAuth()
	if err != nil {
		t.Fatal("auth setup error: " + err.Error())
	}

	ps := NewProfileServer(as)

	err = ps.writePlayerToDB(&data.PlayerData{"player2", 1, 20, time.Now().UTC().Unix()})
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
		wantResponseBody *data.PlayerData
	}{
		{"nil server", nil, "", "", http.StatusInternalServerError, "", nil},
		{"blank session id", ps, "", "", http.StatusUnauthorized, "application/json", nil},
		{"invalid session id", ps, "testSessionID", "", http.StatusUnauthorized, "application/json", nil},
		{"new player", ps, sID, "player5", http.StatusBadRequest, "application/json", nil},
		{"existing player", ps, sID, "player2", http.StatusOK, "application/json", &data.PlayerData{"player2", 1, 20, time.Now().UTC().Unix()}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			newReq := httptest.NewRequest(http.MethodGet, "/profile/player-data/", nil)
			newReq.SetPathValue("id", test.playerID)
			newReq.Header.Set("Session-Id", test.sessionID)
			respRec := httptest.NewRecorder()

			profileServer := test.server
			profileServer.HandlePlayerDataRequest(respRec, newReq)

			gotStatus := respRec.Result().StatusCode

			if gotStatus != test.wantStatus {
				t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantStatus, gotStatus)
			}

			if gotStatus == http.StatusOK {
				gotContentType := respRec.Result().Header.Get("Content-Type")

				if gotContentType != test.wantContentType {
					t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantContentType, gotContentType)
				}

				gotResponseBody := &data.PlayerData{}
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

func TestServer_HandleUpdatePlayerRequest(t *testing.T) {

	authServer := auth.NewServer()
	ps := NewProfileServer(authServer)

	err := ps.writePlayerToDB(&data.PlayerData{"player8", 1, 20, time.Now().UTC().Unix()})
	if err != nil {
		t.Fatalf("%v \n", err.Error())
	}
	err = ps.writePlayerToDB(&data.PlayerData{"player9", 2, 20, time.Now().UTC().Unix()})
	if err != nil {
		t.Fatalf("%v \n", err.Error())
	}
	err = ps.writePlayerToDB(&data.PlayerData{"player10", 10, 50, time.Now().UTC().Unix()})
	if err != nil {
		t.Fatalf("%v \n", err.Error())
	}

	tests := []struct {
		name             string
		server           *Server
		playerID         string
		energyDelta      int32
		newLevel         int32
		wantStatus       int
		wantContentType  string
		wantResponseBody *data.PlayerData
	}{
		{"nil server", nil, "", 0, 0, http.StatusInternalServerError, "", nil},
		{"invalid player", ps, "player7", 0, 0, http.StatusBadRequest, "", nil},
		{"valid player, more energy", ps, "player8", 20, 1, http.StatusOK, "application/json", &data.PlayerData{"player8", 1, 40, time.Now().UTC().Unix()}},
		{"valid player, new level", ps, "player9", 10, 3, http.StatusOK, "application/json", &data.PlayerData{"player9", 3, 30, time.Now().UTC().Unix()}},
		{"valid player, max energy, max level, ", ps, "player10", 100, 100, http.StatusOK, "application/json", &data.PlayerData{"player10", 10, 50, time.Now().UTC().Unix()}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			buf := &bytes.Buffer{}
			reqBody := &PlayerIDLevelEnergy{test.playerID, test.newLevel, test.energyDelta}
			err2 := json.NewEncoder(buf).Encode(reqBody)
			if err2 != nil {
				t.Fatal("could not encode the request body: " + err2.Error())
			}

			newReq := httptest.NewRequest(http.MethodPut, "/profile/player-data-internal", buf)
			respRec := httptest.NewRecorder()

			profileServer := test.server
			profileServer.HandleUpdatePlayerRequest(respRec, newReq)

			gotStatus := respRec.Result().StatusCode

			if gotStatus != test.wantStatus {
				t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantStatus, gotStatus)
			}

			if gotStatus == http.StatusOK {
				gotContentType := respRec.Result().Header.Get("Content-Type")

				if gotContentType != test.wantContentType {
					t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantContentType, gotContentType)
				}

				gotResponseBody := &data.PlayerData{}
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

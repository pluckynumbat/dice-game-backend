package config

import (
	"encoding/json"
	"example.com/dice-game-backend/internal/auth"
	"example.com/dice-game-backend/internal/shared/testsetup"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestNewConfigServer(t *testing.T) {

	configServer := NewServer(auth.NewServer())

	if configServer == nil {
		t.Fatal("new config server should not return a nil server pointer")
	}
}

func TestBasicConfigValidation(t *testing.T) {
	if Config == nil {
		t.Fatal("config should not contain a nil game config")
	}

	if Config.Levels == nil {
		t.Fatal("config should not contain a game config with nil levels")
	}

	if len(Config.Levels) == 0 {
		t.Fatal("config should not contain a game config with empty levels")
	}
}

func TestHandleConfigRequest(t *testing.T) {

	var cs1, cs2 *Server

	as, sID, err := testsetup.SetupTestAuth()
	if err != nil {
		t.Fatal("auth setup error: " + err.Error())
	}

	cs2 = NewServer(as)

	tests := []struct {
		name            string
		server          *Server
		sessionID       string
		wantStatus      int
		wantContentType string
		wantResponse    *GameConfig
	}{
		{"nil server", cs1, "", http.StatusInternalServerError, "", nil},
		{"valid server, blank session id", cs2, "", http.StatusUnauthorized, "application/json", nil},
		{"valid server, valid session id", cs2, sID, http.StatusOK, "application/json", &GameConfig{
			Levels: []LevelConfig{
				{Level: 1, EnergyCost: 3, TotalRolls: 2, Target: 6, EnergyReward: 5},
				{Level: 2, EnergyCost: 3, TotalRolls: 3, Target: 4, EnergyReward: 5},
				{Level: 3, EnergyCost: 4, TotalRolls: 4, Target: 2, EnergyReward: 6},
				{Level: 4, EnergyCost: 4, TotalRolls: 3, Target: 1, EnergyReward: 6},
				{Level: 5, EnergyCost: 4, TotalRolls: 2, Target: 5, EnergyReward: 6},
				{Level: 6, EnergyCost: 5, TotalRolls: 4, Target: 3, EnergyReward: 7},
				{Level: 7, EnergyCost: 5, TotalRolls: 3, Target: 4, EnergyReward: 7},
				{Level: 8, EnergyCost: 5, TotalRolls: 2, Target: 1, EnergyReward: 7},
				{Level: 9, EnergyCost: 6, TotalRolls: 4, Target: 2, EnergyReward: 8},
				{Level: 10, EnergyCost: 6, TotalRolls: 3, Target: 6, EnergyReward: 8},
			},
			DefaultLevel:       1,
			MaxEnergy:          50,
			EnergyRegenSeconds: 5,
			DefaultLevelScore:  99,
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			newReq := httptest.NewRequest(http.MethodGet, "/config/game-config", nil)
			newReq.Header.Set("Session-Id", test.sessionID)
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

				gotResponseBody := &GameConfig{}
				err = json.NewDecoder(respRec.Result().Body).Decode(gotResponseBody)
				if err != nil {
					t.Fatal("could not decode the response body")
				}

				if !reflect.DeepEqual(gotResponseBody, test.wantResponse) {
					t.Errorf("handler gave incorrect results, want: %v, got: %v", test.wantResponse, gotResponseBody)
				}
			}
		})
	}
}

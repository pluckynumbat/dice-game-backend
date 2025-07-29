// Package config: service which deals with the game config
package config

import (
	"encoding/json"
	"example.com/dice-game-backend/internal/validation"
	"fmt"
	"net/http"
)

type LevelConfig struct {
	Level        int32 `json:"level"`
	EnergyCost   int32 `json:"energyCost"`
	TotalRolls   int32 `json:"totalRolls"`
	Target       int32 `json:"target"`
	EnergyReward int32 `json:"energyRewards"`
}

type GameConfig struct {
	Levels             []LevelConfig `json:"levels"`
	DefaultLevel       int32         `json:"defaultLevel"`
	MaxEnergy          int32         `json:"maxEnergy"`
	EnergyRegenSeconds int32         `json:"energyRegenSeconds"`
	DefaultLevelScore  int32         `json:"defaultLevelScore"`
}

type Server struct {
	GameConfig       *GameConfig
	requestValidator validation.RequestValidator
}

func NewConfigServer(rv validation.RequestValidator) *Server {
	return &Server{
		GameConfig: &GameConfig{
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
		},
		requestValidator: rv,
	}
}

// HandleConfigRequest responds with a game config
func (cs *Server) HandleConfigRequest(w http.ResponseWriter, r *http.Request) {

	if cs == nil {
		http.Error(w, "provided config server pointer is nil", http.StatusInternalServerError)
		return
	}

	err := cs.requestValidator.ValidateRequest(r)
	if err != nil {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"User Visible Realm\"")
		http.Error(w, "session error: "+err.Error(), http.StatusUnauthorized)
		return
	}

	fmt.Printf("config requested... \n ")

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(cs.GameConfig)
	if err != nil {
		http.Error(w, "could not encode game config", http.StatusInternalServerError)
	}
}

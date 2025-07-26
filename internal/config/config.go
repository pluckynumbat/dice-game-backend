// Package config: service which deals with the game config
package config

import (
	"encoding/json"
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
	Levels []LevelConfig `json:"levels"`
}

type Server struct {
	gameConfig       *GameConfig
	RequestValidator validation.RequestValidator
}

func NewConfigServer(rv validation.RequestValidator) *Server {
	return &Server{
		gameConfig: &GameConfig{
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
		},
		RequestValidator: rv,
	}
}

// HandleConfigRequest responds with a game config
func (cs *Server) HandleConfigRequest(w http.ResponseWriter, r *http.Request) {

	if cs == nil {
		http.Error(w, "provided config server pointer is nil", http.StatusInternalServerError)
		return
	}

	// TODO: check valid session

	fmt.Printf("config requested... \n ")

	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(cs.gameConfig)
	if err != nil {
		http.Error(w, "could not encode game config", http.StatusInternalServerError)
	}
}

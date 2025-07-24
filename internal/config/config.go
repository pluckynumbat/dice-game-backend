// Package config: service which deals with the game config
package config

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

var gameConfig = GameConfig{
	[]LevelConfig{
		{Level: 1, EnergyCost: 3, TotalRolls: 2, Target: 6, EnergyReward: 6},
		{Level: 2, EnergyCost: 3, TotalRolls: 3, Target: 4, EnergyReward: 6},
		{Level: 3, EnergyCost: 4, TotalRolls: 2, Target: 2, EnergyReward: 7},
		{Level: 4, EnergyCost: 4, TotalRolls: 3, Target: 1, EnergyReward: 7},
		{Level: 5, EnergyCost: 4, TotalRolls: 2, Target: 5, EnergyReward: 8},
	},
}
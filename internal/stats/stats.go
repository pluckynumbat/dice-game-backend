// Package stats: service which holds and provides details regarding relevant player stats
package stats

type PlayerLevelStats struct {
	Level     int32 `json:"level"`
	WinCount  int32 `json:"winCount"`
	LossCount int32 `json:"lossCount"`
	BestScore int32 `json:"BestScore"`
}

type PlayerStats struct {
	LevelStats []PlayerLevelStats `json:"levelStats"`
}
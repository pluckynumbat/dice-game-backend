// Package stats: service which holds and provides details regarding relevant player stats
package stats

import (
	"example.com/dice-game-backend/internal/validation"
	"sync"
)

// PlayerLevelStats are for a given level for a given player
type PlayerLevelStats struct {
	Level     int32 `json:"level"`
	WinCount  int32 `json:"winCount"`
	LossCount int32 `json:"lossCount"`
	BestScore int32 `json:"BestScore"`
}

// PlayerStats are for all levels for a given player
type PlayerStats struct {
	LevelStats []PlayerLevelStats `json:"levelStats"`
}

type Server struct {
	allStats   map[string]PlayerStats // this is for all players (and all levels)
	statsMutex sync.Mutex

	requestValidator validation.RequestValidator
}

func NewStatsServer(rv validation.RequestValidator) *Server {
	return &Server{
		allStats:         map[string]PlayerStats{},
		statsMutex:       sync.Mutex{},
		requestValidator: rv,
	}
}

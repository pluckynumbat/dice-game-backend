// Package stats: service which holds and provides details regarding relevant player stats
package stats

import (
	"encoding/json"
	"example.com/dice-game-backend/internal/validation"
	"fmt"
	"net/http"
	"sync"
)

// PlayerLevelStats are for a given level for a given player
type PlayerLevelStats struct {
	Level     int32 `json:"level"`
	WinCount  int32 `json:"winCount"`
	LossCount int32 `json:"lossCount"`
	BestScore int32 `json:"bestScore"`
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

// HandlePlayerStatsRequest responds with the player stats data if present
func (ss *Server) HandlePlayerStatsRequest(w http.ResponseWriter, r *http.Request) {

	if ss == nil {
		http.Error(w, "provided profile server pointer is nil", http.StatusInternalServerError)
		return
	}

	err := ss.requestValidator.ValidateRequest(r)
	if err != nil {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"User Visible Realm\"")
		http.Error(w, "session error: "+err.Error(), http.StatusUnauthorized)
		return
	}

	id := r.PathValue("id")
	fmt.Printf("player stats requested for id: %v \n ", id)

	ss.statsMutex.Lock()
	defer ss.statsMutex.Unlock()

	statsData := &PlayerStats{}

	playerStats, ok := ss.allStats[id]
	if ok {
		statsData = &playerStats
	} // if no stats exist for the player yet, send an empty entry

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(statsData)
	if err != nil {
		http.Error(w, "could not encode player data", http.StatusInternalServerError)
	}
}

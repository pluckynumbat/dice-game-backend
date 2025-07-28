// Package stats: service which holds and provides details regarding relevant player stats
package stats

import (
	"encoding/json"
	"example.com/dice-game-backend/internal/validation"
	"fmt"
	"net/http"
	"sync"
)

const defaultLevelCount = 10

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

// ReturnUpdatedPlayerStats will update a given PlayerLevelStats entry and return that player's stats
func (ss *Server) ReturnUpdatedPlayerStats(playerID string, newStatsDelta *PlayerLevelStats) (*PlayerStats, error) {

	if ss == nil {
		return nil, fmt.Errorf("provided stats server pointer is nil")
	}

	if newStatsDelta == nil {
		return nil, fmt.Errorf("provided new stats pointer is nil")
	}

	ss.statsMutex.Lock()
	defer ss.statsMutex.Unlock()

	// level to look for
	levelIndex := newStatsDelta.Level - 1

	// get the required player
	playerStats, present := ss.allStats[playerID]

	if !present {
		if levelIndex == 0 {
			// if this is for the first level, this could be the first ever stat entry for that player,
			// in that case create an empty player stats struct, and an empty level stats slice in it
			playerStats = PlayerStats{
				LevelStats: make([]PlayerLevelStats, 0, defaultLevelCount),
			}
		} else {
			// return an error
			return nil, fmt.Errorf("stats entry for id: %v (level %v) is not present \n", playerID, newStatsDelta.Level)
		}
	}

	// check if an entry exists for that level for that player
	if levelIndex < int32(len(playerStats.LevelStats)) {

		// update the level stats from the given delta input
		playerStats.LevelStats[levelIndex].WinCount += newStatsDelta.WinCount
		playerStats.LevelStats[levelIndex].LossCount += newStatsDelta.LossCount

		if newStatsDelta.WinCount == 1 {
			playerStats.LevelStats[levelIndex].BestScore = min(playerStats.LevelStats[levelIndex].BestScore, newStatsDelta.BestScore)
		}
	} else {
		// if not, just get the data from the given stats delta
		playerStats.LevelStats = append(playerStats.LevelStats, *newStatsDelta)
	}

	// write the updated data back to the stats map
	ss.allStats[playerID] = playerStats

	return &playerStats, nil
}

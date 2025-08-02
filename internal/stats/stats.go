// Package stats: service which holds and provides details regarding relevant player stats
package stats

import (
	"bytes"
	"context"
	"encoding/json"
	"example.com/dice-game-backend/internal/config"
	"example.com/dice-game-backend/internal/validation"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const invalidStatusCode int32 = -1

// Stats Specific Errors:
var serverNilError = fmt.Errorf("provided stats server pointer is nil")

type playerStatsNotFoundErr struct {
	playerID string
	level    int32
}

func (err playerStatsNotFoundErr) Error() string {
	return fmt.Sprintf("stats entry for id: %v (level %v) is not present \n", err.playerID, err.level)
}

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

type PlayerStatsResponse struct {
	PlayerID    string      `json:"playerID"`
	PlayerStats PlayerStats `json:"playerStats"`
}

type Server struct {
	allStats   map[string]PlayerStats // this is for all players (and all levels)
	statsMutex sync.Mutex

	defaultLevelCount int32

	requestValidator validation.RequestValidator
}

func NewStatsServer(rv validation.RequestValidator, gc *config.GameConfig) *Server {
	return &Server{
		allStats:   map[string]PlayerStats{},
		statsMutex: sync.Mutex{},

		defaultLevelCount: int32(len(gc.Levels)),

		requestValidator: rv,
	}
}

// HandlePlayerStatsRequest responds with the player stats data if present
func (ss *Server) HandlePlayerStatsRequest(w http.ResponseWriter, r *http.Request) {

	if ss == nil {
		http.Error(w, serverNilError.Error(), http.StatusInternalServerError)
		return
	}

	// check for valid session
	err := ss.requestValidator.ValidateRequest(r)
	if err != nil {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"User Visible Realm\"")
		http.Error(w, "session error: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// get the player id from the request path
	id := r.PathValue("id")
	fmt.Printf("player stats requested for id: %v \n ", id)

	ss.statsMutex.Lock()
	defer ss.statsMutex.Unlock()

	statsData := &PlayerStats{} // create the data struct for the response

	// make a request to the data service to read the stats entry for the player
	plStats, err, statusCode := ss.readStatsFromDB(id)
	if err != nil {
		if statusCode == int32(http.StatusBadRequest) {
			// entry does not exist yet, we will just send back the entry response
		} else {
			http.Error(w, "DB read error: "+err.Error(), http.StatusInternalServerError)
		}
	} else {
		statsData = plStats
	}

	// create and send the response
	response := &PlayerStatsWithID{
		PlayerID:    id,
		PlayerStats: *statsData,
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "could not encode player data", http.StatusInternalServerError)
	}
}

// ReturnUpdatedPlayerStats will update a given PlayerLevelStats entry and return that player's stats
func (ss *Server) ReturnUpdatedPlayerStats(playerID string, newStatsDelta *PlayerLevelStats) (*PlayerStats, error) {

	if ss == nil {
		return nil, serverNilError
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
				LevelStats: make([]PlayerLevelStats, 0, ss.defaultLevelCount),
			}
		} else {
			// return an error
			return nil, playerStatsNotFoundErr{playerID, newStatsDelta.Level}
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

// readStatsFromDB makes an internal (server to server) request to the data service to read the stats for the required player
func (ss *Server) readStatsFromDB(playerID string) (*PlayerStats, error, int32) {

	// create a new context
	ctx, cancel := context.WithTimeout(context.TODO(), 2*time.Second)
	defer cancel()

	// create the request
	reqURL := fmt.Sprintf("http://:5050/data/stats-internal/%v", playerID)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation error: " + err.Error()), invalidStatusCode
	}

	// send the request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request sending error: " + err.Error()), invalidStatusCode
	}
	defer resp.Body.Close()

	// check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("internal read stats request was not successful, status code: %v", resp.StatusCode), int32(resp.StatusCode)
	}

	//decode the response for the player data
	playerStats := &PlayerStats{}
	err = json.NewDecoder(resp.Body).Decode(playerStats)
	if err != nil {
		return nil, fmt.Errorf("error decoding the player stats: " + err.Error()), invalidStatusCode
	}

	return playerStats, nil, invalidStatusCode
}

// writeStatsToDB makes an internal (server to server) request to the data service to write the required player's stats entries
func (ss *Server) writeStatsToDB(plStatsWithID *PlayerStatsWithID) error {

	// create a new context
	ctx, cancel := context.WithTimeout(context.TODO(), 2*time.Second)
	defer cancel()

	// create the request body
	reqBody := &bytes.Buffer{}
	err := json.NewEncoder(reqBody).Encode(plStatsWithID)
	if err != nil {
		return fmt.Errorf("could not encode player data")
	}

	// create the request
	req, err := http.NewRequestWithContext(ctx, "POST", "http://:5050/data/stats-internal", reqBody)
	if err != nil {
		return fmt.Errorf("request creation error: " + err.Error())
	}

	// send the request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request sending error: " + err.Error())
	}
	defer resp.Body.Close()

	// check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("internal write stats request was not successful, status code: %v", resp.StatusCode)
	}

	return nil
}

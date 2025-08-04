// Package stats: service which provides details regarding relevant player stats
package stats

import (
	"bytes"
	"context"
	"encoding/json"
	"example.com/dice-game-backend/internal/config"
	"example.com/dice-game-backend/internal/data"
	"example.com/dice-game-backend/internal/shared/constants"
	"example.com/dice-game-backend/internal/shared/validation"
	"fmt"
	"log"
	"net/http"
	"os"
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

// Stats structs (not used in data storage):

// PlayerIDLevelStats is used as a request body for the internal request to update
// player's stats and return them (just a composite of a string, and player level stats)
type PlayerIDLevelStats struct {
	PlayerID        string                `json:"playerID"`
	LevelStatsDelta data.PlayerLevelStats `json:"levelStatsDelta"`
}

// Server is the core stats service provider
type Server struct {
	statsMutex sync.Mutex

	defaultLevelCount int32

	requestValidator validation.RequestValidator

	logger *log.Logger
}

// NewStatsServer returns an initialized pointer to the stats server
func NewStatsServer(rv validation.RequestValidator) *Server {
	return &Server{
		statsMutex: sync.Mutex{},

		defaultLevelCount: int32(len(config.Config.Levels)),

		requestValidator: rv,

		logger: log.New(os.Stdout, "stats: ", log.Ltime|log.LUTC|log.Lmsgprefix),
	}
}

// Run runs a given stats server on the given port
func (ss *Server) Run(port string) {

	mux := http.NewServeMux()

	mux.HandleFunc("GET /stats/player-stats/{id}", ss.HandlePlayerStatsRequest)
	mux.HandleFunc("POST /stats/player-stats-internal", ss.HandleUpdatePlayerStatsRequest)

	ss.logger.Println("the stats server is up and running...")

	addr := constants.CommonHost + ":" + port
	log.Fatal(http.ListenAndServe(addr, mux))
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
	ss.logger.Printf("player stats requested for id: %v", id)

	ss.statsMutex.Lock()
	defer ss.statsMutex.Unlock()

	statsData := &data.PlayerStats{} // create the data struct for the response

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
	response := &data.PlayerStatsWithID{
		PlayerID:    id,
		PlayerStats: *statsData,
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "could not encode player data: "+err.Error(), http.StatusInternalServerError)
	}
}

// ReturnUpdatedPlayerStats will update a given PlayerLevelStats entry and return that player's stats
func (ss *Server) ReturnUpdatedPlayerStats(playerID string, newStatsDelta *data.PlayerLevelStats) (*data.PlayerStats, error) {

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

	// make a request to the data service to read the stats entry for the player
	present := true // to store if there is an entry for the required player id in the stats DB
	playerStats, err, statusCode := ss.readStatsFromDB(playerID)
	if err != nil {
		if statusCode == int32(http.StatusBadRequest) {
			// entry does not exist yet, this can still be a valid case (dealt with below) if the player has no stats yet
			present = false
		} else {
			return nil, err
		}
	}

	if !present {
		if levelIndex == 0 {
			// if this is for the first level, this could be the first ever stat entry for that player,
			// in that case create an empty player stats struct, and an empty level stats slice in it
			playerStats = &data.PlayerStats{
				LevelStats: make([]data.PlayerLevelStats, 0, ss.defaultLevelCount),
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

	// make a request to the data service to write the stats entry for the player
	plStatsWithID := &data.PlayerStatsWithID{PlayerID: playerID, PlayerStats: *playerStats}
	err = ss.writeStatsToDB(plStatsWithID)
	if err != nil {
		return nil, err
	}

	return playerStats, nil
}

// HandleUpdatePlayerStatsRequest is a wrapper around the ReturnUpdatedPlayerStats() method which will
// be used to field internal (server to server) requests to return updated player stats
func (ss *Server) HandleUpdatePlayerStatsRequest(w http.ResponseWriter, r *http.Request) {

	if ss == nil {
		http.Error(w, serverNilError.Error(), http.StatusInternalServerError)
		return
	}

	// decode the request body, which should be a PlayerIDLevelStats struct
	decodedReq := &PlayerIDLevelStats{}
	err := json.NewDecoder(r.Body).Decode(decodedReq)
	if err != nil {
		http.Error(w, "could not decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	ss.logger.Printf("update and return stats request for id: %v", decodedReq.PlayerID)

	// try to update the stats
	updatedStats, err := ss.ReturnUpdatedPlayerStats(decodedReq.PlayerID, &decodedReq.LevelStatsDelta)
	if err != nil {
		http.Error(w, "could not update player stats: "+err.Error(), http.StatusBadRequest)
		return
	}

	// create and send the response
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(updatedStats)
	if err != nil {
		http.Error(w, "could not encode updated stats: "+err.Error(), http.StatusInternalServerError)
	}
}

// readStatsFromDB makes an internal (server to server) request to the data service to read the stats for the required player
func (ss *Server) readStatsFromDB(playerID string) (*data.PlayerStats, error, int32) {

	// create a new context
	ctx, cancel := context.WithTimeout(context.TODO(), constants.InternalRequestDeadlineSeconds*time.Second)
	defer cancel()

	// create the request
	reqURL := fmt.Sprintf("http://:%v/data/stats-internal/%v", constants.DataServerPort, playerID)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err, invalidStatusCode
	}

	// send the request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err, invalidStatusCode
	}
	defer resp.Body.Close()

	// check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("internal read stats request was not successful, status code: %v", resp.StatusCode), int32(resp.StatusCode)
	}

	//decode the response for the player data
	playerStats := &data.PlayerStats{}
	err = json.NewDecoder(resp.Body).Decode(playerStats)
	if err != nil {
		return nil, err, invalidStatusCode
	}

	return playerStats, nil, invalidStatusCode
}

// writeStatsToDB makes an internal (server to server) request to the data service to write the required player's stats entries
func (ss *Server) writeStatsToDB(plStatsWithID *data.PlayerStatsWithID) error {

	// create a new context
	ctx, cancel := context.WithTimeout(context.TODO(), constants.InternalRequestDeadlineSeconds*time.Second)
	defer cancel()

	// create the request body
	reqBody := &bytes.Buffer{}
	err := json.NewEncoder(reqBody).Encode(plStatsWithID)
	if err != nil {
		return err
	}

	// create the request
	reqURL := fmt.Sprintf("http://:%v/data/stats-internal", constants.DataServerPort)
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, reqBody)
	if err != nil {
		return err
	}

	// send the request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("internal write stats request was not successful, status code: %v", resp.StatusCode)
	}

	return nil
}

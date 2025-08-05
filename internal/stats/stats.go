// Package stats: provides all functionality related to retrieving, updating, and returning the player's
// historic data for each level they have played (like win count, loss count, and best score).
package stats

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

// Stats Specific Errors:
var serverNilError = fmt.Errorf("provided stats server pointer is nil")

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

// NewServer returns an initialized pointer to the stats server
func NewServer(rv validation.RequestValidator) *Server {
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
		errMsg := "error: session validation error: " + err.Error()
		ss.logger.Println(errMsg)
		http.Error(w, errMsg, http.StatusUnauthorized)
		return
	}

	// get the player id from the request path
	id := r.PathValue("id")
	ss.logger.Printf("player stats requested for id: %v", id)

	ss.statsMutex.Lock()
	defer ss.statsMutex.Unlock()

	statsData := &data.PlayerStats{} // create the data struct for the response

	// make a request to the data service to read the stats entry for the player
	plStats, err := ss.readStatsFromDB(id)
	if err != nil {
		if errors.Is(err, data.PlayerStatsNotFoundErr{PlayerID: id}) {
			// entry does not exist yet, we will just send back an empty response for stats
		} else {
			errMsg := "DB read error: " + err.Error()
			ss.logger.Println(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
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
		errMsg := "error: could not encode player data: " + err.Error()
		ss.logger.Println(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
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
	playerStats, err := ss.readStatsFromDB(playerID)
	if err != nil {
		if errors.Is(err, data.PlayerStatsNotFoundErr{PlayerID: playerID}) {
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
			// forward the error from above
			return nil, err
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
		errMsg := "error: could not decode request body: " + err.Error()
		ss.logger.Println(errMsg)
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	ss.logger.Printf("update and return stats request for id: %v", decodedReq.PlayerID)

	// try to update the stats
	updatedStats, err := ss.ReturnUpdatedPlayerStats(decodedReq.PlayerID, &decodedReq.LevelStatsDelta)
	if err != nil {
		errMsg := "error: could not update player stats: " + err.Error()
		ss.logger.Println(errMsg)
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	// create and send the response
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(updatedStats)
	if err != nil {
		errMsg := "error: could not encode updated stats: " + err.Error()
		ss.logger.Println(errMsg)
		http.Error(w, errMsg, http.StatusInternalServerError)
	}
}

// readStatsFromDB makes an internal (server to server) request to the data service to read the stats for the required player
func (ss *Server) readStatsFromDB(playerID string) (*data.PlayerStats, error) {

	// create a new context
	ctx, cancel := context.WithTimeout(context.TODO(), constants.InternalRequestDeadlineSeconds*time.Second)
	defer cancel()

	// create the request
	reqURL := fmt.Sprintf("%v://%v:%v/data/stats-internal/%v", constants.CommonProtocol, constants.CommonHost, constants.DataServerPort, playerID)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	// send the request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// check response status
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, data.PlayerStatsNotFoundErr{PlayerID: playerID}
		} else {
			return nil, fmt.Errorf("internal read stats request was not successful, status code %v", resp.StatusCode)
		}
	}

	//decode the response for the player data
	playerStats := &data.PlayerStats{}
	err = json.NewDecoder(resp.Body).Decode(playerStats)
	if err != nil {
		return nil, err
	}

	return playerStats, nil
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
	reqURL := fmt.Sprintf("%v://%v:%v/data/stats-internal", constants.CommonProtocol, constants.CommonHost, constants.DataServerPort)
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

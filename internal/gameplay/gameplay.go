// Package gameplay: service which deals with entering levels, playing the dice game etc.

package gameplay

import (
	"bytes"
	"context"
	"encoding/json"
	"example.com/dice-game-backend/internal/config"
	"example.com/dice-game-backend/internal/constants"
	"example.com/dice-game-backend/internal/data"
	"example.com/dice-game-backend/internal/profile"
	"example.com/dice-game-backend/internal/stats"
	"example.com/dice-game-backend/internal/validation"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// Stats Specific Errors:
var serverNilError = fmt.Errorf("provided gameplay server pointer is nil")

type EnterLevelRequestBody struct {
	PlayerID string `json:"playerID"`
	Level    int32  `json:"level"`
}

type EnterLevelResponse struct {
	AccessGranted bool            `json:"accessGranted"`
	Player        data.PlayerData `json:"playerData"`
}

type LevelResultRequestBody struct {
	PlayerID string  `json:"playerID"`
	Level    int32   `json:"level"`
	Rolls    []int32 `json:"rolls"`
}

// LevelResult only contains level result details, and is sent as part of the level result response
type LevelResult struct {
	Won              bool  `json:"won"`
	EnergyReward     int32 `json:"energyReward"`
	UnlockedNewLevel bool  `json:"unlockedNewLevel"`
}

type LevelResultResponse struct {
	LevelResult LevelResult      `json:"levelResult"`
	Player      data.PlayerData  `json:"playerData"`
	Stats       data.PlayerStats `json:"statsData"`
}

// Server is the core gameplay service provider
type Server struct {
	requestValidator validation.RequestValidator
	logger           *log.Logger
}

// NewGameplayServer returns an initialized pointer to the gameplay server
func NewGameplayServer(rv validation.RequestValidator) *Server {
	return &Server{
		requestValidator: rv,
		logger:           log.New(os.Stdout, "gameplay: ", log.Ltime|log.LUTC|log.Lmsgprefix),
	}
}

// Run runs a given gameplay server on the given port
func (gs *Server) Run(port string) {

	mux := http.NewServeMux()

	mux.HandleFunc("POST /gameplay/entry", gs.HandleEnterLevelRequest)
	mux.HandleFunc("POST /gameplay/result", gs.HandleLevelResultRequest)

	gs.logger.Println("the gameplay server is up and running...")

	addr := constants.CommonHost + ":" + port
	log.Fatal(http.ListenAndServe(addr, mux))
}

// HandleEnterLevelRequest accepts / rejects a request to enter a level based on current player data
// sends back the acceptance / rejection as well as the updated player data
func (gs *Server) HandleEnterLevelRequest(w http.ResponseWriter, r *http.Request) {

	if gs == nil {
		http.Error(w, serverNilError.Error(), http.StatusInternalServerError)
		return
	}

	err := gs.requestValidator.ValidateRequest(r)
	if err != nil {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"User Visible Realm\"")
		http.Error(w, "session error: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// decode the request
	entryRequest := &EnterLevelRequestBody{}
	err = json.NewDecoder(r.Body).Decode(entryRequest)
	if err != nil {
		http.Error(w, "could not decode the entry request: "+err.Error(), http.StatusBadRequest)
		return
	}
	gs.logger.Printf("request to enter level %v by player id %v", entryRequest.Level, entryRequest.PlayerID)

	// get the config and the player data
	cfg := config.Config
	if entryRequest.Level < 0 || entryRequest.Level > int32(len(cfg.Levels)) {
		http.Error(w, "invalid level in request", http.StatusBadRequest)
		return
	}

	// make a request to the profile service for the player data
	player, err := gs.getPlayerFromProfile(entryRequest.PlayerID, r.Header.Get("Session-Id"))
	if err != nil {
		http.Error(w, "player error: "+err.Error(), http.StatusBadRequest)
		return
	}

	// create the response
	entryResponse := &EnterLevelResponse{
		AccessGranted: false,
		Player:        *player,
	}

	energyCost := cfg.Levels[entryRequest.Level-1].EnergyCost

	// has the player unlocked the level?
	// does the player have enough energy to enter the level?
	if player.Level >= entryRequest.Level && player.Energy >= energyCost {

		entryResponse.AccessGranted = true

		// if player can enter, reduce the amount of energy
		// make a request to the profile service to update the player data
		updatedPlayer, updateErr := gs.updatePlayerData(entryRequest.PlayerID, -energyCost, player.Level)
		if updateErr != nil {
			http.Error(w, "player error: "+updateErr.Error(), http.StatusInternalServerError)
			return
		}

		entryResponse.Player = *updatedPlayer
	}

	// send level entry acceptance / rejection in response
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(entryResponse)
	if err != nil {
		http.Error(w, "could not encode the response: "+err.Error(), http.StatusInternalServerError)
	}
}

// HandleLevelResultRequest checks the rolls that the player made in a given level,
// decides if the level was won or lost, and sends back updated player data
func (gs *Server) HandleLevelResultRequest(w http.ResponseWriter, r *http.Request) {

	if gs == nil {
		http.Error(w, serverNilError.Error(), http.StatusInternalServerError)
		return
	}

	err := gs.requestValidator.ValidateRequest(r)
	if err != nil {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"User Visible Realm\"")
		http.Error(w, "session error: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// decode the request
	request := &LevelResultRequestBody{}
	err = json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		http.Error(w, "could not decode the level result request: "+err.Error(), http.StatusBadRequest)
		return
	}
	gs.logger.Printf("request for level results for level %v by player id %v", request.Level, request.PlayerID)

	// get the config and player, do basic validation there
	cfg := config.Config

	// make a request to the profile service for the player data
	player, err := gs.getPlayerFromProfile(request.PlayerID, r.Header.Get("Session-Id"))
	if err != nil {
		http.Error(w, "player error: "+err.Error(), http.StatusBadRequest)
		return
	}

	if request.Level < 0 || request.Level > int32(len(cfg.Levels)) || request.Level > player.Level {
		http.Error(w, "invalid level in request", http.StatusBadRequest)
		return
	}

	// check rolls against level requirement, decide win/loss and if new level was unlocked
	levelConfig := cfg.Levels[request.Level-1]
	rollCount := int32(len(request.Rolls))
	levelCount := int32(len(cfg.Levels))

	if request.Rolls == nil || rollCount > levelConfig.TotalRolls {
		http.Error(w, "invalid rolls data in request", http.StatusBadRequest)
		return
	}

	won := request.Rolls[rollCount-1] == levelConfig.Target
	newLevelUnlocked := won && request.Level == player.Level && request.Level < levelCount

	// update player data based on win / loss, and if new level was unlocked
	energyDelta := int32(0)
	if won {
		energyDelta = levelConfig.EnergyReward
	}

	newPlayerLevel := player.Level
	if newLevelUnlocked {
		newPlayerLevel += 1
	}

	// create a new level result to send in the response
	levelResult := &LevelResult{
		Won:              won,
		EnergyReward:     energyDelta,
		UnlockedNewLevel: newLevelUnlocked,
	}

	// update the player data to send back in the response
	// make a request to the profile service to update the player data
	updatedPlayer, err := gs.updatePlayerData(request.PlayerID, energyDelta, newPlayerLevel)
	if err != nil {
		http.Error(w, "player error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// update stats entry for this level (update win count, loss count, best score if better)
	newStatsDelta := &data.PlayerLevelStats{
		Level:     request.Level,
		WinCount:  0,
		LossCount: 0,
		BestScore: config.Config.DefaultLevelScore,
	}

	if won {
		newStatsDelta.WinCount = 1
		newStatsDelta.BestScore = rollCount
	} else {
		newStatsDelta.LossCount = 1
	}

	// make a request to the stats server to update the player stats
	updatedStats, err := gs.returnUpdatedPlayerStats(request.PlayerID, newStatsDelta)
	if err != nil {
		http.Error(w, "stats error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// create the response
	response := &LevelResultResponse{
		LevelResult: *levelResult,
		Player:      *updatedPlayer,
		Stats:       *updatedStats,
	}

	// send the response back
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "could not encode the response: "+err.Error(), http.StatusInternalServerError)
	}
}

// getPlayerFromProfile makes an internal (server to server) request to the profile service to get the required player data
func (gs *Server) getPlayerFromProfile(playerID string, sessionID string) (*data.PlayerData, error) {

	// create a new context
	ctx, cancel := context.WithTimeout(context.TODO(), constants.InternalRequestDeadlineSeconds*time.Second)
	defer cancel()

	// create the request
	reqURL := fmt.Sprintf("http://:%v/profile/player-data/%v", constants.ProfileServerPort, playerID)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Session-Id", sessionID)

	// send the request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("internal get player data request was not successful, status code %v", resp.StatusCode)
	}

	//decode the response for the player data
	playerData := &data.PlayerData{}
	err = json.NewDecoder(resp.Body).Decode(playerData)
	if err != nil {
		return nil, err
	}

	return playerData, nil
}

// updatePlayerData makes an internal (server to server) request to the profile service to update the required player data
func (gs *Server) updatePlayerData(playerID string, energyDelta int32, newLevel int32) (*data.PlayerData, error) {

	// create a new context
	ctx, cancel := context.WithTimeout(context.TODO(), constants.InternalRequestDeadlineSeconds*time.Second)
	defer cancel()

	// create the request body
	reqBody := &bytes.Buffer{}
	err := json.NewEncoder(reqBody).Encode(&profile.PlayerIDLevelEnergy{
		PlayerID:    playerID,
		Level:       newLevel,
		EnergyDelta: energyDelta,
	})
	if err != nil {
		return nil, err
	}

	// create the request
	reqURL := fmt.Sprintf("http://:%v/profile/player-data-internal", constants.ProfileServerPort)
	req, err := http.NewRequestWithContext(ctx, "PUT", reqURL, reqBody)
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
		return nil, fmt.Errorf("internal update player request was not successful, status code %v", resp.StatusCode)
	}

	//decode the response for the player data
	playerData := &data.PlayerData{}
	err = json.NewDecoder(resp.Body).Decode(playerData)
	if err != nil {
		return nil, err
	}

	return playerData, nil
}

// returnUpdatedPlayerStats makes an internal (server to server) request to the stats service to update the required player stats
func (gs *Server) returnUpdatedPlayerStats(playerID string, newStatsDelta *data.PlayerLevelStats) (*data.PlayerStats, error) {

	// create a new context
	ctx, cancel := context.WithTimeout(context.TODO(), constants.InternalRequestDeadlineSeconds*time.Second)
	defer cancel()

	// create the request body
	reqBody := &bytes.Buffer{}
	err := json.NewEncoder(reqBody).Encode(&stats.PlayerIDLevelStats{
		PlayerID:        playerID,
		LevelStatsDelta: *newStatsDelta,
	})
	if err != nil {
		return nil, err
	}

	// create the request
	reqURL := fmt.Sprintf("http://:%v/stats/player-stats-internal", constants.StatsServerPort)
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, reqBody)
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
		return nil, fmt.Errorf("internal update player request was not successful, status code %v", resp.StatusCode)
	}

	//decode the response for the player stats
	playerStats := &data.PlayerStats{}
	err = json.NewDecoder(resp.Body).Decode(playerStats)
	if err != nil {
		return nil, err
	}

	return playerStats, nil
}

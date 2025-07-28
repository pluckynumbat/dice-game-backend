// Package gameplay: service which deals with entering levels, playing the dice game etc.

package gameplay

import (
	"encoding/json"
	"example.com/dice-game-backend/internal/config"
	"example.com/dice-game-backend/internal/profile"
	"example.com/dice-game-backend/internal/stats"
	"example.com/dice-game-backend/internal/validation"
	"fmt"
	"net/http"
)

type EnterLevelRequestBody struct {
	PlayerID string `json:"playerID"`
	Level    int32  `json:"level"`
}

type EnterLevelResponse struct {
	AccessGranted bool               `json:"accessGranted"`
	Player        profile.PlayerData `json:"playerData"`
}

type LevelResultRequestBody struct {
	PlayerID string  `json:"playerID"`
	Level    int32   `json:"level"`
	Rolls    []int32 `json:"rolls"`
}

type LevelResultResponse struct {
	LevelWon bool               `json:"levelWon"`
	Player   profile.PlayerData `json:"playerData"`
	Stats    stats.PlayerStats  `json:"statsData"`
}

type Server struct {
	configServer  *config.Server
	profileServer *profile.Server
	statsServer   *stats.Server

	requestValidator validation.RequestValidator
}

func NewGameplayServer(rv validation.RequestValidator, cs *config.Server, ps *profile.Server, ss *stats.Server) *Server {
	return &Server{
		configServer:     cs,
		profileServer:    ps,
		statsServer:      ss,
		requestValidator: rv,
	}
}

// HandleEnterLevelRequest accepts / rejects a request to enter a level based on current player data
// sends back the acceptance / rejection as well as the updated player data
func (gs *Server) HandleEnterLevelRequest(w http.ResponseWriter, r *http.Request) {

	if gs == nil {
		http.Error(w, "provided gameplay server pointer is nil", http.StatusInternalServerError)
		return
	}

	if gs.configServer == nil || gs.profileServer == nil {
		http.Error(w, "config server / profile server pointer is nil, please check construction", http.StatusInternalServerError)
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
		http.Error(w, "could not decode the entry request", http.StatusBadRequest)
		return
	}
	fmt.Printf("request to enter level %v by player id %v \n ", entryRequest.Level, entryRequest.PlayerID)

	// get the config and the player data
	cfg, err := gs.configServer.GetConfig()
	if err != nil {
		http.Error(w, "config error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if entryRequest.Level < 0 || entryRequest.Level > int32(len(cfg.Levels)) {
		http.Error(w, "invalid level in request", http.StatusBadRequest)
		return
	}

	player, err := gs.profileServer.GetPlayer(entryRequest.PlayerID)
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
		updatedPlayer, updateErr := gs.profileServer.UpdatePlayerData(entryRequest.PlayerID, -energyCost, player.Level)
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
		http.Error(w, "could not encode the response", http.StatusInternalServerError)
	}
}

// HandleLevelResultRequest checks the rolls that the player made in a given level,
// decides if the level was won or lost, and sends back updated player data
// TODO: this will also update the stats when that service is in place
func (gs *Server) HandleLevelResultRequest(w http.ResponseWriter, r *http.Request) {

	if gs == nil {
		http.Error(w, "provided gameplay server pointer is nil", http.StatusInternalServerError)
		return
	}

	if gs.configServer == nil || gs.profileServer == nil || gs.statsServer == nil {
		http.Error(w, "config server / profile server / stats server pointer is nil, please check construction", http.StatusInternalServerError)
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
		http.Error(w, "could not decode the level result request", http.StatusBadRequest)
		return
	}
	fmt.Printf("request for level results for level %v by player id %v \n ", request.Level, request.PlayerID)

	// get the config and player, do basic validation there
	cfg, err := gs.configServer.GetConfig()
	if err != nil {
		http.Error(w, "config error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	player, err := gs.profileServer.GetPlayer(request.PlayerID)
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

	updatedPlayer, err := gs.profileServer.UpdatePlayerData(request.PlayerID, energyDelta, newPlayerLevel)
	if err != nil {
		http.Error(w, "player error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: update stats entry for this level and this player when that service is present
	// (update win count, loss count, best score)

	// create the response
	response := &LevelResultResponse{
		LevelWon: won,
		Player:   *updatedPlayer,
	}

	// send the response back
	// TODO: (along with updated stats when that service is present)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "could not encode the response", http.StatusInternalServerError)
	}
}

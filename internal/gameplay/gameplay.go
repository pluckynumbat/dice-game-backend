// Package gameplay: service which deals with entering levels, playing the dice game etc.

package gameplay

import (
	"encoding/json"
	"example.com/dice-game-backend/internal/config"
	"example.com/dice-game-backend/internal/profile"
	"example.com/dice-game-backend/internal/validation"
	"fmt"
	"net/http"
)

type EnterLevelRequest struct {
	PlayerID string `json:"playerID"`
	Level    int32  `json:"level"`
}

type EnterLevelResponse struct {
	AccessGranted bool               `json:"accessGranted"`
	Player        profile.PlayerData `json:"playerData"`
}

type Server struct {
	configServer  *config.Server
	profileServer *profile.Server

	// TODO: will also need a pointer to the stats service

	requestValidator validation.RequestValidator
}

func NewGameplayServer(rv validation.RequestValidator, cs *config.Server, ps *profile.Server) *Server {
	return &Server{
		configServer:     cs,
		profileServer:    ps,
		requestValidator: rv,
	}
}

// HandleEnterLevelRequest accepts / rejects a request to enter a level based on current player data
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
	entryRequest := &EnterLevelRequest{}
	err = json.NewDecoder(r.Body).Decode(entryRequest)
	if err != nil {
		http.Error(w, "could not decode the entry request", http.StatusBadRequest)
		return
	}
	fmt.Printf("request to enter level %v by player id %v \n ", entryRequest.PlayerID, entryRequest.Level)

	// get the config and the player data
	cfg, err := gs.configServer.GetConfig()
	if err != nil {
		http.Error(w, "config error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if entryRequest.Level < 0 || entryRequest.Level >= int32(len(cfg.Levels)) {
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

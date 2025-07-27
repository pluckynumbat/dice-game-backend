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

	err := gs.requestValidator.ValidateRequest(r)
	if err != nil {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"User Visible Realm\"")
		http.Error(w, "session error: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// TODO: decode the request

	fmt.Printf("request to enter level %v by player id %v \n ")

	// TODO: get the config and the player data

	// TODO: compare level requirements with player data

	// TODO: if player can enter, reduce the amount of energy

	w.Header().Set("Content-Type", "application/json")

	// TODO: send level entry acceptance / rejection in response
	err = json.NewEncoder(w).Encode(&EnterLevelResponse{})
	if err != nil {
		http.Error(w, "could not encode the response", http.StatusInternalServerError)
	}
}

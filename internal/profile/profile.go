// Package profile: service which deals with the player data
package profile

import (
	"encoding/json"
	"sync"
)

const defaultLevel = 1
const maxEnergy = 10

type PlayerData struct {
	PlayerID string `json:"playerID"`
	Level    int32  `json:"level"`
	Energy   int32  `json:"energy"`
}

type Server struct {
	players      map[string]PlayerData
	playersMutex sync.Mutex
}

func NewProfileServer() *Server {
	return &Server{
		players:      map[string]PlayerData{},
		playersMutex: sync.Mutex{},
	}
}

// HandleNewPlayerRequest creates a new player in the map
func (ps *Server) HandleNewPlayerRequest(w http.ResponseWriter, r *http.Request) {

	if ps == nil {
		http.Error(w, "provided profile server pointer is nil", http.StatusInternalServerError)
	}

	// TODO: check valid session

	newPlayer := &PlayerData{
		PlayerID: "",
		Level:    defaultLevel,
		Energy:   maxEnergy,
	}

	err := json.NewDecoder(r.Body).Decode(newPlayer)
	if err != nil {
		http.Error(w, "could not decode player id", http.StatusInternalServerError)
		return
	}

}
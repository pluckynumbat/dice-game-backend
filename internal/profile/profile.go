// Package profile: service which deals with the player data
package profile

import (
	"encoding/json"
	"fmt"
	"net/http"
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
		return
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

	ps.playersMutex.Lock()
	defer ps.playersMutex.Unlock()

	_, exists := ps.players[newPlayer.PlayerID]
	if exists {
		http.Error(w, "player exists already", http.StatusBadRequest)
		return
	}

	fmt.Printf("creating new player with id: %v \n ", newPlayer.PlayerID)

	ps.players[newPlayer.PlayerID] = *newPlayer

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(ps.players[newPlayer.PlayerID])
	if err != nil {
		http.Error(w, "could not encode player data", http.StatusInternalServerError)
	}
}

// HandlePlayerDataRequest responds with the player data
func (ps *Server) HandlePlayerDataRequest(w http.ResponseWriter, r *http.Request) {

	if ps == nil {
		http.Error(w, "provided profile server pointer is nil", http.StatusInternalServerError)
		return
	}

	// TODO: check valid session

	id := r.PathValue("id")

	fmt.Printf("player data requested for id: %v \n ", id)

	ps.playersMutex.Lock()
	defer ps.playersMutex.Unlock()

	player, ok := ps.players[id]
	if !ok {
		http.Error(w, "player not found", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(player)
	if err != nil {
		http.Error(w, "could not encode player data", http.StatusInternalServerError)
	}
}

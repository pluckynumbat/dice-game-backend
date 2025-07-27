// Package profile: service which deals with the player data
package profile

import (
	"encoding/json"
	"example.com/dice-game-backend/internal/validation"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const defaultLevel = 1
const maxEnergy = 50
const energyRegenRate = 0.2 // energy regenerated per second TODO: should this come from a config instead?

type PlayerData struct {
	PlayerID       string `json:"playerID"`
	Level          int32  `json:"level"`
	Energy         int32  `json:"energy"`
	LastUpdateTime int64  `json:"lastUpdateTime"`
}

type Server struct {
	players      map[string]PlayerData
	playersMutex sync.Mutex

	requestValidator validation.RequestValidator
}

func NewProfileServer(rv validation.RequestValidator) *Server {
	return &Server{
		players:          map[string]PlayerData{},
		playersMutex:     sync.Mutex{},
		requestValidator: rv,
	}
}

// HandleNewPlayerRequest creates a new player in the map
func (ps *Server) HandleNewPlayerRequest(w http.ResponseWriter, r *http.Request) {

	if ps == nil {
		http.Error(w, "provided profile server pointer is nil", http.StatusInternalServerError)
		return
	}

	err := ps.requestValidator.ValidateRequest(r)
	if err != nil {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"User Visible Realm\"")
		http.Error(w, "session error: "+err.Error(), http.StatusUnauthorized)
		return
	}

	newPlayer := &PlayerData{
		PlayerID:       "",
		Level:          defaultLevel,
		Energy:         maxEnergy,
		LastUpdateTime: time.Now().UTC().Unix(),
	}

	err = json.NewDecoder(r.Body).Decode(newPlayer)
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

	err := ps.requestValidator.ValidateRequest(r)
	if err != nil {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"User Visible Realm\"")
		http.Error(w, "session error: "+err.Error(), http.StatusUnauthorized)
		return
	}

	id := r.PathValue("id")

	fmt.Printf("player data requested for id: %v \n ", id)

	ps.playersMutex.Lock()
	defer ps.playersMutex.Unlock()

	player, ok := ps.players[id]
	if !ok {
		http.Error(w, "player not found", http.StatusBadRequest)
		return
	}

	// passive energy regeneration
	err = ps.updateEnergy(&player, 0)
	if err != nil {
		http.Error(w, "energy error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// write back to the map
	ps.players[id] = player

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(player)
	if err != nil {
		http.Error(w, "could not encode player data", http.StatusInternalServerError)
	}
}

// GetPlayer returns the player data when requested, with updated energy from the passive regeneration
func (ps *Server) GetPlayer(playerID string) (*PlayerData, error) {

	if ps == nil {
		return nil, fmt.Errorf("provided profile server pointer is nil")
	}
	ps.playersMutex.Lock()
	defer ps.playersMutex.Unlock()

	player, ok := ps.players[playerID]
	if !ok {
		return nil, fmt.Errorf("player with id: %v was not found \n", playerID)
	}

	// passive energy regeneration
	err := ps.updateEnergy(&player, 0)
	if err != nil {
		return nil, err
	}

	// write back to the map
	ps.players[playerID] = player

	return &player, nil
}

// UpdatePlayerData will first apply passive energy regeneration to the player,
// then apply the given energy delta, and finally change the level of the player if needed
func (ps *Server) UpdatePlayerData(playerID string, energyDelta int32, newLevel int32) (*PlayerData, error) {

	if ps == nil {
		return nil, fmt.Errorf("provided profile server pointer is nil")
	}
	ps.playersMutex.Lock()
	defer ps.playersMutex.Unlock()

	player, ok := ps.players[playerID]
	if !ok {
		return nil, fmt.Errorf("player with id: %v was not found \n", playerID)
	}

	// update energy based on passive energy regeneration & new energyDelta
	err := ps.updateEnergy(&player, energyDelta)
	if err != nil {
		return nil, err
	}

	// update level
	if player.Level != newLevel {
		player.Level = newLevel
	}

	// write the player back to the map
	ps.players[playerID] = player

	return &player, nil
}

// updateEnergy will update energy values of the given player:
// first it will update (possibly stale) energy based on passive energy regeneration
// then it will update it based on the provided energy delta
func (ps *Server) updateEnergy(player *PlayerData, newEnergyDelta int32) error {

	if player == nil {
		return fmt.Errorf("nil player data pointer")
	}

	now := time.Now().UTC().Unix()

	// 1. make energy values current: (update the energy of the player based
	// on time passed since last update, and the energy regeneration rate)
	if now > player.LastUpdateTime {

		extraEnergy := float64(now-player.LastUpdateTime) * energyRegenRate
		player.Energy = min(player.Energy+int32(extraEnergy), maxEnergy)
	}

	// 2. update to final value based on provided delta (which can be positive / negative)
	if newEnergyDelta != 0 {
		player.Energy = min(player.Energy+newEnergyDelta, maxEnergy)
	}

	// 3. make the timestamp current
	player.LastUpdateTime = now

	return nil
}

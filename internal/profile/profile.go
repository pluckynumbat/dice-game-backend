// Package profile: service which deals with the player data
package profile

import (
	"encoding/json"
	"example.com/dice-game-backend/internal/config"
	"example.com/dice-game-backend/internal/validation"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Profile Specific Errors:
var serverNilError = fmt.Errorf("provided profile server pointer is nil")

type playerNotFoundErr struct {
	playerID string
}

func (err playerNotFoundErr) Error() string {
	return fmt.Sprintf("player with id: %v was not found \n", err.playerID)
}

// NewPlayerRequestBody just contains the player ID
type NewPlayerRequestBody struct {
	PlayerID string `json:"playerID"`
}

// PlayerData (response struct for the requests) stores player related live data like level , energy etc.
type PlayerData struct {
	PlayerID       string `json:"playerID"`
	Level          int32  `json:"level"`
	Energy         int32  `json:"energy"`
	LastUpdateTime int64  `json:"lastUpdateTime"`
}

type Server struct {
	players      map[string]PlayerData
	playersMutex sync.Mutex

	defaultLevel         int32
	maxLevel             int32
	maxEnergy            int32
	energyRegenPerSecond float64

	requestValidator validation.RequestValidator
}

func NewProfileServer(rv validation.RequestValidator, gc *config.GameConfig) *Server {

	ps := &Server{
		players:      map[string]PlayerData{},
		playersMutex: sync.Mutex{},

		defaultLevel:         gc.DefaultLevel,
		maxLevel:             int32(len(gc.Levels)),
		maxEnergy:            gc.MaxEnergy,
		energyRegenPerSecond: 0,

		requestValidator: rv,
	}

	// avoid divide by zero
	if gc.EnergyRegenSeconds != 0 {
		ps.energyRegenPerSecond = 1 / float64(gc.EnergyRegenSeconds)
	}

	return ps
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
		Level:          ps.defaultLevel,
		Energy:         ps.maxEnergy,
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

	player, err := ps.GetPlayer(id)
	if err != nil {
		http.Error(w, "player error: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(player)
	if err != nil {
		http.Error(w, "could not encode player data", http.StatusInternalServerError)
	}
}

// GetPlayer returns the player data when requested, with updated energy from the passive regeneration
func (ps *Server) GetPlayer(playerID string) (*PlayerData, error) {

	if ps == nil {
		return nil, serverNilError
	}
	ps.playersMutex.Lock()
	defer ps.playersMutex.Unlock()

	player, ok := ps.players[playerID]
	if !ok {
		return nil, playerNotFoundErr{playerID}
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
		return nil, serverNilError
	}
	ps.playersMutex.Lock()
	defer ps.playersMutex.Unlock()

	player, ok := ps.players[playerID]
	if !ok {
		return nil, playerNotFoundErr{playerID}
	}

	// update energy based on passive energy regeneration & new energyDelta
	err := ps.updateEnergy(&player, energyDelta)
	if err != nil {
		return nil, err
	}

	// update level (if needed)
	if player.Level < newLevel {
		player.Level = min(newLevel, ps.maxLevel)
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

		extraEnergy := float64(now-player.LastUpdateTime) * ps.energyRegenPerSecond
		player.Energy = min(player.Energy+int32(extraEnergy), ps.maxEnergy)
	}

	// 2. update to final value based on provided delta (which can be positive / negative)
	if newEnergyDelta != 0 {
		player.Energy = min(player.Energy+newEnergyDelta, ps.maxEnergy)
	}

	// 3. make the timestamp current
	player.LastUpdateTime = now

	return nil
}

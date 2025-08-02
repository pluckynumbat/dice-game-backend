// Package profile: service which deals with the player data
package profile

import (
	"bytes"
	"context"
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

// NewPlayerRequestBody just contains the player ID
type NewPlayerRequestBody struct {
	PlayerID string `json:"playerID"`
}

// PlayerData (response struct for the client requests and request body for write requests to the data service)
// stores player related live data like level , energy etc.
type PlayerData struct {
	PlayerID       string `json:"playerID"`
	Level          int32  `json:"level"`
	Energy         int32  `json:"energy"`
	LastUpdateTime int64  `json:"lastUpdateTime"`
}

type Server struct {
	playersMutex sync.Mutex

	defaultLevel         int32
	maxLevel             int32
	maxEnergy            int32
	energyRegenPerSecond float64

	requestValidator validation.RequestValidator
}

func NewProfileServer(rv validation.RequestValidator, gc *config.GameConfig) *Server {

	ps := &Server{
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
		http.Error(w, serverNilError.Error(), http.StatusInternalServerError)
		return
	}

	err := ps.requestValidator.ValidateRequest(r)
	if err != nil {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"User Visible Realm\"")
		http.Error(w, "session error: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// decode the request body for the player ID
	decodedReq := &NewPlayerRequestBody{}
	err = json.NewDecoder(r.Body).Decode(decodedReq)
	if err != nil {
		http.Error(w, "could not decode player id", http.StatusInternalServerError)
		return
	}

	// create the new player struct from the player ID
	newPlayer := &PlayerData{
		PlayerID:       decodedReq.PlayerID,
		Level:          ps.defaultLevel,
		Energy:         ps.maxEnergy,
		LastUpdateTime: time.Now().UTC().Unix(),
	}

	ps.playersMutex.Lock()
	defer ps.playersMutex.Unlock()

	// check with the data service to see if the player exists already (they should not)
	// so successful get here means failure for us!
	_, err = ps.readPlayerFromDB(decodedReq.PlayerID)
	if err == nil {
		http.Error(w, "player exists already", http.StatusBadRequest)
		return
	}

	fmt.Printf("creating new player with id: %v \n ", newPlayer.PlayerID)

	// tell the data service to store the new player in the player DB
	err = ps.writePlayerToDB(newPlayer)
	if err != nil {
		http.Error(w, "DB write error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// send the response back
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(newPlayer)
	if err != nil {
		http.Error(w, "could not encode player data", http.StatusInternalServerError)
	}
}

// HandlePlayerDataRequest responds with the player data
func (ps *Server) HandlePlayerDataRequest(w http.ResponseWriter, r *http.Request) {

	if ps == nil {
		http.Error(w, serverNilError.Error(), http.StatusInternalServerError)
		return
	}

	err := ps.requestValidator.ValidateRequest(r)
	if err != nil {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"User Visible Realm\"")
		http.Error(w, "session error: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// get the id from the request uri
	id := r.PathValue("id")
	fmt.Printf("player data requested for id: %v \n ", id)

	player, err := ps.GetPlayer(id)
	if err != nil {
		http.Error(w, "player error: "+err.Error(), http.StatusBadRequest)
		return
	}

	// send the response back
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

	// send request to the data service to look the player up
	player, err := ps.readPlayerFromDB(playerID)
	if err != nil {
		return nil, fmt.Errorf("DB read error: " + err.Error())
	}

	// passive energy regeneration
	err = ps.updateEnergy(player, 0)
	if err != nil {
		return nil, err
	}

	// send request to the data service to write the player back to the DB
	err = ps.writePlayerToDB(player)
	if err != nil {
		return nil, fmt.Errorf("DB write error: " + err.Error())
	}

	return player, nil
}

// UpdatePlayerData will first apply passive energy regeneration to the player,
// then apply the given energy delta, and finally change the level of the player if needed
func (ps *Server) UpdatePlayerData(playerID string, energyDelta int32, newLevel int32) (*PlayerData, error) {

	if ps == nil {
		return nil, serverNilError
	}

	ps.playersMutex.Lock()
	defer ps.playersMutex.Unlock()

	// send request to the data service to look the player up
	player, err := ps.readPlayerFromDB(playerID)
	if err != nil {
		return nil, fmt.Errorf("DB read error: " + err.Error())
	}

	// update energy based on passive energy regeneration & new energyDelta
	err = ps.updateEnergy(player, energyDelta)
	if err != nil {
		return nil, err
	}

	// update level (if needed)
	if player.Level < newLevel {
		player.Level = min(newLevel, ps.maxLevel)
	}

	// send request to the data service to write back the player
	err = ps.writePlayerToDB(player)
	if err != nil {
		return nil, fmt.Errorf("DB write error: " + err.Error())
	}

	return player, nil
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

// readPlayerFromDB makes an internal (server to server) request to the data service to read the required player
func (ps *Server) readPlayerFromDB(playerID string) (*PlayerData, error) {

	// create a new context
	ctx, cancel := context.WithTimeout(context.TODO(), 2*time.Second)
	defer cancel()

	// create the request
	reqURL := fmt.Sprintf("http://:5050/data/player-internal/%v", playerID)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation error: " + err.Error())
	}

	// send the request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request sending error: " + err.Error())
	}
	defer resp.Body.Close()

	// check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("config request was not successful")
	}

	//decode the response for the player data
	playerData := &PlayerData{}
	err = json.NewDecoder(resp.Body).Decode(playerData)
	if err != nil {
		return nil, fmt.Errorf("error decoding the player data: " + err.Error())
	}

	return playerData, nil
}

// writePlayerToDB makes an internal (server to server) request to the data service to write the required player entry
func (ps *Server) writePlayerToDB(player *PlayerData) error {

	// create a new context
	ctx, cancel := context.WithTimeout(context.TODO(), 2*time.Second)
	defer cancel()

	// create the request body
	reqBody := &bytes.Buffer{}
	err := json.NewEncoder(reqBody).Encode(player)
	if err != nil {
		return fmt.Errorf("could not encode player data")
	}

	// create the request
	req, err := http.NewRequestWithContext(ctx, "POST", "http://:5050/data/player-internal", reqBody)
	if err != nil {
		return fmt.Errorf("request creation error: " + err.Error())
	}

	// send the request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request sending error: " + err.Error())
	}
	defer resp.Body.Close()

	// check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("set player to db request was not successful")
	}

	return nil
}

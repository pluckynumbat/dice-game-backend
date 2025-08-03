// Package profile: service which deals with the player data
package profile

import (
	"bytes"
	"context"
	"encoding/json"
	"example.com/dice-game-backend/internal/config"
	"example.com/dice-game-backend/internal/constants"
	"example.com/dice-game-backend/internal/types"
	"example.com/dice-game-backend/internal/validation"
	"fmt"
	"log"
	"net/http"
	"os"
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

// Server is the core profile service provider
type Server struct {
	playersMutex sync.Mutex

	defaultLevel         int32
	maxLevel             int32
	maxEnergy            int32
	energyRegenPerSecond float64

	requestValidator validation.RequestValidator

	logger *log.Logger
}

// NewProfileServer returns an initialized pointer to the profile server
func NewProfileServer(rv validation.RequestValidator) *Server {

	ps := &Server{
		playersMutex: sync.Mutex{},

		defaultLevel:         config.Config.DefaultLevel,
		maxLevel:             int32(len(config.Config.Levels)),
		maxEnergy:            config.Config.MaxEnergy,
		energyRegenPerSecond: 0,

		requestValidator: rv,
		logger:           log.New(os.Stdout, "profile: ", log.Ltime|log.LUTC|log.Lmsgprefix),
	}

	// avoid divide by zero
	if config.Config.EnergyRegenSeconds != 0 {
		ps.energyRegenPerSecond = 1 / float64(config.Config.EnergyRegenSeconds)
	}

	return ps
}

// Run runs a given profile server on the given port
func (ps *Server) Run(port string) {

	mux := http.NewServeMux()

	mux.HandleFunc("POST /profile/new-player", ps.HandleNewPlayerRequest)
	mux.HandleFunc("GET /profile/player-data/{id}", ps.HandlePlayerDataRequest)
	mux.HandleFunc("PUT /profile/player-data-internal", ps.HandleUpdatePlayerRequest)

	ps.logger.Println("the profile server is up and running...")

	addr := constants.CommonHost + ":" + port
	log.Fatal(http.ListenAndServe(addr, mux))
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
	decodedReq := &types.NewPlayerRequestBody{}
	err = json.NewDecoder(r.Body).Decode(decodedReq)
	if err != nil {
		http.Error(w, "could not decode player id: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// create the new player struct from the player ID
	newPlayer := &types.PlayerData{
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

	ps.logger.Printf("creating new player with id: %v", newPlayer.PlayerID)

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
		http.Error(w, "could not encode player data: "+err.Error(), http.StatusInternalServerError)
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
	ps.logger.Printf("player data requested for id: %v", id)

	player, err := ps.GetPlayer(id)
	if err != nil {
		http.Error(w, "player error: "+err.Error(), http.StatusBadRequest)
		return
	}

	// send the response back
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(player)
	if err != nil {
		http.Error(w, "could not encode player data: "+err.Error(), http.StatusInternalServerError)
	}
}

// GetPlayer returns the player data when requested, with updated energy from the passive regeneration
func (ps *Server) GetPlayer(playerID string) (*types.PlayerData, error) {

	if ps == nil {
		return nil, serverNilError
	}

	ps.playersMutex.Lock()
	defer ps.playersMutex.Unlock()

	// send request to the data service to look the player up
	player, err := ps.readPlayerFromDB(playerID)
	if err != nil {
		return nil, err
	}

	// passive energy regeneration
	err = ps.updateEnergy(player, 0)
	if err != nil {
		return nil, err
	}

	// send request to the data service to write the player back to the DB
	err = ps.writePlayerToDB(player)
	if err != nil {
		return nil, err
	}

	return player, nil
}

// UpdatePlayerData will first apply passive energy regeneration to the player,
// then apply the given energy delta, and finally change the level of the player if needed
func (ps *Server) UpdatePlayerData(playerID string, energyDelta int32, newLevel int32) (*types.PlayerData, error) {

	if ps == nil {
		return nil, serverNilError
	}

	ps.playersMutex.Lock()
	defer ps.playersMutex.Unlock()

	// send request to the data service to look the player up
	player, err := ps.readPlayerFromDB(playerID)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	return player, nil
}

// HandleUpdatePlayerRequest is a wrapper around the UpdatePlayerData() method which will
// be used to field internal (server to server) requests to return updated player data
func (ps *Server) HandleUpdatePlayerRequest(w http.ResponseWriter, r *http.Request) {

	if ps == nil {
		http.Error(w, serverNilError.Error(), http.StatusInternalServerError)
		return
	}

	// decode the request body, which should be a PlayerIDLevelEnergy struct
	decodedReq := &types.PlayerIDLevelEnergy{}
	err := json.NewDecoder(r.Body).Decode(decodedReq)
	if err != nil {
		http.Error(w, "could not decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	ps.logger.Printf("update player data request for id: %v", decodedReq.PlayerID)

	// try to update the player data
	updatedPlayer, err := ps.UpdatePlayerData(decodedReq.PlayerID, decodedReq.EnergyDelta, decodedReq.Level)
	if err != nil {
		http.Error(w, "could not update player data: "+err.Error(), http.StatusBadRequest)
		return
	}

	// create and send the response
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(updatedPlayer)
	if err != nil {
		http.Error(w, "could not encode updated player data: "+err.Error(), http.StatusInternalServerError)
	}
}

// updateEnergy will update energy values of the given player:
// first it will update (possibly stale) energy based on passive energy regeneration
// then it will update it based on the provided energy delta
func (ps *Server) updateEnergy(player *types.PlayerData, newEnergyDelta int32) error {

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
func (ps *Server) readPlayerFromDB(playerID string) (*types.PlayerData, error) {

	// create a new context
	ctx, cancel := context.WithTimeout(context.TODO(), constants.InternalRequestDeadlineSeconds*time.Second)
	defer cancel()

	// create the request
	reqURL := fmt.Sprintf("http://:%v/data/player-internal/%v", constants.DataServerPort, playerID)
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
		if resp.StatusCode == http.StatusBadRequest {
			return nil, playerNotFoundErr{playerID}
		} else {
			return nil, fmt.Errorf("internal read player request was not successful, status code %v \n", resp.StatusCode)
		}
	}

	//decode the response for the player data
	playerData := &types.PlayerData{}
	err = json.NewDecoder(resp.Body).Decode(playerData)
	if err != nil {
		return nil, err
	}

	return playerData, nil
}

// writePlayerToDB makes an internal (server to server) request to the data service to write the required player entry
func (ps *Server) writePlayerToDB(player *types.PlayerData) error {

	// create a new context
	ctx, cancel := context.WithTimeout(context.TODO(), constants.InternalRequestDeadlineSeconds*time.Second)
	defer cancel()

	// create the request body
	reqBody := &bytes.Buffer{}
	err := json.NewEncoder(reqBody).Encode(player)
	if err != nil {
		return err
	}

	// create the request
	reqURL := fmt.Sprintf("http://:%v/data/player-internal", constants.DataServerPort)
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
		return fmt.Errorf("internal write player request was not successful, status code %v \n", resp.StatusCode)
	}

	return nil
}

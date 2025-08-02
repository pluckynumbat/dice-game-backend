// Package data: the storage service for the backend, it stores player data and player stats
// All requests to this server are internal (only come from other servers in the backend)
package data

import (
	"encoding/json"
	"example.com/dice-game-backend/internal/profile"
	"example.com/dice-game-backend/internal/stats"
	"fmt"
	"log"
	"net/http"
	"sync"
)

const serverHost string = ""
const serverPort string = "5050"

// Data service Specific Errors:
var serverNilError = fmt.Errorf("provided data server pointer is nil")

type playerNotFoundErr struct {
	playerID string
}

func (err playerNotFoundErr) Error() string {
	return fmt.Sprintf("player with id: %v was not found in the player DB \n", err.playerID)
}

type playerStatsNotFoundErr struct {
	playerID string
}

func (err playerStatsNotFoundErr) Error() string {
	return fmt.Sprintf("stats entry for id: %v was not found in the stats DB \n", err.playerID)
}

type Server struct {
	playersDB    map[string]profile.PlayerData
	playersMutex sync.Mutex

	statsDB    map[string]stats.PlayerStats
	statsMutex sync.Mutex
}

func NewDataServer() *Server {

	ds := &Server{
		playersDB:    map[string]profile.PlayerData{},
		playersMutex: sync.Mutex{},

		statsDB:    map[string]stats.PlayerStats{},
		statsMutex: sync.Mutex{},
	}

	return ds
}

// RunDataServer runs a given data server on the designated port
func (ds *Server) RunDataServer() {

	if ds == nil {
		fmt.Println(serverNilError)
		return
	}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /data/player-internal", ds.HandleWritePlayerDataRequest)
	mux.HandleFunc("GET /data/player-internal/{id}", ds.HandleReadPlayerDataRequest)

	mux.HandleFunc("POST /data/stats-internal", ds.HandleWritePlayerStatsRequest)
	mux.HandleFunc("GET /data/stats-internal/{id}", ds.HandleReadPlayerStatsRequest)

	addr := serverHost + ":" + serverPort
	log.Fatal(http.ListenAndServe(addr, mux))
}

// HandleWritePlayerDataRequest writes the given player data to a player DB entry
// (creating a new player DB entry if not present)
func (ds *Server) HandleWritePlayerDataRequest(w http.ResponseWriter, r *http.Request) {

	if ds == nil {
		http.Error(w, serverNilError.Error(), http.StatusInternalServerError)
		return
	}

	// decode the request body, which should be a PlayerData struct
	decodedReq := &profile.PlayerData{}
	err := json.NewDecoder(r.Body).Decode(decodedReq)
	if err != nil {
		http.Error(w, "could not decode request body", http.StatusBadRequest)
		return
	}

	if decodedReq.PlayerID == "" {
		http.Error(w, "cannot write an entry with a blank player id", http.StatusBadRequest)
		return
	}

	fmt.Printf("writing player DB entry for id: %v \n ", decodedReq.PlayerID)

	ds.playersMutex.Lock()
	defer ds.playersMutex.Unlock()

	// write the entry to the database
	ds.playersDB[decodedReq.PlayerID] = *decodedReq

	// provide the success response, the body is meaningless
	// (status of 200: operation will be considered a success)
	w.Header().Set("Content-Type", "text/plain")
	_, err = fmt.Fprint(w, "success")
	if err != nil {
		http.Error(w, "could not write response", http.StatusInternalServerError)
		return
	}
}

// HandleReadPlayerDataRequest returns the player DB entry of the requested player ID (if present)
func (ds *Server) HandleReadPlayerDataRequest(w http.ResponseWriter, r *http.Request) {

	if ds == nil {
		http.Error(w, serverNilError.Error(), http.StatusInternalServerError)
		return
	}

	// get the id from the path value of the request
	id := r.PathValue("id")
	fmt.Printf("player DB entry requested for id: %v \n ", id)

	ds.playersMutex.Lock()
	defer ds.playersMutex.Unlock()

	// fetch the entry (if present) from the database
	player, ok := ds.playersDB[id]
	if !ok {
		notFoundErr := playerNotFoundErr{id}
		http.Error(w, notFoundErr.Error(), http.StatusBadRequest)
		return
	}

	//write the response with the player entry in it and set it back
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(player)
	if err != nil {
		http.Error(w, "could not encode player data", http.StatusInternalServerError)
	}
}

// HandleWritePlayerStatsRequest writes the given player stats to a stats DB entry
// (creating a new stats DB entry if not present)
func (ds *Server) HandleWritePlayerStatsRequest(w http.ResponseWriter, r *http.Request) {

	if ds == nil {
		http.Error(w, serverNilError.Error(), http.StatusInternalServerError)
		return
	}

	// decode the request body, which should be a PlayerStatsWithID struct
	decodedReq := &stats.PlayerStatsWithID{}
	err := json.NewDecoder(r.Body).Decode(decodedReq)
	if err != nil {
		http.Error(w, "could not decode request body", http.StatusBadRequest)
		return
	}

	if decodedReq.PlayerID == "" {
		http.Error(w, "cannot write an entry with a blank player id", http.StatusBadRequest)
		return
	}

	fmt.Printf("writing stats DB entry for id: %v \n ", decodedReq.PlayerID)

	ds.statsMutex.Lock()
	defer ds.statsMutex.Unlock()

	// write the entry to the database
	ds.statsDB[decodedReq.PlayerID] = decodedReq.PlayerStats

	// provide the success response, the body is meaningless
	// (status of 200: operation will be considered a success)
	w.Header().Set("Content-Type", "text/plain")
	_, err = fmt.Fprint(w, "success")
	if err != nil {
		http.Error(w, "could not write response", http.StatusInternalServerError)
		return
	}
}

// HandleReadPlayerStatsRequest returns the stats DB entry of the requested player ID (if present)
func (ds *Server) HandleReadPlayerStatsRequest(w http.ResponseWriter, r *http.Request) {

	if ds == nil {
		http.Error(w, serverNilError.Error(), http.StatusInternalServerError)
		return
	}

	// get the id from the path value of the request
	id := r.PathValue("id")
	fmt.Printf("stats DB entry requested for id: %v \n ", id)

	ds.statsMutex.Lock()
	defer ds.statsMutex.Unlock()

	// fetch the entry (if present) from the database
	plStats, ok := ds.statsDB[id]
	if !ok {
		notFoundErr := playerStatsNotFoundErr{id}
		http.Error(w, notFoundErr.Error(), http.StatusBadRequest)
		return
	}

	//write the response with the player entry in it and set it back
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(plStats)
	if err != nil {
		http.Error(w, "could not encode player data", http.StatusInternalServerError)
	}
}

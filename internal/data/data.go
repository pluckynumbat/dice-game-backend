// Package data: is the storage service for the backend, it stores player data and player stats
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

	mux.HandleFunc("POST /data/player-internal", ds.HandleSetPlayerRequest)
	mux.HandleFunc("GET /data/player-internal/{id}", ds.HandleGetPlayerRequest)

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

	fmt.Printf("setting player DB entry for id: %v \n ", id)

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
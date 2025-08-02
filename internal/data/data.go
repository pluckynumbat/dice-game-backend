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
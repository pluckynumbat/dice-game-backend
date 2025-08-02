// Package data: is the storage service for the backend, it stores player data and player stats
package data

import (
	"example.com/dice-game-backend/internal/profile"
	"example.com/dice-game-backend/internal/stats"
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
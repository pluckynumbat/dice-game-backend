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

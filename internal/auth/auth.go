// Package auth: service which deals with the authenticating the player and managing user sessions
package auth

import "sync"

type SessionData struct {
	PlayerID       string
	SessionID      string
	LastActionTime int64
}

type Server struct {
	credentials map[string]string
	credMutex   sync.Mutex

	sessions  map[string]*SessionData
	sessMutex sync.Mutex
}

func NewAuthServer() *Server {
	return &Server{
		credentials: map[string]string{},
		credMutex:   sync.Mutex{},

		sessions:  map[string]*SessionData{},
		sessMutex: sync.Mutex{},
	}
}

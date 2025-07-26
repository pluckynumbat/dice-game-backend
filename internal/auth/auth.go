// Package auth: service which deals with the authenticating the player and managing user sessions
package auth

import "sync"

type Server struct {
	credentials map[string]string
	credMutex   sync.Mutex

	sessions map[string]string
	sessMutex sync.Mutex
}

func NewAuthServer() *Server {
	return &Server{
		credentials: map[string]string{},
		credMutex:   sync.Mutex{},

		sessions: map[string]string{},
		sessMutex:   sync.Mutex{},
	}
}

// Package gameplay: service which deals with entering levels, playing the dice game etc.

package gameplay

import (
	"example.com/dice-game-backend/internal/config"
	"example.com/dice-game-backend/internal/profile"
	"example.com/dice-game-backend/internal/validation"
)

type Server struct {
	configServer  *config.Server
	profileServer *profile.Server

	// TODO: will also need a pointer to the stats service

	requestValidator validation.RequestValidator
}

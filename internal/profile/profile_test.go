package profile

import (
	"errors"
	"example.com/dice-game-backend/internal/auth"
	"example.com/dice-game-backend/internal/config"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestNewProfileServer(t *testing.T) {

	authServer := auth.NewAuthServer()
	profileServer := NewProfileServer(authServer, config.NewConfigServer(authServer).GameConfig)

	if profileServer == nil {
		t.Fatal("new profile server should not return a nil server pointer")
	}

	if profileServer.players == nil {
		t.Fatal("new profile server should not contain a nil players pointer")
	}
}
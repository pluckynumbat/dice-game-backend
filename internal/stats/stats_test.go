package stats

import (
	"bytes"
	"encoding/json"
	"errors"
	"example.com/dice-game-backend/internal/auth"
	"example.com/dice-game-backend/internal/config"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestNewStatsServer(t *testing.T) {

	authServer := auth.NewAuthServer()
	statsServer := NewStatsServer(authServer, config.NewConfigServer(authServer).GameConfig)

	if statsServer == nil {
		t.Fatal("new config server should not return a nil server pointer")
	}

	if statsServer.allStats == nil {
		t.Fatal("new config server should not contain a nil all stats pointer")
	}
}
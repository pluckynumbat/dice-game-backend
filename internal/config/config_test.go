package config

import (
	"testing"
)

func TestNewConfigServer(t *testing.T) {

	configServer := NewConfigServer()

	if configServer == nil {
		t.Fatal("new config server should not return a nil server pointer")
	}

	if configServer.gameConfig == nil {
		t.Fatal("new config server should not contain a nil game config")
	}

	if configServer.gameConfig.Levels == nil {
		t.Fatal("new config server should not contain a game config with nil levels")
	}

	if len(configServer.gameConfig.Levels) == 0 {
		t.Fatal("new config server should not contain a game config with empty levels")
	}
}
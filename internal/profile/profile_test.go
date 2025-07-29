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

func TestServer_GetPlayer(t *testing.T) {

	authServer := auth.NewAuthServer()
	ps := NewProfileServer(authServer, config.NewConfigServer(authServer).GameConfig)
	ps.players["player2"] = PlayerData{"player2", 1, 50, time.Now().UTC().Unix()}
	ps.players["player3"] = PlayerData{"player3", 1, 20, time.Now().UTC().Unix() - 100}

	tests := []struct {
		name       string
		server     *Server
		playerID   string
		wantPlayer *PlayerData
		expError   error
	}{
		{"nil server", nil, "", nil, serverNilError},
		{"invalid player", ps, "player1", nil, playerNotFoundErr{"player1"}},
		{"valid player", ps, "player2", &PlayerData{"player2", 1, 50, time.Now().UTC().Unix()}, nil},
		{"valid player, restore energy", ps, "player2", &PlayerData{"player2", 1, 50, time.Now().UTC().Unix()}, nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			gotPlayer, gotErr := test.server.GetPlayer(test.playerID)
			if gotErr != nil {
				if errors.Is(gotErr, test.expError) {
					fmt.Println(gotErr)
				} else {
					t.Fatalf("GetPlayer() failed with an unexpected error, %v", gotErr)
				}
			} else {
				if !reflect.DeepEqual(gotPlayer, test.wantPlayer) {
					t.Errorf("GetPlayer() gave incorrect results, want: %v, got: %v", test.wantPlayer, gotPlayer)
				}
			}
		})
	}
}

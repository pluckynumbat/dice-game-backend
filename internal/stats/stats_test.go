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


func TestServer_ReturnUpdatedPlayerStats(t *testing.T) {

	var s1, s2 *Server

	authServer := auth.NewAuthServer()
	s2 = NewStatsServer(authServer, config.NewConfigServer(authServer).GameConfig)
	s2.allStats["player2"] = PlayerStats{
		nil,
	}
	s2.allStats["player3"] = PlayerStats{
		LevelStats: []PlayerLevelStats{
			{1, 2, 3, 1},
			{2, 1, 4, 2},
			{3, 0, 1, 99},
		},
	}

	tests := []struct {
		name      string
		server    *Server
		playerID  string
		lvlStats  *PlayerLevelStats
		wantStats *PlayerStats
		expError  error
	}{
		{"nil server", s1, "player1", &PlayerLevelStats{}, &PlayerStats{}, serverNilError},
		{"invalid player", s2, "player1", &PlayerLevelStats{5, 1, 0, 4}, nil, playerStatsNotFoundErr{"player1", 5}},
		{"valid new player", s2, "player2", &PlayerLevelStats{1, 0, 1, 99}, &PlayerStats{
			LevelStats: []PlayerLevelStats{
				{1, 0, 1, 99},
			},
		}, nil},
		{"valid existing player", s2, "player3", &PlayerLevelStats{3, 1, 0, 3}, &PlayerStats{
			LevelStats: []PlayerLevelStats{
				{1, 2, 3, 1},
				{2, 1, 4, 2},
				{3, 1, 1, 3},
			},
		}, nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			gotStats, gotErr := test.server.ReturnUpdatedPlayerStats(test.playerID, test.lvlStats)
			if gotErr != nil {
				if errors.Is(gotErr, test.expError) {
					fmt.Println(gotErr)
				} else {
					t.Fatalf("ReturnUpdatedPlayerStats() failed with an unexpected error, %v", gotErr)
				}
			} else {
				if !reflect.DeepEqual(gotStats, test.wantStats) {
					t.Errorf("ReturnUpdatedPlayerStats() gave incorrect results, want: %v, got: %v", test.wantStats, gotStats)
				}
			}
		})
	}
}

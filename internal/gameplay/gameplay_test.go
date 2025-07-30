package gameplay

import (
	"example.com/dice-game-backend/internal/auth"
	"example.com/dice-game-backend/internal/config"
	"example.com/dice-game-backend/internal/profile"
	"example.com/dice-game-backend/internal/stats"
	"testing"
)

func TestNewGameplayServer(t *testing.T) {

	as := auth.NewAuthServer()
	cs := config.NewConfigServer(as)
	ps := profile.NewProfileServer(as, cs.GameConfig)
	ss := stats.NewStatsServer(as, cs.GameConfig)

	gs := NewGameplayServer(as, ps, ss, cs.GameConfig)

	if gs == nil {
		t.Fatal("new profile server should not return a nil server pointer")
	}
}

package gameplay

import (
	"bytes"
	"encoding/json"
	"example.com/dice-game-backend/internal/auth"
	"example.com/dice-game-backend/internal/config"
	"example.com/dice-game-backend/internal/profile"
	"example.com/dice-game-backend/internal/stats"
	"net/http"
	"net/http/httptest"
	"reflect"
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

func setupTestAuth() (*auth.Server, string, error) {
	buf := &bytes.Buffer{}
	reqBody := &auth.LoginRequestBody{IsNewUser: true, ServerVersion: "0"}
	err := json.NewEncoder(buf).Encode(reqBody)
	if err != nil {
		return nil, "", err
	}

	newAuthReq := httptest.NewRequest(http.MethodPost, "/auth/login", buf)
	newAuthReq.SetBasicAuth("user1", "pass1")
	authRespRec := httptest.NewRecorder()

	as := auth.NewAuthServer()
	as.HandleLoginRequest(authRespRec, newAuthReq)
	sID := authRespRec.Header().Get("Session-Id")

	return as, sID, nil
}
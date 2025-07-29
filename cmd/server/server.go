package main

import (
	"example.com/dice-game-backend/internal/auth"
	"example.com/dice-game-backend/internal/config"
	"example.com/dice-game-backend/internal/gameplay"
	"example.com/dice-game-backend/internal/profile"
	"example.com/dice-game-backend/internal/stats"
	"fmt"
	"log"
	"net/http"
)

const serverHost string = ""
const serverPort string = "8080"

func main() {
	fmt.Println("starting the server...")

	mux := http.NewServeMux()

	authServer := auth.NewAuthServer()

	configServer := config.NewConfigServer(authServer)
	profileServer := profile.NewProfileServer(authServer)
	statsServer := stats.NewStatsServer(authServer)

	gameplayServer := gameplay.NewGameplayServer(authServer, configServer, profileServer, statsServer)

	mux.HandleFunc("POST /auth/login", authServer.HandleLoginRequest)
	mux.HandleFunc("GET /auth/logout", authServer.HandleLogoutRequest)

	mux.HandleFunc("GET /config/game-config", configServer.HandleConfigRequest)

	mux.HandleFunc("POST /profile/new-player", profileServer.HandleNewPlayerRequest)
	mux.HandleFunc("GET /profile/player-data/{id}", profileServer.HandlePlayerDataRequest)

	mux.HandleFunc("GET /stats/player-stats/{id}", statsServer.HandlePlayerStatsRequest)

	mux.HandleFunc("POST /gameplay/entry", gameplayServer.HandleEnterLevelRequest)
	mux.HandleFunc("POST /gameplay/result", gameplayServer.HandleLevelResultRequest)

	addr := serverHost + ":" + serverPort
	log.Fatal(http.ListenAndServe(addr, mux))
}

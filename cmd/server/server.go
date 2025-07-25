package main

import (
	"example.com/dice-game-backend/internal/config"
	"example.com/dice-game-backend/internal/profile"
	"fmt"
	"log"
	"net/http"
)

const serverHost string = ""
const serverPort string = "8080"

func main() {
	fmt.Println("starting the server...")

	mux := http.NewServeMux()

	configServer := config.NewConfigServer()

	profileServer := profile.NewProfileServer()

	mux.HandleFunc("GET /config/game-config", configServer.HandleConfigRequest)

	mux.HandleFunc("POST /profile/new-player", profileServer.HandleNewPlayerRequest)
	mux.HandleFunc("GET /profile/player-data/{id}", profileServer.HandlePlayerDataRequest)

	addr := serverHost + ":" + serverPort
	log.Fatal(http.ListenAndServe(addr, mux))
}

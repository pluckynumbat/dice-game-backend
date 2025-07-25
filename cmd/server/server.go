package main

import (
	"example.com/dice-game-backend/internal/config"
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

	mux.HandleFunc("GET /config/game-config", configServer.HandleConfigRequest)


	addr := serverHost + ":" + serverPort
	log.Fatal(http.ListenAndServe(addr, mux))
}

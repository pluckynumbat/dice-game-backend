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
	fmt.Println("started the server...")

	configServer := config.NewConfigServer()

	http.HandleFunc("GET /config/game-config", configServer.HandleConfigRequest)


	addr := serverHost + ":" + serverPort
	log.Fatal(http.ListenAndServe(addr, nil))
}

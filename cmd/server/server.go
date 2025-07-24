package main

import (
	"example.com/dice-game-backend/internal/config"
	"fmt"
	"log"
	"net/http"
)

const serverHost string = "localhost"
const serverPort string = "8080"

func main() {
	fmt.Println("started the server...")

	http.HandleFunc("GET /config/v1/game-config", config.HandleConfigRequest)

	addr := serverHost + ":" + serverPort
	log.Fatal(http.ListenAndServe(addr, nil))
}

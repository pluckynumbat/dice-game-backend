// Used to spin up an auth server as an independent microservice on the given port
package main

import (
	"example.com/dice-game-backend/internal/auth"
	"example.com/dice-game-backend/internal/constants"
	"fmt"
)

func main() {
	fmt.Println("starting the auth server...")
	authServer := auth.NewAuthServer()
	authServer.RunAuthServer(constants.AuthServerPort)
}

// Used to spin up a config server as an independent microservice on the designated port
package main

import (
	"example.com/dice-game-backend/internal/config"
	"example.com/dice-game-backend/internal/shared/constants"
	"example.com/dice-game-backend/internal/shared/validation"
	"fmt"
	"net/http"
)

type requestValidator struct {
}

func (rv *requestValidator) ValidateRequest(req *http.Request) error {

	if rv == nil {
		return fmt.Errorf("the validator is nil")
	}

	return validation.ValidateRequest(req)
}

func main() {
	fmt.Println("starting the config server...")
	configServer := config.NewConfigServer(&requestValidator{})
	configServer.Run(constants.ConfigServerPort)
}

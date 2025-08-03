// Used to spin up a stats server as an independent microservice on the given port
package main

import (
	"example.com/dice-game-backend/internal/constants"
	"example.com/dice-game-backend/internal/stats"
	"example.com/dice-game-backend/internal/validation"
	"fmt"
	"net/http"
)

type requestValidator struct{}

func (rv *requestValidator) ValidateRequest(req *http.Request) error {

	if rv == nil {
		return fmt.Errorf("the validator is nil")
	}
	return validation.ValidateRequest(req)
}

func main() {
	fmt.Println("starting the stats server...")

	statsServer := stats.NewStatsServer(&requestValidator{})
	statsServer.Run(constants.StatsServerPort)
}

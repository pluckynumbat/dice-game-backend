// Used to spin up a stats server as an independent microservice on the given port
package main

import (
	"example.com/dice-game-backend/internal/shared/constants"
	"example.com/dice-game-backend/internal/shared/validation"
	"example.com/dice-game-backend/internal/stats"
	"fmt"
	"net/http"
)

// the request validator struct implements a wrapper around the common method
// that propagates session based validation requests to the auth service
type requestValidator struct{}

func (rv *requestValidator) ValidateRequest(req *http.Request) error {

	if rv == nil {
		return fmt.Errorf("the validator is nil")
	}
	return validation.ValidateRequest(req)
}

func main() {
	fmt.Println("starting the stats server...")
	statsServer := stats.NewServer(&requestValidator{})
	statsServer.Run(constants.StatsServerPort)
}

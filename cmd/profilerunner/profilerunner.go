// Used to spin up a profile server as an independent microservice on the given port
package main

import (
	"example.com/dice-game-backend/internal/profile"
	"example.com/dice-game-backend/internal/shared/constants"
	"example.com/dice-game-backend/internal/shared/validation"
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
	fmt.Println("starting the profile server...")
	profileServer := profile.NewServer(&requestValidator{})
	profileServer.Run(constants.ProfileServerPort)
}

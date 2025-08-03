// Convenience runner used to spin up all the different microservices from a single
// terminal command, and then wait on user input to shut them all down when done
package main

import (
	"example.com/dice-game-backend/internal/validation"
	"fmt"
	"net/http"
)

// the request validator struct implements a wrapper around the common method
// that propagates session based validation requests to the auth service
type requestValidator struct {
}

func (rv *requestValidator) ValidateRequest(req *http.Request) error {

	if rv == nil {
		return fmt.Errorf("the validator is nil")
	}

	return validation.ValidateRequest(req)
}


func main() {
	fmt.Println("starting all the servers...")
}

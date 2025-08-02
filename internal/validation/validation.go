// Package validation adds a custom interface used for validation purposes
// as well as an implementation suitable for microservices
package validation

import (
	"context"
	"example.com/dice-game-backend/internal/constants"
	"fmt"
	"net/http"
	"time"
)

// RequestValidator implementor can validate http requests
// (currently used by auth.Server to validate requests based on valid sessions)
type RequestValidator interface {
	ValidateRequest(req *http.Request) error
}

// ValidateRequest is an implementation that the servers will use when running as their own microservices
// They will send an internal request to the auth server, and check the response for errors
func ValidateRequest(req *http.Request) error {

	// extract the "Session-Id" header
	sessionIdHeader := req.Header["Session-Id"]
	if sessionIdHeader == nil {
		return fmt.Errorf("no session id header in the request")
	}

	// create a context, then a request with it
	ctx, cancel := context.WithTimeout(req.Context(), 5*time.Second)
	defer cancel()

	reqURL := fmt.Sprintf("http://:%v/auth/validation-internal", constants.AuthServerPort)
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, nil)
	if err != nil {
		return fmt.Errorf("request creation error: %v \n", err)
	}
	req.Header.Set("Session-ID", sessionIdHeader[0])

	// send the request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request sending error: %v \n", err)
	}
	defer resp.Body.Close()

	// check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("validation was not successful")
	}

	return nil
}

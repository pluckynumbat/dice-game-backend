// Package validation adds a custom interface used for validation purposes
package validation

import "net/http"

// RequestValidator implementor can validate http requests
// (currently used by auth.Server to validate requests based on valid sessions)
type RequestValidator interface {
	ValidateRequest(req *http.Request) error
}

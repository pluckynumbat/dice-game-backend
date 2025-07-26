// Package auth: service which deals with the authenticating the player and managing user sessions
package auth

import "sync"

type LoginResponse struct {
	PlayerID string `json:"playerID"`
}

type SessionData struct {
	PlayerID       string
	SessionID      string
	LastActionTime int64
}

type Server struct {
	credentials map[string]string
	credMutex   sync.Mutex

	sessions  map[string]*SessionData
	sessMutex sync.Mutex
}

func NewAuthServer() *Server {
	return &Server{
		credentials: map[string]string{},
		credMutex:   sync.Mutex{},

		sessions:  map[string]*SessionData{},
		sessMutex: sync.Mutex{},
	}
}

// decodeAuthHeaderPayload will take the authorization header and return a username and password if successful
// reference: https://en.wikipedia.org/wiki/Basic_access_authentication
func (as *Server) decodeAuthHeaderPayload(encodedCred string) (string, string, error) {

	if as == nil {
		return "", "", fmt.Errorf("provided auth server pointer is nil")
	}

	// trim away the first 6 elements which are the prefix 'Basic '
	encodedCred = encodedCred[6:]

	// decode the base64 data
	decodedCred, err := base64.StdEncoding.DecodeString(encodedCred)

	if err != nil {
		return "", "", fmt.Errorf("cannot decode the given credentials")
	}

	// separate the username and password
	decodedStrings := strings.Split(string(decodedCred), ":")

	return decodedStrings[0], decodedStrings[1], nil
}

// generatePlayerID generates a sha 256 hash from the username,
// and returns the first few elements of it as the new player id
func (as *Server) generatePlayerID(input string) (string, error) {

	if input == "" {
		return "", fmt.Errorf("input is empty")
	}

	hash := sha256.New()
	hash.Write([]byte(input))
	hashBytes := hash.Sum(nil)

	resultString := hex.EncodeToString(hashBytes[:4])

	return resultString, nil
}

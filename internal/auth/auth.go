package auth

import "sync"

type Server struct {
	credentials map[string]string
	credMutex   sync.Mutex
}

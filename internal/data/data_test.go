package data

import (
	"encoding/json"
	"example.com/dice-game-backend/internal/profile"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestNewDataServer(t *testing.T) {
	dataServer := NewDataServer()

	if dataServer == nil {
		t.Fatal("new data server should not return a nil server pointer")
	}

	if dataServer.playersDB == nil {
		t.Fatal("new data server should not contain a nil playersDB pointer")
	}

	if dataServer.statsDB == nil {
		t.Fatal("new profile server should not contain a nil statsDB pointer")
	}
}


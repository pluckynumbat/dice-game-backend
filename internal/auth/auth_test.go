package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestNewAuthServer(t *testing.T) {
	authServer := NewAuthServer()

	if authServer == nil {
		t.Fatal("new profile server should not return a nil server pointer")
	}

	if authServer.credentials == nil {
		t.Fatal("new profile server should not contain a nil credentials pointer")
	}

	if authServer.sessions == nil {
		t.Fatal("new profile server should not contain a nil credentials pointer")
	}

	if authServer.activePlayerIDs == nil {
		t.Fatal("new profile server should not contain a nil active player IDs pointer")
	}

	if authServer.serverVersion != strconv.FormatInt(time.Now().UTC().Unix(), 10) {
		t.Error("new profile server's server version should be the current UTC unix timestamp in seconds")
	}
}
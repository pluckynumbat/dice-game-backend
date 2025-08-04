// Convenience runner used to spin up all the different microservices from a single
// terminal command, and then wait on user input to shut them all down when done
package main

import (
	"example.com/dice-game-backend/internal/auth"
	"example.com/dice-game-backend/internal/config"
	"example.com/dice-game-backend/internal/data"
	"example.com/dice-game-backend/internal/gameplay"
	"example.com/dice-game-backend/internal/profile"
	"example.com/dice-game-backend/internal/shared/constants"
	"example.com/dice-game-backend/internal/shared/validation"
	"example.com/dice-game-backend/internal/stats"
	"fmt"
	"net/http"
	"time"
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

// This function loops till the player inputs the given quit keys ('0', or 'q', or 'Q')
// or manually interrupts (ctrl+c) the terminal window
func waitLoop() {
	userInput := ""

	for done := false; done != true; {

		_, err := fmt.Scan(&userInput)
		if err != nil {
			fmt.Println(fmt.Errorf("input failed with %v \n", err))
			break
		}

		switch userInput {
		case "Q", "q", "0":
			fmt.Println("shutting down all the servers...")
			done = true

		default:
			done = false
		}
	}
}

func main() {
	fmt.Println("starting all the servers...")

	rv := &requestValidator{}

	authServer := auth.NewServer()
	go authServer.Run(constants.AuthServerPort)

	dataServer := data.NewServer()
	go dataServer.Run(constants.DataServerPort)

	configServer := config.NewServer(rv)
	go configServer.Run(constants.ConfigServerPort)

	profileServer := profile.NewServer(rv)
	go profileServer.Run(constants.ProfileServerPort)

	statsServer := stats.NewStatsServer(rv)
	go statsServer.Run(constants.StatsServerPort)

	gameplayServer := gameplay.NewGameplayServer(rv)
	go gameplayServer.Run(constants.GameplayServerPort)

	time.Sleep(500 * time.Millisecond) // wait some time so that the following instructions to exit the loop are on the last line
	fmt.Println("at any point, press 0 or q or Q (followed by Enter) to quit...")
	waitLoop()
}

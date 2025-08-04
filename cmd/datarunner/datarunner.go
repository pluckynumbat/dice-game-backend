// Used to spin up a data server as an independent microservice on the given port
package main

import (
	"example.com/dice-game-backend/internal/data"
	"example.com/dice-game-backend/internal/shared/constants"
	"fmt"
)

func main() {
	fmt.Println("starting the data server...")
	dataServer := data.NewDataServer()
	dataServer.Run(constants.DataServerPort)
}

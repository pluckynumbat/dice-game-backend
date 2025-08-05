# Dice Game Backend

Welcome to the dice game backend! \
This is a microservice based architecture with 6 core services \
It is meant to be along with the [client repository](https://github.com/pluckynumbat/dice-game-client)

## Getting Started
1. Prerequisites: go version 1.24 or higher
2. Clone this repo, and navigate to the root of it!

## Part 1. Running the application
There are 2 ways to run the backend, **All In One** mode or **Manual** mode, I'll explain both in turn below:

## All In One Mode
### How to run:
#### via terminal
From the root of the repository, just type the command: \
`go run cmd/allrunner/allrunner.go` 

#### via IDE (like Goland)
Open the project in the IDE, navigate to `cmd/allrunner/allrunner.go` and press play on the main function

### what is **All In One** mode?
 - This spins up all the 6 core services on their designated ports, and provides a command line interface in the same window, 
where you can press the keys `0` or `q` or `Q` (followed by `Enter`) to shut down all servers and quit!
 - This is really convenient to use, and you can view all the logs from one place, but if you want individual control over the different services, you would like the manual mode!

## Manual Mode
### How to run:
#### via terminal
 - Open 6 terminal windows, and navigate to the root of the repository in them

 - Type these commands in the different windows to start the individual services:\
auth service:   `go run cmd/authrunner/authrunner.go` \
data service:   `go run cmd/datarunner/datarunner.go` \
config service: `go run cmd/configrunner/configrunner.go` \
profile service: `go run cmd/profilerunner/profilerunner.go` \
stats service: `go run cmd/statsrunner/statsrunner.go` \
gameplay service: `go run cmd/gameplayrunner/gameplayrunner.go`

#### via IDE (like Goland)
Open the project in an IDE, navigate to those 6 files above and press play on the main functions

### what is **Manual** mode?
- This mode gives you individual control over each service, and you can turn them on/off (via ctrl+c or closing the terminal window) to see the effects of individual services going up / down!
- It also separates the logs in different windows, which can help with monitoring!

### All the information you need to run is above, and the following is just more context about the various details!


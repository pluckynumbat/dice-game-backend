# Dice Game Backend

Welcome to the dice game backend! \
This is a microservice based architecture with 6 core services \
It is meant to be used along with the [client repository](https://github.com/pluckynumbat/dice-game-client)

## Getting Started
1. Prerequisites: go version 1.24 or higher
2. Clone this repo, and navigate to the root of it!

---
## Part 1. Running the application
There are 2 ways to run the backend, **All In One** mode or **Manual** mode, I'll explain both in turn below:

---
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

---
## Manual Mode
### How to run:
#### via terminal
 - Open 6 terminal tabs / windows, and navigate to the root of the repository in them

 - Type these commands in the different tabs / windows to start the individual services:\
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

---
### All the information you need to run is above, and the following is just more context about the various details!

---
## Part 2. More Settings

### Constants:
The [constants](https://github.com/pluckynumbat/dice-game-backend/blob/main/internal/shared/constants/constants.go) file (located at `project-root/internal/shared/constants.go`) holds settings like port numbers for the services which you might want to change if needed.
If changed, the [constants file in the client repo](https://github.com/pluckynumbat/dice-game-client/blob/main/Assets/Scripts/Constants.cs) should also be changed in the same way.

### Config:
The [config](https://github.com/pluckynumbat/dice-game-backend/blob/main/internal/config/config.go#L44) is hard coded and located in the config service, here: `project-root/internal/config/config.go`. Feel free to change that! One of the unit tests for the config service runs a validation check on the hard coded config which you can run to make sure the values are reasonable.

### About Tests:
Some tests have spinning up services as part of their setup. Please make sure that the backend is not running if / when you need to run tests. (It will fail since the port is already in use)

---
## Part 3. Additional information about the services

### The [auth](https://github.com/pluckynumbat/dice-game-backend/blob/main/internal/auth/auth.go) service (always critical):
 - This deals with the authenticating the player and managing user sessions.
 - It holds credentials, sessions, and active player IDs as maps.
 - This service also acts as the session based request validator for other services (except for data service).
 - **Important**: If this service goes down and then is restarted, player has to go through the login flow again, but the progression is not lost (that depends on the data service) 
 - **Bonus**: This service runs a session sweeper which checks the sessions map every `6` hours, and deletes sessions that have not been interacted with for `24` hours! Those settings are constants in the auth service file, and can be changed [there](https://github.com/pluckynumbat/dice-game-backend/blob/main/internal/auth/auth.go#L21) if needed!

**Public Endpoints:** login (Post), logout (Delete) \
**Internal Endpoints:** validation-internal (Post)
---
### The [data](https://github.com/pluckynumbat/dice-game-backend/blob/main/internal/data/data.go) service (always critical):
- This is the storage service for the backend. 
- It stores player data and player stats as `playersDB` and `statsDB` (both are in memory maps)
- All requests to this server are internal (only come from other servers in the backend)
- **Important**: If this service goes down and then is restarted, the player data and stats (all progression) are lost.

**Internal Endpoints:** player-internal (Post), player-internal/{id} (Get), stats-internal (Post), stats-internal/{id} (Get)

---
### The [config](https://github.com/pluckynumbat/dice-game-backend/blob/main/internal/config/config.go) service (critical for client startup):
- This provides a wrapper over the config, which can be used directly by other services.
- The client accesses the config at startup via the public get config request.

**Public Endpoints:**  game-config (Get)

---
### The [profile](https://github.com/pluckynumbat/dice-game-backend/blob/main/internal/profile/profile.go) service (critical for client startup, and during gameplay):
- This service provides all functionality related to retrieving, updating, and returning the player's dynamic data like level and energy.
- It handles new player / get player requests from the client, and sends internal requests to the data service to read / write to the `playersDB`.
- It also gets internal requests from the gameplay service.

**Public Endpoints:**  new-player (Post), player-data/{id} (Get) \
**Internal Endpoints:** player-data-internal (Put)

---
### The [stats](https://github.com/pluckynumbat/dice-game-backend/blob/main/internal/stats/stats.go) service (critical for client startup, and during gameplay):
- This service provides all functionality related to retrieving, updating, and returning the player's historic data for each level they have played (like win count, loss count, and best score).
- It handles get stats requests from the client, and sends internal requests to the data service to read / write to the `statsDB`.
- It also gets internal requests from the gameplay service.

**Public Endpoints:** player-stats/{id} (Get) \
**Internal Endpoints:** player-stats-internal (Post)

---
### The [gameplay](https://github.com/pluckynumbat/dice-game-backend/blob/main/internal/gameplay/gameplay.go) service (critical during gameplay):
- This service provides all functionality related to gameplay aspects like entering a level, getting the level results, and updating the player's live data and stats based on that.
- It handles gameplay requests from the client, and sends internal requests to the profile and stats services.

**Public Endpoints:** entry (Post), result (Post)

---
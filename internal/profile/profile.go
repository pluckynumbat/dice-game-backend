// Package profile: service which deals with the player data
package profile

const defaultLevel = 1
const maxEnergy = 10

type PlayerData struct {
	PlayerID string `json:"playerID"`
	Level    int32  `json:"level"`
	Energy   int32  `json:"energy"`
}
// Package types contains types shared by services
// currently used for data that previously only in profile and stats, but is now used in data service as well
package types

// NewPlayerRequestBody just contains the player ID
type NewPlayerRequestBody struct {
	PlayerID string `json:"playerID"`
}

// PlayerIDLevelEnergy is used as a request body for the internal request to
// update players data and return them
type PlayerIDLevelEnergy struct {
	PlayerID    string `json:"playerID"`
	Level       int32  `json:"level"`
	EnergyDelta int32  `json:"energyDelta"`
}

// PlayerIDLevelStats is used as a request body for the internal request to
// update players stats and return them
type PlayerIDLevelStats struct {
	PlayerID        string           `json:"playerID"`
	LevelStatsDelta PlayerLevelStats `json:"levelStatsDelta"`
}

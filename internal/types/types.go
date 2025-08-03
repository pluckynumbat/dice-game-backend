// Package types contains types shared by services
// currently used for data that previously only in profile and stats, but is now used in data service as well
package types

// NewPlayerRequestBody just contains the player ID
type NewPlayerRequestBody struct {
	PlayerID string `json:"playerID"`
}

// PlayerData (response struct for the client requests and request body for write requests to the data service)
// stores player related live data like level , energy etc.
type PlayerData struct {
	PlayerID       string `json:"playerID"`
	Level          int32  `json:"level"`
	Energy         int32  `json:"energy"`
	LastUpdateTime int64  `json:"lastUpdateTime"`
}

// PlayerIDLevelEnergy is used as a request body for the internal request to
// update players data and return them
type PlayerIDLevelEnergy struct {
	PlayerID    string `json:"playerID"`
	Level       int32  `json:"level"`
	EnergyDelta int32  `json:"energyDelta"`
}

// PlayerLevelStats are for a given level for a given player
type PlayerLevelStats struct {
	Level     int32 `json:"level"`
	WinCount  int32 `json:"winCount"`
	LossCount int32 `json:"lossCount"`
	BestScore int32 `json:"bestScore"`
}

// PlayerStats are for all levels for a given player
type PlayerStats struct {
	LevelStats []PlayerLevelStats `json:"levelStats"`
}

// PlayerStatsWithID is used as the client response for the public get stats api
// and as the request body for the internal request to the data service to write stats to the DB
type PlayerStatsWithID struct {
	PlayerID    string      `json:"playerID"`
	PlayerStats PlayerStats `json:"playerStats"`
}

// PlayerIDLevelStats is used as a request body for the internal request to
// update players stats and return them
type PlayerIDLevelStats struct {
	PlayerID        string           `json:"playerID"`
	LevelStatsDelta PlayerLevelStats `json:"levelStatsDelta"`
}

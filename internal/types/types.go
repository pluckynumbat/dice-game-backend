// Package types contains types shared by services
// currently used for data that previously only in profile and stats, but is now used in data service as well
package types

// PlayerIDLevelStats is used as a request body for the internal request to
// update players stats and return them
type PlayerIDLevelStats struct {
	PlayerID        string           `json:"playerID"`
	LevelStatsDelta PlayerLevelStats `json:"levelStatsDelta"`
}

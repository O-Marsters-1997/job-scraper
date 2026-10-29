package dto

import "time"

// PlaygroundCandidate is one scored Job in the playground window, with its
// band and whether it is Reviewed.
type PlaygroundCandidate struct {
	JobID             string
	Band              string
	FirstDiscoveredAt time.Time
	Reviewed          bool
}

// PlaygroundQueue is the Jobs left to review, in review order, with per-band progress.
type PlaygroundQueue struct {
	JobIDs []string         `json:"jobIds"`
	Bands  []PlaygroundBand `json:"bands"`
}

type PlaygroundBand struct {
	Band      string `json:"band"`
	Reviewed  int    `json:"reviewed"`
	Remaining int    `json:"remaining"`
}

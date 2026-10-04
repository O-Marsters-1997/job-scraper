package dto

import "time"

type ScoringOption struct {
	ID        string     `json:"id"`
	Dimension Dimension  `json:"dimension"`
	Label     string     `json:"label"`
	Level     int        `json:"level,omitempty"`
	Question  string     `json:"-"`
	RetiredAt *time.Time `json:"-"`
}

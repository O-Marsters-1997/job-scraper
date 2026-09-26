package dto

import "time"

type ScoringOption struct {
	ID        string     `json:"id"`
	Dimension Dimension  `json:"dimension"`
	Label     string     `json:"label"`
	Question  string     `json:"-"`
	RetiredAt *time.Time `json:"-"`
}

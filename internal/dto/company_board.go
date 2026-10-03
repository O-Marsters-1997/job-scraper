package dto

import "time"

type BoardStatus string

const (
	BoardCandidate BoardStatus = "candidate"
	BoardVerified  BoardStatus = "verified"
	BoardRetired   BoardStatus = "retired"
)

type CompanyBoard struct {
	ID                 string
	CompanyID          string
	Source             string
	BoardToken         string
	Status             BoardStatus
	VerificationMethod string
	DiscoveredVia      string
	VerifiedAt         *time.Time
	LastLinkedAt       *time.Time
	RetiredAt          *time.Time
	LastCompletedAt    *time.Time
	CreatedAt          time.Time
}

// CardBoard is a verified Board of the Company a discovery card names.
type CardBoard struct {
	CompanySlug string
	Source      string
	BoardToken  string
	Tracked     bool
}

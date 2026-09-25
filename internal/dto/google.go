package dto

import "time"

// GoogleToken holds an encrypted OAuth token row as returned from the DB.
type GoogleToken struct {
	AccessTokenEnc  string
	RefreshTokenEnc string
	TokenType       string
	Expiry          time.Time
	Scope           string
}

// GoogleStatus is the wire shape for GET /google/status.
type GoogleStatus struct {
	Connected bool   `json:"connected"`
	Email     string `json:"email,omitempty"`
}

// UpsertGoogleTokenInput carries the values needed to insert or update a
// google_oauth_tokens row.
type UpsertGoogleTokenInput struct {
	UserID          string
	AccessTokenEnc  string
	RefreshTokenEnc string
	TokenType       string
	Expiry          time.Time
	Scope           string
}

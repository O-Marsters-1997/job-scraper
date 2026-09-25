package dto

import "time"

type GoogleToken struct {
	AccessTokenEnc  string
	RefreshTokenEnc string
	TokenType       string
	Expiry          time.Time
	Scope           string
}

type GoogleStatus struct {
	Connected bool   `json:"connected"`
	Email     string `json:"email,omitempty"`
}

type UpsertGoogleTokenInput struct {
	UserID          string
	AccessTokenEnc  string
	RefreshTokenEnc string
	TokenType       string
	Expiry          time.Time
	Scope           string
}

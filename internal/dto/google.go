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
	Connected   bool   `json:"connected"`
	Email       string `json:"email,omitempty"`
	CanWrite    bool   `json:"canWrite"`
	CanEditDocs bool   `json:"canEditDocs"`
}

type UpsertGoogleTokenInput struct {
	UserID          string
	AccessTokenEnc  string
	RefreshTokenEnc string
	TokenType       string
	Expiry          time.Time
	Scope           string
}

package dto

import "time"

type User struct {
	ID           string
	Username     string
	PasswordHash string
	Email        string
}

type Session struct {
	ID        string
	UserID    string
	Username  string
	ExpiresAt time.Time
}

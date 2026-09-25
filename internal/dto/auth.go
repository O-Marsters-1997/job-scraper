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

type LoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type SignupInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type AuthUserView struct {
	Username string `json:"username"`
}

type MeView struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

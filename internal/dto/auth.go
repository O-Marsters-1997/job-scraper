package dto

import "time"

type User struct {
	ID           string
	Username     string
	PasswordHash string
	Email        string
}

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

type Session struct {
	ID        string
	UserID    string
	Username  string
	Role      string
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

// CreateUserInput is the identity facade's input for cmd/admin's CreateUser.
type CreateUserInput struct {
	Username     string
	PasswordHash string
	Email        string
}

type AuthUserView struct {
	Username string `json:"username"`
}

type MeView struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	IsAdmin  bool   `json:"isAdmin"`
}

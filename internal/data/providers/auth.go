package providers

import (
	"context"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// UserProvider is the single access point for user persistence.
// Callers never import pgsqlc or pgtype directly.
type UserProvider interface {
	GetUserByUsername(ctx context.Context, username string) (dto.User, error)
	CreateUser(ctx context.Context, username, passwordHash string) (dto.User, error)
}

// SessionProvider is the single access point for session persistence.
type SessionProvider interface {
	CreateSession(ctx context.Context, userID string, expiresAt time.Time) (dto.Session, error)
	GetSession(ctx context.Context, id string) (dto.Session, error)
	DeleteSession(ctx context.Context, id string) error
	DeleteExpiredSessions(ctx context.Context) error
}

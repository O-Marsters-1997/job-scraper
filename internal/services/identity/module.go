// Package identity is the identity context: users, sessions, and the
// cookie-based auth routes (ADR 0011).
package identity

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/identity/store"
)

type Module struct {
	store   *store.Store
	service *Service
}

func New(pool *pgxpool.Pool, seeder StatusSeeder) *Module {
	st := store.New(pool)
	return &Module{store: st, service: NewService(st, seeder)}
}

// Middleware authenticates requests using the session_id cookie.
func (m *Module) Middleware() func(http.Handler) http.Handler {
	return sessionMiddleware(m.store)
}

// DeleteExpiredSessions is called by the worker's daily cleanup.
func (m *Module) DeleteExpiredSessions(ctx context.Context) error {
	return m.store.DeleteExpiredSessions(ctx)
}

// CreateUser is called by cmd/admin's create-user command; it does not seed
// default application statuses or start a session.
func (m *Module) CreateUser(ctx context.Context, in dto.CreateUserInput) (dto.User, error) {
	return m.store.CreateUser(ctx, in.Username, in.PasswordHash, in.Email)
}

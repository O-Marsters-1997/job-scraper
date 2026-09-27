// Package identity is the identity context: users, sessions, profile, AI
// credentials and the Google Link (ADR 0011).
package identity

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/aicredentials"
	"github.com/ollymarsters/job-scraper/internal/services/aiprefs"
	"github.com/ollymarsters/job-scraper/internal/services/google"
	"github.com/ollymarsters/job-scraper/internal/services/identity/store"
	"github.com/ollymarsters/job-scraper/internal/services/profile"
)

type Module struct {
	store         *store.Store
	service       *Service
	aiCredentials *aicredentials.Service
	aiPrefs       *aiprefs.Service
	profile       *profile.Service
	google        *google.Service
	googleClient  *google.Client
}

// New wires the full identity context, including the Google Link and AI
// credentials.
func New(pool *pgxpool.Pool, seeder StatusSeeder, googleClientID, googleClientSecret, googleRedirectURL string) (*Module, error) {
	st := store.New(pool)

	aiCreds, err := aicredentials.New(st)
	if err != nil {
		return nil, fmt.Errorf("identity.New: %w", err)
	}

	googleClient := google.NewClient(googleClientID, googleClientSecret, googleRedirectURL, st)

	return &Module{
		store:         st,
		service:       NewService(st, seeder),
		aiCredentials: aiCreds,
		aiPrefs:       aiprefs.New(aiCreds),
		profile:       profile.New(st),
		google:        google.NewService(googleClient),
		googleClient:  googleClient,
	}, nil
}

// NewFacade wires only identity's user/session store, for cmd/admin and the
// worker's daily cleanup (ADR 0011). Routes, PublicRoutes and the Google/AI
// facade methods panic on a Module built this way.
func NewFacade(pool *pgxpool.Pool, seeder StatusSeeder) *Module {
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

// Get returns userID's decrypted AI provider key, satisfying scoring's
// Credentials port.
func (m *Module) Get(ctx context.Context, userID, provider string) (string, error) {
	return m.aiCredentials.Get(ctx, userID, provider)
}

// GetProfile satisfies scoring's ProfileReader port.
func (m *Module) GetProfile(ctx context.Context, userID string) (dto.Profile, error) {
	return m.store.GetProfile(ctx, userID)
}

// DocsClient is identity's Google Docs/Drive surface, satisfying
// cvtemplates' and trackeddocs' own DocsClient interfaces.
func (m *Module) DocsClient() *google.Client {
	return m.googleClient
}

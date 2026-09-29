// Package identity is the identity context: users, sessions, profile, AI
// credentials and the Google Link (ADR 0011).
package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/google"
	"github.com/ollymarsters/job-scraper/internal/services/identity/store"
)

type googleClient interface {
	AuthURL(state string) string
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)
	SaveToken(ctx context.Context, userID string, tok *oauth2.Token) error
	HTTPClientForUser(ctx context.Context, userID string) (*http.Client, error)
	DeleteToken(ctx context.Context, userID string) error
	ListTabs(ctx context.Context, userID, docID string) ([]google.Tab, error)
	FileMeta(ctx context.Context, userID, docID string) (google.FileMeta, error)
	GetDocument(ctx context.Context, userID, docID, tabID string) (json.RawMessage, error)
	ExportPDF(ctx context.Context, userID, docID, tabID string) (io.ReadCloser, error)
}

// Deps are Build's collaborators; New builds the real ones and calls Build.
// Tests call Build directly with fakes (ADR 0012).
type Deps struct {
	Store        Store
	Seeder       StatusSeeder
	GoogleClient googleClient
}

type Module struct {
	store        Store
	service      *Service
	google       *google.Service
	googleClient googleClient
}

func Build(deps Deps) (*Module, error) {
	credKey, err := credentialKeyFromEnv()
	if err != nil {
		return nil, fmt.Errorf("identity.Build: %w", err)
	}
	return &Module{
		store:        deps.Store,
		service:      NewService(deps.Store, deps.Seeder, credKey),
		google:       google.NewService(deps.GoogleClient),
		googleClient: deps.GoogleClient,
	}, nil
}

// New wires the full identity context, including the Google Link and AI
// credentials.
func New(pool *pgxpool.Pool, seeder StatusSeeder, googleClientID, googleClientSecret, googleRedirectURL string) (*Module, error) {
	st := store.New(pool)
	googleClient := google.NewClient(googleClientID, googleClientSecret, googleRedirectURL, st)
	return Build(Deps{
		Store:        st,
		Seeder:       seeder,
		GoogleClient: googleClient,
	})
}

// NewFacade wires only identity's user/session store, for cmd/admin and the
// worker's daily cleanup (ADR 0011). Routes, PublicRoutes and the Google/AI
// facade methods panic on a Module built this way.
func NewFacade(pool *pgxpool.Pool, seeder StatusSeeder) *Module {
	st := store.New(pool)
	return &Module{store: st, service: NewService(st, seeder, nil)}
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
	return m.service.GetCredential(ctx, userID, provider)
}

// GetProfile satisfies scoring's ProfileReader port.
func (m *Module) GetProfile(ctx context.Context, userID string) (dto.Profile, error) {
	return m.store.GetProfile(ctx, userID)
}

func (m *Module) DocsClient() googleClient {
	return m.googleClient
}

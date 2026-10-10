// Package identity is the identity context: users, sessions, profile, AI
// credentials and the Google Link (ADR 0011).
package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/google"
	"github.com/ollymarsters/job-scraper/internal/services/identity/store"
	"github.com/ollymarsters/job-scraper/internal/tokencrypt"
)

type googleClient interface {
	AuthURL(state string, write bool) string
	HasScope(ctx context.Context, userID, scope string) (bool, error)
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)
	SaveToken(ctx context.Context, userID string, tok *oauth2.Token) error
	HTTPClientForUser(ctx context.Context, userID string) (*http.Client, error)
	DeleteToken(ctx context.Context, userID string) error
	ListTabs(ctx context.Context, userID, docID string) ([]google.Tab, error)
	FileMeta(ctx context.Context, userID, docID string) (google.FileMeta, error)
	GetDocument(ctx context.Context, userID, docID, tabID string) (json.RawMessage, error)
	ExportPDF(ctx context.Context, userID, docID, tabID string) (io.ReadCloser, error)
	CopyFile(ctx context.Context, userID, fileID, name string) (string, error)
	BatchUpdate(ctx context.Context, userID, docID string, requests []json.RawMessage) error
	DeleteFile(ctx context.Context, userID, fileID string) error
	RenameFile(ctx context.Context, userID, fileID, name string) error
}

// Deps are Build's collaborators; New builds the real ones and calls Build.
// Tests call Build directly with fakes (ADR 0012).
type Deps struct {
	Store        Store
	Seeder       StatusSeeder
	GoogleClient googleClient
	Cipher       *tokencrypt.Cipher
	KeyUsage     KeyUsageFetcher
	Now          func() time.Time
}

type Module struct {
	store   Store
	service *Service
}

func Build(deps Deps) *Module {
	return &Module{store: deps.Store, service: NewService(deps)}
}

// New wires the full identity context, including the Google Link and AI
// credentials.
func New(pool *pgxpool.Pool, seeder StatusSeeder, googleClientID, googleClientSecret, googleRedirectURL string) (*Module, error) {
	credCipher, err := tokencrypt.FromEnv("AI_CREDENTIAL_ENC_KEY")
	if err != nil {
		return nil, fmt.Errorf("identity.New: %w", err)
	}
	googleCipher, err := tokencrypt.FromEnv("GOOGLE_TOKEN_ENC_KEY")
	if err != nil {
		return nil, fmt.Errorf("identity.New: %w", err)
	}
	st := store.New(pool)
	return Build(Deps{
		Store:        st,
		Seeder:       seeder,
		GoogleClient: google.NewClient(googleClientID, googleClientSecret, googleRedirectURL, st, googleCipher),
		Cipher:       credCipher,
	}), nil
}

// NewFacade wires only identity's user/session store, for cmd/admin and the
// worker's daily cleanup (ADR 0011). Routes, PublicRoutes and the Google/AI
// facade methods panic on a Module built this way.
func NewFacade(pool *pgxpool.Pool, seeder StatusSeeder) *Module {
	st := store.New(pool)
	return Build(Deps{Store: st, Seeder: seeder})
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

// UserIDByUsername resolves a username to its user ID for cmd/admin's
// explicit user arguments.
func (m *Module) UserIDByUsername(ctx context.Context, username string) (string, error) {
	user, err := m.store.GetUserByUsername(ctx, username)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}

// SetRole sets username's role. It returns apperr.Invalid for a role other
// than dto.RoleUser or dto.RoleAdmin and data.ErrNotFound for an unknown user.
func (m *Module) SetRole(ctx context.Context, username, role string) error {
	if role != dto.RoleUser && role != dto.RoleAdmin {
		return apperr.Invalid(fmt.Sprintf("role must be %q or %q, got %q", dto.RoleUser, dto.RoleAdmin, role))
	}
	return m.store.SetRole(ctx, username, role)
}

// ResetPassword replaces username's password. It returns apperr.Invalid for an
// empty password and data.ErrNotFound for an unknown user.
func (m *Module) ResetPassword(ctx context.Context, username, password string) error {
	return m.service.ResetPassword(ctx, username, password)
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
	return m.service.google
}

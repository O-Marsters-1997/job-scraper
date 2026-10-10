package identity

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/identity/store"
	"github.com/ollymarsters/job-scraper/internal/tokencrypt"
)

const sessionTTL = 30 * 24 * time.Hour

var bcryptCost = bcrypt.DefaultCost

type Store interface {
	GetUserByUsername(ctx context.Context, username string) (dto.User, error)
	CreateUser(ctx context.Context, username, passwordHash, email string) (dto.User, error)
	CreateUserTx(ctx context.Context, tx pgx.Tx, username, passwordHash, email string) (dto.User, error)
	Begin(ctx context.Context) (pgx.Tx, error)
	CreateSession(ctx context.Context, userID string, expiresAt time.Time) (dto.Session, error)
	GetSession(ctx context.Context, id string) (dto.Session, error)
	DeleteSession(ctx context.Context, id string) error
	DeleteExpiredSessions(ctx context.Context) error
	SetRole(ctx context.Context, username, role string) error
	GetProfile(ctx context.Context, userID string) (dto.Profile, error)
	UpdateEmail(ctx context.Context, userID, email string) (dto.Profile, error)
	UpsertUserAICredential(ctx context.Context, userID, provider, encKey string) error
	GetUserAICredential(ctx context.Context, userID, provider string) (string, error)
	DeleteUserAICredential(ctx context.Context, userID, provider string) error
	ListUserAICredentialProviders(ctx context.Context, userID string) ([]string, error)
}

// StatusSeeder seeds a new user's default application Statuses inside tx;
// the applications Module satisfies this (ADR 0011).
type StatusSeeder interface {
	SeedDefaults(ctx context.Context, tx pgx.Tx, userID string) error
}

type Service struct {
	store  Store
	seeder StatusSeeder
	cipher *tokencrypt.Cipher
	google googleClient

	keyUsage KeyUsageFetcher
	now      func() time.Time
	usage    aiUsageCache
}

func NewService(deps Deps) *Service {
	s := &Service{
		store: deps.Store, seeder: deps.Seeder, cipher: deps.Cipher, google: deps.GoogleClient,
		keyUsage: deps.KeyUsage, now: deps.Now,
	}
	if s.keyUsage == nil {
		s.keyUsage = newHTTPKeyUsage()
	}
	if s.now == nil {
		s.now = time.Now
	}
	return s
}

func (s *Service) Login(ctx context.Context, username, password string) (dto.Session, dto.User, error) {
	user, err := s.store.GetUserByUsername(ctx, username)
	if err != nil {
		return dto.Session{}, dto.User{}, apperr.Unauthorized("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return dto.Session{}, dto.User{}, apperr.Unauthorized("invalid credentials")
	}
	session, err := s.store.CreateSession(ctx, user.ID, time.Now().Add(sessionTTL))
	return session, user, err
}

// Signup creates a user and seeds their default application statuses in
// one transaction, then starts a session.
func (s *Service) Signup(ctx context.Context, username, password, email string) (dto.Session, dto.User, error) {
	if username == "" || password == "" {
		return dto.Session{}, dto.User{}, apperr.Invalid("bad request")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return dto.Session{}, dto.User{}, err
	}

	tx, err := s.store.Begin(ctx)
	if err != nil {
		return dto.Session{}, dto.User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	user, err := s.store.CreateUserTx(ctx, tx, username, string(hash), email)
	if err != nil {
		if errors.Is(err, store.ErrUsernameTaken) {
			return dto.Session{}, dto.User{}, apperr.Conflict(err.Error())
		}
		return dto.Session{}, dto.User{}, err
	}
	if err := s.seeder.SeedDefaults(ctx, tx, user.ID); err != nil {
		return dto.Session{}, dto.User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return dto.Session{}, dto.User{}, err
	}

	session, err := s.store.CreateSession(ctx, user.ID, time.Now().Add(sessionTTL))
	return session, user, err
}

func (s *Service) Logout(ctx context.Context, sessionID string) error {
	return s.store.DeleteSession(ctx, sessionID)
}

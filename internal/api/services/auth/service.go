package auth

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

const sessionTTL = 30 * 24 * time.Hour

var bcryptCost = bcrypt.DefaultCost

type Store interface {
	providers.UserProvider
	providers.SessionProvider
}

// StatusSeeder seeds a new user's default application Statuses. Signup
// calls it through this local interface rather than importing
// internal/applications directly (ADR 0011).
type StatusSeeder interface {
	SeedDefaults(ctx context.Context, userID string) error
}

type Service struct {
	store  Store
	seeder StatusSeeder
}

func New(store Store, seeder StatusSeeder) *Service {
	return &Service{store: store, seeder: seeder}
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

// Signup creates a user, seeds their default application statuses and
// starts a session.
func (s *Service) Signup(ctx context.Context, username, password, email string) (dto.Session, dto.User, error) {
	if username == "" || password == "" {
		return dto.Session{}, dto.User{}, apperr.Invalid("bad request")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return dto.Session{}, dto.User{}, err
	}
	user, err := s.store.CreateUser(ctx, username, string(hash), email)
	if err != nil {
		if errors.Is(err, providers.ErrUsernameTaken) {
			return dto.Session{}, dto.User{}, apperr.Conflict(err.Error())
		}
		return dto.Session{}, dto.User{}, err
	}
	if err := s.seeder.SeedDefaults(ctx, user.ID); err != nil {
		slog.Error("seed default statuses failed", slog.Any("err", err))
	}

	session, err := s.store.CreateSession(ctx, user.ID, time.Now().Add(sessionTTL))
	return session, user, err
}

func (s *Service) Logout(ctx context.Context, sessionID string) error {
	return s.store.DeleteSession(ctx, sessionID)
}

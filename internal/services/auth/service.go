// Package auth holds the domain rules for signing up, logging in and out:
// password hashing, username uniqueness and session creation. Persistence
// goes through the Store port declared here. Cookie handling stays in the
// handler; it's HTTP mechanics, not a domain rule.
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

// sessionTTL is how long a session lasts after login or signup.
const sessionTTL = 30 * 24 * time.Hour

// bcryptCost is the work factor for hashing passwords. Overridden to
// bcrypt.MinCost in tests.
var bcryptCost = bcrypt.DefaultCost

type Store interface {
	providers.UserProvider
	providers.SessionProvider
	SeedDefaultStatuses(ctx context.Context, userID string) error
}

type Service struct {
	store Store
}

func New(store Store) *Service {
	return &Service{store: store}
}

// Login verifies the given credentials and starts a session.
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
	// Best-effort: a signed-up user without seeded statuses can still add
	// their own, so this doesn't fail the signup.
	if err := s.store.SeedDefaultStatuses(ctx, user.ID); err != nil {
		slog.Error("seed default statuses failed", slog.Any("err", err))
	}

	session, err := s.store.CreateSession(ctx, user.ID, time.Now().Add(sessionTTL))
	return session, user, err
}

func (s *Service) Logout(ctx context.Context, sessionID string) error {
	return s.store.DeleteSession(ctx, sessionID)
}

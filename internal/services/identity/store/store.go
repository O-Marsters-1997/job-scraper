// Package store is the identity context's Postgres store: users, sessions,
// profile, AI credentials and the Google Link's OAuth tokens.
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/google"
	"github.com/ollymarsters/job-scraper/internal/services/identity/store/sqlc"
)

var ErrUsernameTaken = apperr.Conflict("username already taken")

type Store struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, queries: sqlc.New(pool)}
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (dto.User, error) {
	u, err := s.queries.GetUserByUsername(ctx, username)
	if err != nil {
		return dto.User{}, data.QueryErr("GetUserByUsername", err)
	}
	return toUserDTO(u), nil
}

func (s *Store) CreateUser(ctx context.Context, username, passwordHash, email string) (dto.User, error) {
	return createUser(ctx, s.queries, username, passwordHash, email)
}

// CreateUserTx runs CreateUser inside tx, so signup can roll it back
// alongside applications.SeedDefaults on seeding failure (ADR 0011).
func (s *Store) CreateUserTx(ctx context.Context, tx pgx.Tx, username, passwordHash, email string) (dto.User, error) {
	return createUser(ctx, s.queries.WithTx(tx), username, passwordHash, email)
}

func createUser(ctx context.Context, q *sqlc.Queries, username, passwordHash, email string) (dto.User, error) {
	u, err := q.CreateUser(ctx, sqlc.CreateUserParams{
		Username:     username,
		PasswordHash: passwordHash,
		Email:        data.Text(email),
	})
	if err != nil {
		if data.IsUniqueViolation(err) {
			return dto.User{}, ErrUsernameTaken
		}
		return dto.User{}, fmt.Errorf("store.CreateUser: %w", err)
	}
	return toUserDTO(u), nil
}

// Begin starts a transaction for signup, which must create the user and
// seed their default Statuses atomically (ADR 0011).
func (s *Store) Begin(ctx context.Context) (pgx.Tx, error) {
	return s.pool.Begin(ctx)
}

func (s *Store) CreateSession(ctx context.Context, userID string, expiresAt time.Time) (dto.Session, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.Session{}, err
	}
	sess, err := s.queries.CreateSession(ctx, sqlc.CreateSessionParams{
		UserID:    uid,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return dto.Session{}, fmt.Errorf("store.CreateSession: %w", err)
	}
	return toSessionDTO(sess), nil
}

func (s *Store) GetSession(ctx context.Context, id string) (dto.Session, error) {
	sid, err := data.UUID(id)
	if err != nil {
		return dto.Session{}, err
	}
	row, err := s.queries.GetSession(ctx, sid)
	if err != nil {
		return dto.Session{}, data.QueryErr("GetSession", err)
	}
	return toSessionRowDTO(row), nil
}

func (s *Store) DeleteSession(ctx context.Context, id string) error {
	sid, err := data.UUID(id)
	if err != nil {
		return err
	}
	if err := s.queries.DeleteSession(ctx, sid); err != nil {
		return fmt.Errorf("store.DeleteSession: %w", err)
	}
	return nil
}

func (s *Store) DeleteExpiredSessions(ctx context.Context) error {
	if err := s.queries.DeleteExpiredSessions(ctx); err != nil {
		return fmt.Errorf("store.DeleteExpiredSessions: %w", err)
	}
	return nil
}

func (s *Store) GetProfile(ctx context.Context, userID string) (dto.Profile, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.Profile{}, err
	}
	row, err := s.queries.GetUserProfile(ctx, uid)
	if err != nil {
		return dto.Profile{}, data.QueryErr("GetProfile", err)
	}
	return toProfileDTO(row.Username, row.Email), nil
}

func (s *Store) UpdateEmail(ctx context.Context, userID, email string) (dto.Profile, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.Profile{}, err
	}
	row, err := s.queries.UpdateUserEmail(ctx, sqlc.UpdateUserEmailParams{
		ID:    uid,
		Email: data.Text(email),
	})
	if err != nil {
		return dto.Profile{}, data.QueryErr("UpdateEmail", err)
	}
	return toProfileDTO(row.Username, row.Email), nil
}

func (s *Store) UpsertUserAICredential(ctx context.Context, userID, provider, encKey string) error {
	uid, err := data.UUID(userID)
	if err != nil {
		return err
	}
	if _, err := s.queries.UpsertUserAICredential(ctx, sqlc.UpsertUserAICredentialParams{
		UserID:    uid,
		Provider:  provider,
		ApiKeyEnc: encKey,
	}); err != nil {
		return fmt.Errorf("store.UpsertUserAICredential: %w", err)
	}
	return nil
}

func (s *Store) GetUserAICredential(ctx context.Context, userID, provider string) (string, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return "", err
	}
	row, err := s.queries.GetUserAICredential(ctx, sqlc.GetUserAICredentialParams{
		UserID:   uid,
		Provider: provider,
	})
	if err != nil {
		return "", data.QueryErr("GetUserAICredential", err)
	}
	return row.ApiKeyEnc, nil
}

func (s *Store) DeleteUserAICredential(ctx context.Context, userID, provider string) error {
	uid, err := data.UUID(userID)
	if err != nil {
		return err
	}
	if err := s.queries.DeleteUserAICredential(ctx, sqlc.DeleteUserAICredentialParams{
		UserID:   uid,
		Provider: provider,
	}); err != nil {
		return fmt.Errorf("store.DeleteUserAICredential: %w", err)
	}
	return nil
}

func (s *Store) ListUserAICredentialProviders(ctx context.Context, userID string) ([]string, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListUserAICredentialProviders(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListUserAICredentialProviders: %w", err)
	}
	return rows, nil
}

func (s *Store) GetGoogleToken(ctx context.Context, userID string) (dto.GoogleToken, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.GoogleToken{}, err
	}
	row, err := s.queries.GetGoogleOAuthToken(ctx, uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.GoogleToken{}, google.ErrTokenNotFound
	}
	if err != nil {
		return dto.GoogleToken{}, fmt.Errorf("store.GetGoogleToken: %w", err)
	}
	return toGoogleTokenDTO(row), nil
}

func (s *Store) UpsertGoogleToken(ctx context.Context, input dto.UpsertGoogleTokenInput) error {
	uid, err := data.UUID(input.UserID)
	if err != nil {
		return err
	}
	var expiry pgtype.Timestamptz
	if !input.Expiry.IsZero() {
		expiry = pgtype.Timestamptz{Time: input.Expiry, Valid: true}
	}
	if err := s.queries.UpsertGoogleOAuthToken(ctx, sqlc.UpsertGoogleOAuthTokenParams{
		UserID:          uid,
		AccessTokenEnc:  input.AccessTokenEnc,
		RefreshTokenEnc: input.RefreshTokenEnc,
		TokenType:       input.TokenType,
		Expiry:          expiry,
		Scope:           input.Scope,
	}); err != nil {
		return fmt.Errorf("store.UpsertGoogleToken: %w", err)
	}
	return nil
}

func (s *Store) DeleteGoogleToken(ctx context.Context, userID string) error {
	uid, err := data.UUID(userID)
	if err != nil {
		return err
	}
	if err := s.queries.DeleteGoogleOAuthToken(ctx, uid); err != nil {
		return fmt.Errorf("store.DeleteGoogleToken: %w", err)
	}
	return nil
}

// Package store is the identity context's Postgres store: users and
// sessions.
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/identity/store/sqlc"
)

var (
	ErrNotFound      = apperr.NotFound("not found")
	ErrUsernameTaken = apperr.Conflict("username already taken")
)

type Store struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, queries: sqlc.New(pool)}
}

func parseUUID(s string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(s); err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid uuid %q: %w", s, err)
	}
	return id, nil
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (dto.User, error) {
	u, err := s.queries.GetUserByUsername(ctx, username)
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.User{}, ErrNotFound
	}
	if err != nil {
		return dto.User{}, fmt.Errorf("store.GetUserByUsername: %w", err)
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
		Email:        pgtype.Text{String: email, Valid: email != ""},
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
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
	uid, err := parseUUID(userID)
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
	sid, err := parseUUID(id)
	if err != nil {
		return dto.Session{}, err
	}
	row, err := s.queries.GetSession(ctx, sid)
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.Session{}, ErrNotFound
	}
	if err != nil {
		return dto.Session{}, fmt.Errorf("store.GetSession: %w", err)
	}
	return toSessionRowDTO(row), nil
}

func (s *Store) DeleteSession(ctx context.Context, id string) error {
	sid, err := parseUUID(id)
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

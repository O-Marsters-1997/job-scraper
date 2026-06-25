package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data/db/pgsqlc"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func fromUser(u pgsqlc.User) dto.User {
	return dto.User{
		ID:           u.ID.String(),
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		Email:        u.Email.String,
	}
}

func fromSession(s pgsqlc.Session) dto.Session {
	return dto.Session{
		ID:        s.ID.String(),
		UserID:    s.UserID.String(),
		ExpiresAt: s.ExpiresAt.Time,
	}
}

func fromSessionRow(r pgsqlc.GetSessionRow) dto.Session {
	return dto.Session{
		ID:        r.ID.String(),
		UserID:    r.UserID.String(),
		Username:  r.Username,
		ExpiresAt: r.ExpiresAt.Time,
	}
}

func parseUUID(s string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(s); err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid uuid %q: %w", s, err)
	}
	return id, nil
}

func (db *DB) GetUserByUsername(ctx context.Context, username string) (dto.User, error) {
	u, err := db.queries.GetUserByUsername(ctx, username)
	if err != nil {
		return dto.User{}, fmt.Errorf("db.GetUserByUsername: %w", err)
	}
	return fromUser(u), nil
}

func (db *DB) CreateUser(ctx context.Context, username, passwordHash, email string) (dto.User, error) {
	u, err := db.queries.CreateUser(ctx, pgsqlc.CreateUserParams{
		Username:     username,
		PasswordHash: passwordHash,
		Email:        pgtype.Text{String: email, Valid: email != ""},
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return dto.User{}, providers.ErrUsernameTaken
		}
		return dto.User{}, fmt.Errorf("db.CreateUser: %w", err)
	}
	return fromUser(u), nil
}

func (db *DB) CreateSession(ctx context.Context, userID string, expiresAt time.Time) (dto.Session, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return dto.Session{}, err
	}
	s, err := db.queries.CreateSession(ctx, pgsqlc.CreateSessionParams{
		UserID:    uid,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return dto.Session{}, fmt.Errorf("db.CreateSession: %w", err)
	}
	return fromSession(s), nil
}

func (db *DB) GetSession(ctx context.Context, id string) (dto.Session, error) {
	uid, err := parseUUID(id)
	if err != nil {
		return dto.Session{}, err
	}
	row, err := db.queries.GetSession(ctx, uid)
	if err != nil {
		return dto.Session{}, fmt.Errorf("db.GetSession: %w", err)
	}
	return fromSessionRow(row), nil
}

func (db *DB) DeleteSession(ctx context.Context, id string) error {
	uid, err := parseUUID(id)
	if err != nil {
		return err
	}
	if err := db.queries.DeleteSession(ctx, uid); err != nil {
		return fmt.Errorf("db.DeleteSession: %w", err)
	}
	return nil
}

func (db *DB) DeleteExpiredSessions(ctx context.Context) error {
	if err := db.queries.DeleteExpiredSessions(ctx); err != nil {
		return fmt.Errorf("db.DeleteExpiredSessions: %w", err)
	}
	return nil
}

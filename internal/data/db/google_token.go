package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/oauth2"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/tokencrypt"
)

// GetGoogleToken retrieves and decrypts the stored OAuth token for the user.
func (db *DB) GetGoogleToken(ctx context.Context, userID string) (dto.GoogleToken, error) {
	const q = `
		SELECT access_token_enc, refresh_token_enc, token_type, expiry, scope
		FROM google_oauth_tokens
		WHERE user_id = $1`

	var (
		accessEnc  string
		refreshEnc string
		tokenType  string
		expiry     pgtype.Timestamptz
		scope      string
	)

	err := db.pool.QueryRow(ctx, q, userID).Scan(
		&accessEnc, &refreshEnc, &tokenType, &expiry, &scope,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dto.GoogleToken{}, providers.ErrGoogleTokenNotFound
		}
		return dto.GoogleToken{}, fmt.Errorf("db.GetGoogleToken: %w", err)
	}

	return dto.GoogleToken{
		AccessTokenEnc:  accessEnc,
		RefreshTokenEnc: refreshEnc,
		TokenType:       tokenType,
		Expiry:          expiry.Time,
		Scope:           scope,
	}, nil
}

// UpsertGoogleToken inserts or replaces the encrypted OAuth token for the user.
func (db *DB) UpsertGoogleToken(ctx context.Context, input dto.UpsertGoogleTokenInput) error {
	const q = `
		INSERT INTO google_oauth_tokens
		    (user_id, access_token_enc, refresh_token_enc, token_type, expiry, scope, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
		    access_token_enc  = EXCLUDED.access_token_enc,
		    refresh_token_enc = EXCLUDED.refresh_token_enc,
		    token_type        = EXCLUDED.token_type,
		    expiry            = EXCLUDED.expiry,
		    scope             = EXCLUDED.scope,
		    updated_at        = NOW()`

	var expiry pgtype.Timestamptz
	if !input.Expiry.IsZero() {
		expiry = pgtype.Timestamptz{Time: input.Expiry, Valid: true}
	}

	_, err := db.pool.Exec(ctx, q,
		input.UserID,
		input.AccessTokenEnc,
		input.RefreshTokenEnc,
		input.TokenType,
		expiry,
		input.Scope,
	)
	if err != nil {
		return fmt.Errorf("db.UpsertGoogleToken: %w", err)
	}
	return nil
}

// DeleteGoogleToken removes the stored OAuth token for the user.
func (db *DB) DeleteGoogleToken(ctx context.Context, userID string) error {
	const q = `DELETE FROM google_oauth_tokens WHERE user_id = $1`
	if _, err := db.pool.Exec(ctx, q, userID); err != nil {
		return fmt.Errorf("db.DeleteGoogleToken: %w", err)
	}
	return nil
}

// GoogleTokenStore adapts *DB to the google.TokenStore interface, handling
// encryption/decryption of access and refresh tokens.
type GoogleTokenStore struct {
	db *DB
}

// NewGoogleTokenStore wraps a *DB to implement google.TokenStore.
func NewGoogleTokenStore(db *DB) *GoogleTokenStore {
	return &GoogleTokenStore{db: db}
}

// GetToken retrieves and decrypts the OAuth token for the given user.
func (s *GoogleTokenStore) GetToken(ctx context.Context, userID string) (*oauth2.Token, error) {
	row, err := s.db.GetGoogleToken(ctx, userID)
	if err != nil {
		return nil, err
	}

	access, err := tokencrypt.Decrypt(row.AccessTokenEnc)
	if err != nil {
		return nil, fmt.Errorf("GoogleTokenStore.GetToken decrypt access: %w", err)
	}
	refresh, err := tokencrypt.Decrypt(row.RefreshTokenEnc)
	if err != nil {
		return nil, fmt.Errorf("GoogleTokenStore.GetToken decrypt refresh: %w", err)
	}

	return &oauth2.Token{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    row.TokenType,
		Expiry:       row.Expiry,
	}, nil
}

// SaveToken encrypts and persists the OAuth token for the given user.
func (s *GoogleTokenStore) SaveToken(ctx context.Context, userID string, tok *oauth2.Token) error {
	accessEnc, err := tokencrypt.Encrypt(tok.AccessToken)
	if err != nil {
		return fmt.Errorf("GoogleTokenStore.SaveToken encrypt access: %w", err)
	}
	refreshEnc, err := tokencrypt.Encrypt(tok.RefreshToken)
	if err != nil {
		return fmt.Errorf("GoogleTokenStore.SaveToken encrypt refresh: %w", err)
	}

	var expiry time.Time
	if !tok.Expiry.IsZero() {
		expiry = tok.Expiry
	}

	return s.db.UpsertGoogleToken(ctx, dto.UpsertGoogleTokenInput{
		UserID:          userID,
		AccessTokenEnc:  accessEnc,
		RefreshTokenEnc: refreshEnc,
		TokenType:       tok.TokenType,
		Expiry:          expiry,
		Scope:           "https://www.googleapis.com/auth/drive.readonly",
	})
}

// DeleteToken removes the stored OAuth token for the given user.
func (s *GoogleTokenStore) DeleteToken(ctx context.Context, userID string) error {
	return s.db.DeleteGoogleToken(ctx, userID)
}

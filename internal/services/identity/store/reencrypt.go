package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/services/identity/store/sqlc"
)

func (s *Store) ReencryptAICredentials(ctx context.Context, rewrite func(userID, provider, enc string) (string, error)) (int, error) {
	return s.sweep(ctx, s.queries.ListLegacyAICredentialIDs, func(q *sqlc.Queries, id pgtype.UUID) error {
		row, err := q.LockAICredential(ctx, id)
		if err != nil {
			return err
		}
		enc, err := rewrite(row.UserID.String(), row.Provider, row.ApiKeyEnc)
		if err != nil {
			return err
		}
		return q.SetAICredentialEnc(ctx, sqlc.SetAICredentialEncParams{ID: id, ApiKeyEnc: enc})
	})
}

func (s *Store) ReencryptGoogleTokens(ctx context.Context, rewrite func(userID, enc string) (string, error)) (int, error) {
	return s.sweep(ctx, s.queries.ListLegacyGoogleTokenIDs, func(q *sqlc.Queries, id pgtype.UUID) error {
		row, err := q.LockGoogleToken(ctx, id)
		if err != nil {
			return err
		}
		userID := row.UserID.String()
		access, err := rewrite(userID, row.AccessTokenEnc)
		if err != nil {
			return err
		}
		refresh, err := rewrite(userID, row.RefreshTokenEnc)
		if err != nil {
			return err
		}
		return q.SetGoogleTokenEnc(ctx, sqlc.SetGoogleTokenEncParams{ID: id, AccessTokenEnc: access, RefreshTokenEnc: refresh})
	})
}

func (s *Store) sweep(
	ctx context.Context,
	list func(context.Context) ([]pgtype.UUID, error),
	rewriteRow func(*sqlc.Queries, pgtype.UUID) error,
) (int, error) {
	total := 0
	for {
		ids, err := list(ctx)
		if err != nil {
			return total, fmt.Errorf("list legacy rows: %w", err)
		}
		if len(ids) == 0 {
			return total, nil
		}
		for _, id := range ids {
			err := s.inTx(ctx, func(q *sqlc.Queries) error { return rewriteRow(q, id) })
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			if err != nil {
				return total, fmt.Errorf("rewrite row %s: %w", id.String(), err)
			}
			total++
		}
	}
}

func (s *Store) inTx(ctx context.Context, fn func(*sqlc.Queries) error) (err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if rbErr := tx.Rollback(context.WithoutCancel(ctx)); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			err = errors.Join(err, rbErr)
		}
	}()
	if err := fn(s.queries.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

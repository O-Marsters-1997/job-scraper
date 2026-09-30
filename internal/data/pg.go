package data

import (
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

const uniqueViolation = "23505"

func QueryErr(op string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return fmt.Errorf("store.%s: %w", op, err)
}

func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolation
}

func UUID(s string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(s); err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid uuid %q: %w", s, err)
	}
	return id, nil
}

func UUIDs(ids []string) ([]pgtype.UUID, error) {
	out := make([]pgtype.UUID, len(ids))
	for i, s := range ids {
		id, err := UUID(s)
		if err != nil {
			return nil, err
		}
		out[i] = id
	}
	return out, nil
}

func Text(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

func TimePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}

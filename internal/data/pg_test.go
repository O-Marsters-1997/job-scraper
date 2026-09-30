package data_test

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data"
)

func TestQueryErr(t *testing.T) {
	boom := errors.New("boom")
	if err := data.QueryErr("Get", pgx.ErrNoRows); !errors.Is(err, data.ErrNotFound) {
		t.Errorf("QueryErr(ErrNoRows) = %v, want ErrNotFound", err)
	}
	err := data.QueryErr("Get", boom)
	if !errors.Is(err, boom) || err.Error() != "store.Get: boom" {
		t.Errorf("QueryErr(boom) = %v, want wrapped store.Get: boom", err)
	}
}

func TestIsUniqueViolation(t *testing.T) {
	if !data.IsUniqueViolation(&pgconn.PgError{Code: "23505"}) {
		t.Error("IsUniqueViolation(23505) = false, want true")
	}
	if data.IsUniqueViolation(&pgconn.PgError{Code: "23503"}) || data.IsUniqueViolation(errors.New("x")) {
		t.Error("IsUniqueViolation(other) = true, want false")
	}
}

func TestUUID(t *testing.T) {
	const s = "11111111-2222-3333-4444-555555555555"
	id, err := data.UUID(s)
	if err != nil || id.String() != s {
		t.Errorf("UUID(%q) = %v, %v", s, id.String(), err)
	}
	if _, err := data.UUID("nope"); err == nil {
		t.Error("UUID(nope) error = nil, want error")
	}
	if got := (pgtype.UUID{}).String(); got != "" {
		t.Errorf("invalid UUID String() = %q, want empty", got)
	}
}

func TestUUIDs(t *testing.T) {
	ids, err := data.UUIDs([]string{"11111111-2222-3333-4444-555555555555"})
	if err != nil || len(ids) != 1 || !ids[0].Valid {
		t.Errorf("UUIDs(valid) = %v, %v", ids, err)
	}
	if _, err := data.UUIDs([]string{"nope"}); err == nil {
		t.Error("UUIDs(nope) error = nil, want error")
	}
}

func TestText(t *testing.T) {
	if got := data.Text(""); got.Valid {
		t.Errorf("Text(\"\") = %+v, want NULL", got)
	}
	if got := data.Text("a"); !got.Valid || got.String != "a" {
		t.Errorf("Text(a) = %+v", got)
	}
}

func TestTimePtr(t *testing.T) {
	if got := data.TimePtr(pgtype.Timestamptz{}); got != nil {
		t.Errorf("TimePtr(NULL) = %v, want nil", got)
	}
	now := time.Now()
	if got := data.TimePtr(pgtype.Timestamptz{Time: now, Valid: true}); got == nil || !got.Equal(now) {
		t.Errorf("TimePtr(now) = %v", got)
	}
}

package data_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data"
)

const validUUID = "11111111-2222-3333-4444-555555555555"

func TestQueryErr(t *testing.T) {
	boom := errors.New("boom")
	tests := []struct {
		name string
		in   error
		want error
	}{
		{name: "no rows is ErrNotFound", in: pgx.ErrNoRows, want: data.ErrNotFound},
		{name: "other errors stay wrapped", in: boom, want: boom},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := data.QueryErr("Get", tt.in); !errors.Is(err, tt.want) {
				t.Errorf("QueryErr(%v) = %v, want %v", tt.in, err, tt.want)
			}
		})
	}
}

func TestIsUniqueViolation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "unique violation", err: &pgconn.PgError{Code: "23505"}, want: true},
		{name: "foreign key violation", err: &pgconn.PgError{Code: "23503"}},
		{name: "non-pg error", err: errors.New("x")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := data.IsUniqueViolation(tt.err); got != tt.want {
				t.Errorf("IsUniqueViolation(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestUUID(t *testing.T) {
	id, err := data.UUID(validUUID)
	if err != nil {
		t.Fatalf("UUID(%q) err = %v", validUUID, err)
	}
	if got := id.String(); got != validUUID {
		t.Errorf("UUID(%q).String() = %q", validUUID, got)
	}

	if _, err := data.UUID("nope"); err == nil {
		t.Error("UUID(nope) err = nil, want error")
	}
}

func TestUUIDs(t *testing.T) {
	ids, err := data.UUIDs([]string{validUUID})
	if err != nil {
		t.Fatalf("UUIDs(valid) err = %v", err)
	}
	if len(ids) != 1 || !ids[0].Valid {
		t.Errorf("UUIDs(valid) = %v, want one valid ID", ids)
	}

	if _, err := data.UUIDs([]string{"nope"}); err == nil {
		t.Error("UUIDs(nope) err = nil, want error")
	}
}

func TestText(t *testing.T) {
	tests := []struct {
		in   string
		want pgtype.Text
	}{
		{in: ""},
		{in: "a", want: pgtype.Text{String: "a", Valid: true}},
	}
	for _, tt := range tests {
		if diff := cmp.Diff(tt.want, data.Text(tt.in)); diff != "" {
			t.Errorf("Text(%q) (-want +got):\n%s", tt.in, diff)
		}
	}
}

func TestTimePtr(t *testing.T) {
	if got := data.TimePtr(pgtype.Timestamptz{}); got != nil {
		t.Errorf("TimePtr(NULL) = %v, want nil", got)
	}

	now := time.Now()
	got := data.TimePtr(pgtype.Timestamptz{Time: now, Valid: true})
	if got == nil || !got.Equal(now) {
		t.Errorf("TimePtr(now) = %v, want %v", got, now)
	}
}

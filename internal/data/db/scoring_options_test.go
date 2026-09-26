package db_test

import (
	"context"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestListScoringOptions(t *testing.T) {
	ctx := context.Background()

	dims := []dto.Dimension{
		dto.DimensionTech, dto.DimensionRole, dto.DimensionDomain,
		dto.DimensionSeniority, dto.DimensionWork, dto.DimensionStage,
	}
	for _, dim := range dims {
		id := "test:" + string(dim)
		if _, err := testDB.Pool().Exec(ctx, `
			INSERT INTO scoring_options (id, dimension, label, question) VALUES ($1, $2, $1, $1)
		`, id, dim); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if _, err := testDB.Pool().Exec(ctx, `DELETE FROM scoring_options WHERE id = $1`, id); err != nil {
				t.Fatalf("cleanup: %v", err)
			}
		})
	}

	options, err := testDB.ListScoringOptions(ctx)
	if err != nil {
		t.Fatal(err)
	}

	byDimension := make(map[dto.Dimension]int)
	for _, o := range options {
		byDimension[o.Dimension]++
	}
	for _, dim := range dims {
		if byDimension[dim] == 0 {
			t.Errorf("dimension %q has no options", dim)
		}
	}
}

func TestListScoringOptions_RoundTripsEnumAndRetiredAt(t *testing.T) {
	ctx := context.Background()

	_, err := testDB.Pool().Exec(ctx, `
		INSERT INTO scoring_options (id, dimension, label, question, retired_at)
		VALUES ('test:round-trip', 'seniority', 'Round trip', 'Does this round trip?', NOW())
	`)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := testDB.Pool().Exec(ctx, `DELETE FROM scoring_options WHERE id = 'test:round-trip'`); err != nil {
			t.Fatalf("cleanup: %v", err)
		}
	})

	options, err := testDB.ListScoringOptions(ctx)
	if err != nil {
		t.Fatal(err)
	}

	var found *dto.ScoringOption
	for i, o := range options {
		if o.ID == "test:round-trip" {
			found = &options[i]
			break
		}
	}
	if found == nil {
		t.Fatal("round-trip option not found")
	}
	if found.Dimension != dto.DimensionSeniority {
		t.Errorf("dimension = %q, want %q", found.Dimension, dto.DimensionSeniority)
	}
	if found.RetiredAt == nil {
		t.Error("retired_at should round-trip as non-nil")
	}
}

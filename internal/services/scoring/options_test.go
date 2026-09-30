package scoring_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/scoringtest"
)

func TestOptions(t *testing.T) {
	retired := time.Now()
	store := scoringtest.NewFakeStore()
	store.SeedOptions([]dto.ScoringOption{
		{ID: "tech:go", Dimension: dto.DimensionTech, Label: "Go", Question: "Does the role use Go?"},
		{ID: "tech:cobol", Dimension: dto.DimensionTech, Label: "COBOL", Question: "Does the role use COBOL?", RetiredAt: &retired},
	})
	svc := newService(t, store)

	got, err := svc.Options(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Dimensions) != len(scoring.Dimensions) {
		t.Fatalf("dimensions = %d, want %d", len(got.Dimensions), len(scoring.Dimensions))
	}
	want := []dto.ScoringOption{{ID: "tech:go", Dimension: dto.DimensionTech, Label: "Go"}}
	if diff := cmp.Diff(want, got.Options); diff != "" {
		t.Errorf("options mismatch, retired excluded and question text stripped (-want +got):\n%s", diff)
	}
}

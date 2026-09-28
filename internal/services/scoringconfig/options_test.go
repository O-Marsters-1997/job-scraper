package scoringconfig_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/scoringtest"
	"github.com/ollymarsters/job-scraper/internal/services/scoringconfig"
)

func TestOptions(t *testing.T) {
	retired := time.Now()
	store := scoringtest.NewFakeStore()
	store.SeedOptions([]dto.ScoringOption{
		{ID: "tech:go", Dimension: dto.DimensionTech, Label: "Go", Question: "Does the role use Go?"},
		{ID: "tech:cobol", Dimension: dto.DimensionTech, Label: "COBOL", Question: "Does the role use COBOL?", RetiredAt: &retired},
	})
	svc := scoringconfig.New(store, scoringtest.Reconsiders(), scoringtest.Recomputes(0), &fakeExtractor{}, &fakeCredentials{}, scoringtest.Backfills(0))

	got, err := svc.Options(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Dimensions) != len(scoringconfig.Dimensions) {
		t.Fatalf("dimensions = %d, want %d", len(got.Dimensions), len(scoringconfig.Dimensions))
	}
	want := []dto.ScoringOption{{ID: "tech:go", Dimension: dto.DimensionTech, Label: "Go"}}
	if diff := cmp.Diff(want, got.Options); diff != "" {
		t.Errorf("options mismatch, retired excluded and question text stripped (-want +got):\n%s", diff)
	}
}

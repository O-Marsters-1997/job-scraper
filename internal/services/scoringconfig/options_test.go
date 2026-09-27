package scoringconfig_test

import (
	"context"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoringconfig"
)

func TestOptions(t *testing.T) {
	retired := time.Now()
	store := newFakeStore()
	store.options = []dto.ScoringOption{
		{ID: "tech:go", Dimension: dto.DimensionTech, Label: "Go", Question: "Does the role use Go?"},
		{ID: "tech:cobol", Dimension: dto.DimensionTech, Label: "COBOL", Question: "Does the role use COBOL?", RetiredAt: &retired},
	}
	svc := scoringconfig.New(store, &fakeReconsiderer{}, &fakeRecomputer{}, &fakeExtractor{}, &fakeCredentials{})

	got, err := svc.Options(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Dimensions) != len(scoringconfig.Dimensions) {
		t.Fatalf("dimensions = %d, want %d", len(got.Dimensions), len(scoringconfig.Dimensions))
	}
	if len(got.Options) != 1 {
		t.Fatalf("options = %d, want 1 (retired excluded): %+v", len(got.Options), got.Options)
	}
	want := dto.ScoringOption{ID: "tech:go", Dimension: dto.DimensionTech, Label: "Go"}
	if got.Options[0] != want {
		t.Fatalf("option = %+v, want %+v (question text must not survive to the view)", got.Options[0], want)
	}
}

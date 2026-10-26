package dto_test

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

var update = flag.Bool("update", false, "rewrite the iOS fixtures under ios/FastTrackTests/Fixtures")

func TestIOSFixtures(t *testing.T) {
	score := 92
	at := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)

	fixtures := map[string]any{
		"job_page": dto.JobPage{
			Total: 1,
			Items: []dto.Job{{
				ID:               "job-1",
				Title:            "Senior Go Engineer",
				Location:         "London",
				URL:              "https://example.com/jobs/1",
				CompanySlug:      "monzo",
				ScrapedAt:        at,
				SuitabilityScore: &score,
				Band:             "Great",
				Breakdown: []dto.ScoreRow{
					{Key: "tech:go", Label: "Go", Stance: "want", Resolved: "yes", Effect: "meets"},
					{Key: "domain:fintech", Label: "Fintech", Stance: "want", Resolved: "no", Effect: "misses"},
				},
			}},
		},
	}

	for name, v := range fixtures {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("..", "..", "ios", "FastTrackTests", "Fixtures", name+".json")
			got, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				t.Fatalf("marshal %s: %v", name, err)
			}
			got = append(got, '\n')

			if *update {
				if err := os.WriteFile(path, got, 0o644); err != nil {
					t.Fatalf("write %s: %v", path, err)
				}
				return
			}

			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v (run go test ./internal/dto -run TestIOSFixtures -update)", path, err)
			}
			var want, gotDecoded any
			if err := json.Unmarshal(raw, &want); err != nil {
				t.Fatalf("decode %s: %v", path, err)
			}
			if err := json.Unmarshal(got, &gotDecoded); err != nil {
				t.Fatalf("decode marshalled %s: %v", name, err)
			}
			if diff := cmp.Diff(want, gotDecoded); diff != "" {
				t.Errorf("%s is stale; rerun with -update (-fixture +dto):\n%s", path, diff)
			}
		})
	}
}

package sourcetest

import (
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

var update = flag.Bool("update", false, "rewrite golden files from parser output")

// RunGolden serves snapshots/<fixture> as src's API response and compares the fetched jobs to
// snapshots/<fixture minus extension>.golden.json. Run with -update to rewrite it. src must
// expose its HTTP client, as every Source built on sources.PaginatedBase does.
func RunGolden(t *testing.T, fixture string, src sources.Source) {
	t.Helper()

	body, err := os.ReadFile(filepath.Join("snapshots", fixture))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	withClient, ok := src.(interface{ Client() *http.Client })
	if !ok {
		t.Fatalf("%T does not expose Client()", src)
	}
	withClient.Client().Transport = Respond(string(body))

	start := time.Now()
	got, _, err := src.FetchPage(t.Context(), "")
	if err != nil {
		t.Fatalf("FetchPage(%s) error: %v", fixture, err)
	}
	MatchGolden(t, fixture, got, start)
}

// MatchGolden compares got to snapshots/<fixture minus extension>.golden.json, ignoring
// UpdatedAt values that fell back to the time of the fetch begun at start. Run with -update
// to rewrite the golden.
func MatchGolden(t *testing.T, fixture string, got []dto.Job, start time.Time) {
	t.Helper()

	clearTimeNowFallbacks(got, start)
	goldenPath := filepath.Join("snapshots", strings.TrimSuffix(fixture, filepath.Ext(fixture))+".golden.json")
	if *update {
		out, err := json.MarshalIndent(got, "", "  ")
		if err != nil {
			t.Fatalf("marshal golden: %v", err)
		}
		if err := os.WriteFile(goldenPath, append(out, '\n'), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}

	wantBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v (run `go test -update` to create it)", goldenPath, err)
	}
	var want []dto.Job
	if err := json.Unmarshal(wantBytes, &want); err != nil {
		t.Fatalf("unmarshal golden: %v", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("FetchPage(%s) mismatch (-want +got):\n%s", fixture, diff)
	}
}

func clearTimeNowFallbacks(jobs []dto.Job, parseStart time.Time) {
	for i := range jobs {
		if !jobs[i].UpdatedAt.Before(parseStart) {
			jobs[i].UpdatedAt = time.Time{}
		}
	}
}

// PollReported serves snapshots/<fixture> as src's API response and returns the Reported
// count from one PollBoard.
func PollReported(t *testing.T, fixture string, src *sources.BoardSource) int {
	t.Helper()

	body, err := os.ReadFile(filepath.Join("snapshots", fixture))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	src.Client().Transport = Respond(string(body))

	res, err := src.PollBoard(t.Context())
	if err != nil {
		t.Fatalf("PollBoard(%s) error: %v", fixture, err)
	}
	return res.Reported
}

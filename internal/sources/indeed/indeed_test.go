package indeed_test

import (
	"os"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/sources"
	"github.com/ollymarsters/job-scraper/internal/sources/indeed"
)

// TestSnapshots runs against the placeholder fixtures in snapshots/ — see
// snapshots/README.md. They are NOT real Indeed captures.
func TestSnapshots(t *testing.T) {
	sources.RunSnapshotTests(t, indeed.New(indeed.Config{}))
}

// TestParseURLs_AntiBotBlock guards the semantic-200 check: BrightData's Web
// Unlocker returns HTTP 200 even for anti-bot interstitials, so a page with
// zero job cards and no recognizable "no results" marker must error rather
// than silently look like a fresh, empty scrape.
//
// testdata/blocked_cloudflare.html is a real page captured by this
// environment's own direct (unproxied, non-BrightData) fetch attempt against
// indeed.com, which was blocked with a Cloudflare "Additional Verification
// Required" interstitial. It's not a BrightData-specific capture, but it is a
// real anti-bot block page and exercises the same zero-cards/no-marker shape.
func TestParseURLs_AntiBotBlock(t *testing.T) {
	f, err := os.Open("testdata/blocked_cloudflare.html")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer func() { _ = f.Close() }()

	_, err = indeed.ParseURLs(f)
	if err == nil {
		t.Fatal("expected an error for a blocked/interstitial page, got nil")
	}
}

// TestParseURLs_NoResultsIsNotAnError verifies a genuine empty-search-results
// page (zero cards, but a real "no results" marker) does not error — only an
// unrecognized zero-card page should be treated as a suspected block.
func TestParseURLs_NoResultsIsNotAnError(t *testing.T) {
	const noResultsHTML = `<!DOCTYPE html><html><body>
		<div id="mosaic-provider-jobcards">
			<p>Sorry, we did not match any jobs to your search.</p>
		</div>
	</body></html>`

	jobs, err := indeed.ParseURLs(strings.NewReader(noResultsHTML))
	if err != nil {
		t.Fatalf("expected no error for a genuine no-results page, got: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("expected 0 jobs, got %d", len(jobs))
	}
}

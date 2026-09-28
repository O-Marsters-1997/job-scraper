package personio

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

// jobs_optiply.xml mirrors the current Personio feed shape
// ({token}.jobs.personio.com/xml): <workzag-jobs> of <position> elements with
// <name>, <office>, and a nested <jobDescriptions>. There is no apply-URL
// element, so the parser builds the URL from token+id and concatenates the
// description blocks.
func TestParse_Golden(t *testing.T) {
	sourcetest.RunGolden(t, "jobs_optiply.xml", "optiply", parse)
}

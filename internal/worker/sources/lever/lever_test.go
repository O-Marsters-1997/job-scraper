package lever

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

func TestParse_Golden(t *testing.T) {
	sourcetest.RunGolden(t, "postings_acme.json", "acme", parse)
}

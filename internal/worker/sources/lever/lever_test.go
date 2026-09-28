package lever_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources/lever"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

func TestFetchPage_Golden(t *testing.T) {
	sourcetest.RunGolden(t, "postings_acme.json", lever.New("acme"))
}

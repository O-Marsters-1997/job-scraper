package recruitee_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources/recruitee"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

func TestFetchPage_Golden(t *testing.T) {
	sourcetest.RunGolden(t, "offers_acme.json", recruitee.New("acme"))
}

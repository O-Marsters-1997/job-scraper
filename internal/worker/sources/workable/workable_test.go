package workable_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/workable"
)

func TestFetchPage_Golden(t *testing.T) {
	sourcetest.RunGolden(t, "jobs_pearltalent.json", workable.New("pearltalent"))
}

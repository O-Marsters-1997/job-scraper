package personio_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources/personio"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

func TestFetchPage_Golden(t *testing.T) {
	sourcetest.RunGolden(t, "jobs_optiply.xml", personio.New("optiply"))
}

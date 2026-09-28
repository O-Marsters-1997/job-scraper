package greenhouse_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources/greenhouse"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

func TestFetchPage_Golden(t *testing.T) {
	sourcetest.RunGolden(t, "board_acme.json", greenhouse.New("acme"))
}

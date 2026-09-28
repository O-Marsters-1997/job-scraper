package ashby_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources/ashby"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

func TestFetchPage_Golden(t *testing.T) {
	sourcetest.RunGolden(t, "board_askdragonfly.json", ashby.New("askdragonfly"))
}

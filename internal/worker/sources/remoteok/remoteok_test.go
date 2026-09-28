package remoteok_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources/remoteok"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

func TestFetchPage_Golden(t *testing.T) {
	sourcetest.RunGolden(t, "feed_sample.json", remoteok.New(""))
}

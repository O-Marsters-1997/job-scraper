package wis_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/wis"
)

func TestSnapshots(t *testing.T) {
	sourcetest.RunSnapshotTests(t, wis.New(wis.Config{}))
}

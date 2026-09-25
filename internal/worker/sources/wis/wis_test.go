package wis_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/wis"
)

func TestSnapshots(t *testing.T) {
	sources.RunSnapshotTests(t, wis.New(wis.Config{}))
}

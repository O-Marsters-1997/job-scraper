package wis_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/sources"
	"github.com/ollymarsters/job-scraper/internal/sources/wis"
)

func TestSnapshots(t *testing.T) {
	sources.RunSnapshotTestsURLs(t, wis.ParseURLs)
}

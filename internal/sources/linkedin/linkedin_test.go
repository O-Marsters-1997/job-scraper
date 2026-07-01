package linkedin_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/sources"
	"github.com/ollymarsters/job-scraper/internal/sources/linkedin"
)

func TestSnapshots(t *testing.T) {
	sources.RunSnapshotTests(t, linkedin.New(linkedin.Config{}))
}

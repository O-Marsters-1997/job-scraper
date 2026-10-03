package personio_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources/personio"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

func TestFetchPage_Golden(t *testing.T) {
	sourcetest.RunGolden(t, "jobs_optiply.xml", personio.New("optiply"))
}

func TestPollBoard_Reported(t *testing.T) {
	if got, want := sourcetest.PollReported(t, "jobs_optiply.xml", personio.New("optiply")), 2; got != want {
		t.Errorf("PollBoard(%s).Reported = %d, want %d", "jobs_optiply.xml", got, want)
	}
}

package workable_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/workable"
)

func TestFetchPage_Golden(t *testing.T) {
	sourcetest.RunGolden(t, "jobs_pearltalent.json", workable.New("pearltalent"))
}

func TestPollBoard_Reported(t *testing.T) {
	if got, want := sourcetest.PollReported(t, "jobs_pearltalent.json", workable.New("pearltalent")), 2; got != want {
		t.Errorf("PollBoard(%s).Reported = %d, want %d", "jobs_pearltalent.json", got, want)
	}
}

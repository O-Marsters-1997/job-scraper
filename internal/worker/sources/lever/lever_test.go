package lever_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources/lever"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

func TestFetchPage_Golden(t *testing.T) {
	sourcetest.RunGolden(t, "postings_acme.json", lever.New("acme"))
}

func TestFetchPage_GoldenWithPay(t *testing.T) {
	sourcetest.RunGolden(t, "postings_weride.json", lever.New("weride"))
}

func TestFetchPage_GoldenWorkplaceTypes(t *testing.T) {
	sourcetest.RunGolden(t, "postings_spotify.json", lever.New("spotify"))
}

func TestPollBoard_Reported(t *testing.T) {
	if got, want := sourcetest.PollReported(t, "postings_acme.json", lever.New("acme")), 2; got != want {
		t.Errorf("PollBoard(%s).Reported = %d, want %d", "postings_acme.json", got, want)
	}
}

package pinpoint_test

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources/pinpoint"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

func TestFetchPage_Golden(t *testing.T) {
	sourcetest.RunGolden(t, "postings_workwithus.json", pinpoint.New("workwithus"))
}

func TestFetchPage_GoldenEmptyBoard(t *testing.T) {
	sourcetest.RunGolden(t, "postings_empty.json", pinpoint.New("empty"))
}

func TestPollBoard_Reported(t *testing.T) {
	tests := []struct {
		fixture string
		want    int
	}{
		{"postings_workwithus.json", 3},
		{"postings_empty.json", 0},
	}
	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			if got := sourcetest.PollReported(t, tt.fixture, pinpoint.New("acme")); got != tt.want {
				t.Errorf("PollBoard(%s).Reported = %d, want %d", tt.fixture, got, tt.want)
			}
		})
	}
}

package teamtailor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/teamtailor"
)

func TestFetchPage_Golden(t *testing.T) {
	sourcetest.RunGolden(t, "jobs_nested.xml", teamtailor.New("nested"))
}

func TestFetchPage_GoldenHybrid(t *testing.T) {
	sourcetest.RunGolden(t, "jobs_ocado.xml", teamtailor.New("ocadoretail"))
}

func TestFetchPage_GoldenEmptyBoard(t *testing.T) {
	sourcetest.RunGolden(t, "jobs_empty.xml", teamtailor.New("empty"))
}

func TestPollBoard_Reported(t *testing.T) {
	tests := []struct {
		fixture string
		want    int
	}{
		{"jobs_nested.xml", 3},
		{"jobs_ocado.xml", 2},
		{"jobs_empty.xml", 0},
	}
	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			if got := sourcetest.PollReported(t, tt.fixture, teamtailor.New("acme")); got != tt.want {
				t.Errorf("PollBoard(%s).Reported = %d, want %d", tt.fixture, got, tt.want)
			}
		})
	}
}

func TestBoardName(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("snapshots", "jobs_ocado.xml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if got, want := teamtailor.BoardName(body), "Ocado Retail"; got != want {
		t.Errorf("BoardName() = %q, want %q", got, want)
	}
	if got := teamtailor.BoardName([]byte("not xml")); got != "" {
		t.Errorf("BoardName(not xml) = %q, want empty", got)
	}
}

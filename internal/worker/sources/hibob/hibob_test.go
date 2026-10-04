package hibob_test

import (
	"net/http"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/worker/sources/hibob"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

func TestFetchPage_Golden(t *testing.T) {
	sourcetest.RunGolden(t, "job_ad_mixed.json", hibob.New("acme"))
}

func TestFetchPage_GoldenEmptyBoard(t *testing.T) {
	sourcetest.RunGolden(t, "job_ad_empty.json", hibob.New("empty"))
}

type headerGate struct {
	t        *testing.T
	identity string
	calls    int
}

func (g *headerGate) RoundTrip(req *http.Request) (*http.Response, error) {
	g.calls++
	if got := req.Header.Get("companyidentifier"); got != g.identity {
		g.t.Errorf("request companyidentifier = %q, want %q", got, g.identity)
		return sourcetest.RespondStatus(http.StatusUnauthorized, "").RoundTrip(req)
	}
	return sourcetest.Respond(`{"jobAdDetails":[]}`).RoundTrip(req)
}

func TestFetchPage_SendsCompanyIdentifierHeader(t *testing.T) {
	src := hibob.New("acme")
	gate := &headerGate{t: t, identity: "acme"}
	src.Client().Transport = gate

	if _, _, err := src.FetchPage(t.Context(), ""); err != nil {
		t.Fatalf("FetchPage() error: %v", err)
	}
	if gate.calls != 1 {
		t.Errorf("requests made = %d, want 1", gate.calls)
	}
}

func TestPollBoard_Reported(t *testing.T) {
	tests := []struct {
		fixture string
		want    int
	}{
		{"job_ad_mixed.json", 4},
		{"job_ad_empty.json", 0},
	}
	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			if got := sourcetest.PollReported(t, tt.fixture, hibob.New("acme")); got != tt.want {
				t.Errorf("PollBoard(%s).Reported = %d, want %d", tt.fixture, got, tt.want)
			}
		})
	}
}

package workable_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/workable"
)

func noGap() *sources.Gate { return sources.NewGate(0, time.Hour) }

func TestFetchPage_Golden(t *testing.T) {
	sourcetest.RunGolden(t, "jobs_pearltalent.json", workable.New("pearltalent", noGap(), noGap()))
}

func TestPollBoard_Reported(t *testing.T) {
	if got, want := sourcetest.PollReported(t, "jobs_pearltalent.json", workable.New("pearltalent", noGap(), noGap())), 2; got != want {
		t.Errorf("PollBoard(%s).Reported = %d, want %d", "jobs_pearltalent.json", got, want)
	}
}

type pagedTransport struct {
	bodies []string
	tokens []string
}

func (p *pagedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	raw, _ := io.ReadAll(req.Body)
	var in struct{ Token string }
	_ = json.Unmarshal(raw, &in)
	p.tokens = append(p.tokens, in.Token)
	body := p.bodies[len(p.tokens)-1]
	return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(strings.NewReader(body))}, nil
}

func TestPollBoard_FollowsNextPage(t *testing.T) {
	tr := &pagedTransport{bodies: []string{
		`{"total":2,"results":[{"shortcode":"A","title":"a","workplace":"hybrid"}],"nextPage":"tok1"}`,
		`{"total":2,"results":[{"shortcode":"B","title":"b","workplace":"on_site"}],"nextPage":null}`,
	}}
	src := workable.New("acme", noGap(), noGap())
	src.Client().Transport = tr

	res, err := src.PollBoard(t.Context())
	if err != nil {
		t.Fatalf("PollBoard() error: %v", err)
	}

	var gotIDs, gotArrangements []string
	for _, j := range res.Jobs {
		gotIDs = append(gotIDs, j.ProviderPostingID)
		gotArrangements = append(gotArrangements, j.WorkArrangement)
	}
	if diff := cmp.Diff([]string{"A", "B"}, gotIDs); diff != "" {
		t.Errorf("PollBoard() job IDs mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"hybrid", "onsite"}, gotArrangements); diff != "" {
		t.Errorf("PollBoard() arrangements mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"", "tok1"}, tr.tokens); diff != "" {
		t.Errorf("PollBoard() request tokens mismatch (-want +got):\n%s", diff)
	}
	if res.Reported != 2 {
		t.Errorf("PollBoard().Reported = %d, want 2", res.Reported)
	}
}

type refusingTransport struct{ requests int }

func (r *refusingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	r.requests++
	header := http.Header{"Retry-After": {"86400"}}
	return &http.Response{StatusCode: http.StatusTooManyRequests, Status: "429 Too Many Requests", Header: header, Body: io.NopCloser(strings.NewReader(`{"error":"rate_limit"}`))}, nil
}

func TestPollBoard_RateLimitDefersLaterRequests(t *testing.T) {
	gate := noGap()
	tr := &refusingTransport{}
	first := workable.New("acme", gate, noGap())
	first.Client().Transport = tr
	_, err := first.PollBoard(t.Context())
	var status *sources.StatusError
	if !errors.As(err, &status) || status.Code != http.StatusTooManyRequests {
		t.Fatalf("PollBoard() = %v, want a 429 StatusError", err)
	}
	other := workable.New("other", gate, noGap())
	other.Client().Transport = tr
	_, err = other.PollBoard(t.Context())
	if !errors.Is(err, sources.ErrDeferred) {
		t.Errorf("PollBoard() after the 429 = %v, want ErrDeferred", err)
	}
	if tr.requests != 1 {
		t.Errorf("sent %d requests, want only the one that was refused", tr.requests)
	}
}

func TestPollBoard_ProbeRateLimitLeavesChecksRunning(t *testing.T) {
	gate, probeGate := noGap(), noGap()
	probe := workable.New("candidate", gate, probeGate)
	probe.Client().Transport = &refusingTransport{}
	_, err := probe.PollBoard(sources.WithProbe(t.Context()))
	var status *sources.StatusError
	if !errors.As(err, &status) || status.Code != http.StatusTooManyRequests {
		t.Fatalf("probe PollBoard() = %v, want a 429 StatusError", err)
	}

	again := workable.New("another", gate, probeGate)
	again.Client().Transport = &refusingTransport{}
	if _, err := again.PollBoard(sources.WithProbe(t.Context())); !errors.Is(err, sources.ErrDeferred) {
		t.Errorf("second probe after the 429 = %v, want ErrDeferred", err)
	}

	check := workable.New("verified", gate, probeGate)
	check.Client().Transport = &pagedTransport{bodies: []string{`{"total":1,"results":[{"shortcode":"A","title":"a"}],"nextPage":null}`}}
	res, err := check.PollBoard(t.Context())
	if err != nil {
		t.Fatalf("check PollBoard() after a probe 429 error: %v", err)
	}
	if len(res.Jobs) != 1 {
		t.Errorf("check PollBoard() returned %d jobs, want 1", len(res.Jobs))
	}
}

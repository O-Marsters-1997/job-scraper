package workable_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/worker/sources"
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
	src := workable.New("acme")
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
	workable.ResetLimiter()
	tr := &refusingTransport{}
	first := workable.New("acme")
	first.Client().Transport = tr
	_, err := first.PollBoard(t.Context())
	var status *sources.StatusError
	if !errors.As(err, &status) || status.Code != http.StatusTooManyRequests {
		t.Fatalf("PollBoard() = %v, want a 429 StatusError", err)
	}
	other := workable.New("other")
	other.Client().Transport = tr
	_, err = other.PollBoard(t.Context())
	if !errors.Is(err, sources.ErrDeferred) {
		t.Errorf("PollBoard() after the 429 = %v, want ErrDeferred", err)
	}
	if tr.requests != 1 {
		t.Errorf("sent %d requests, want only the one that was refused", tr.requests)
	}
}

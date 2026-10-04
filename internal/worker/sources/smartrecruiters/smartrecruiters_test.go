package smartrecruiters_test

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/worker/sources/smartrecruiters"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/sourcetest"
)

const (
	onsiteID = "744000153370939"
	hybridID = "744000153257599"
	londonID = "744000153239979"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("snapshots", name))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return string(body)
}

func listPath(token string) string { return "/v1/companies/" + token + "/postings" }

// boardResponder serves the Wise fixtures under token. The posting cache is keyed by token, so
// each test uses its own.
func boardResponder(t *testing.T, token string) *sourcetest.PathResponder {
	t.Helper()
	listPath := listPath(token)
	return sourcetest.RespondByPath(map[string]string{
		listPath + "?limit=100&offset=0": fixture(t, "postings_page1.json"),
		listPath + "?limit=100&offset=2": fixture(t, "postings_page2.json"),
		listPath + "/" + onsiteID:        fixture(t, "posting_"+onsiteID+".json"),
		listPath + "/" + hybridID:        fixture(t, "posting_"+hybridID+".json"),
		listPath + "/" + londonID:        fixture(t, "posting_"+londonID+".json"),
	})
}

func TestFetchPage_GoldenOverTwoListPagesAndDetails(t *testing.T) {
	src := smartrecruiters.New("Wise")
	src.Client().Transport = boardResponder(t, "Wise")

	start := time.Now()
	got, next, err := src.FetchPage(t.Context(), "")
	if err != nil {
		t.Fatalf("FetchPage() error: %v", err)
	}
	if next != "" {
		t.Errorf("FetchPage() next = %q, want empty", next)
	}
	sourcetest.MatchGolden(t, "postings_wise.json", got, start)
}

func TestPollBoard_ReportsTotalFound(t *testing.T) {
	src := smartrecruiters.New("Reported")
	responder := sourcetest.RespondByPath(map[string]string{
		"/v1/companies/Reported/postings": `{"totalFound":0,"content":[]}`,
	})
	src.Client().Transport = responder

	res, err := src.PollBoard(t.Context())
	if err != nil {
		t.Fatalf("PollBoard() error: %v", err)
	}
	if res.Reported != 0 || len(res.Jobs) != 0 {
		t.Errorf("PollBoard() = %d jobs, Reported %d, want none", len(res.Jobs), res.Reported)
	}

	src = smartrecruiters.New("WiseReported")
	src.Client().Transport = boardResponder(t, "WiseReported")
	res, err = src.PollBoard(t.Context())
	if err != nil {
		t.Fatalf("PollBoard() error: %v", err)
	}
	if got, want := res.Reported, 3; got != want {
		t.Errorf("PollBoard().Reported = %d, want %d", got, want)
	}
}

func TestPollBoard_SecondPollMakesNoDetailCalls(t *testing.T) {
	responder := boardResponder(t, "WiseCached")
	src := smartrecruiters.New("WiseCached")
	src.Client().Transport = responder

	first, err := src.PollBoard(t.Context())
	if err != nil {
		t.Fatalf("first PollBoard() error: %v", err)
	}
	second, err := src.PollBoard(t.Context())
	if err != nil {
		t.Fatalf("second PollBoard() error: %v", err)
	}

	for _, id := range []string{onsiteID, hybridID, londonID} {
		if got := responder.Calls(listPath("WiseCached") + "/" + id); got != 1 {
			t.Errorf("detail calls for %s = %d, want 1 across both polls", id, got)
		}
	}
	if len(second.Jobs) != len(first.Jobs) {
		t.Errorf("second poll jobs = %d, want %d", len(second.Jobs), len(first.Jobs))
	}
}

func TestPollBoard_SkipsWithdrawnPosting(t *testing.T) {
	listPath := listPath("WiseWithdrawn")
	src := smartrecruiters.New("WiseWithdrawn")
	src.Client().Transport = sourcetest.RespondByPath(map[string]string{
		listPath + "?limit=100&offset=0": fixture(t, "postings_page1.json"),
		listPath + "?limit=100&offset=2": fixture(t, "postings_page2.json"),
		listPath + "/" + onsiteID:        fixture(t, "posting_"+onsiteID+".json"),
		listPath + "/" + hybridID:        fixture(t, "posting_"+hybridID+".json"),
	})

	res, err := src.PollBoard(t.Context())
	if err != nil {
		t.Fatalf("PollBoard() error: %v", err)
	}
	if got, want := len(res.Jobs), 2; got != want {
		t.Errorf("PollBoard() jobs = %d, want %d", got, want)
	}
}

func TestPollBoard_ServerErrorFailsThePoll(t *testing.T) {
	src := smartrecruiters.New("Down")
	src.Client().Transport = sourcetest.RespondStatus(http.StatusInternalServerError, "")

	if _, err := src.PollBoard(t.Context()); err == nil {
		t.Error("PollBoard() = nil error, want one")
	}
}

func TestBoardName(t *testing.T) {
	if got, want := smartrecruiters.BoardName([]byte(fixture(t, "postings_page1.json"))), "Wise"; got != want {
		t.Errorf("BoardName() = %q, want %q", got, want)
	}
	if got := smartrecruiters.BoardName([]byte(`{"totalFound":0,"content":[]}`)); got != "" {
		t.Errorf("BoardName(empty board) = %q, want empty", got)
	}
}

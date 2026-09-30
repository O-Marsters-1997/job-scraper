package scraper_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/scraper"
)

type boardStoreStub struct {
	board dto.BoardPoll
	state string
}

func (s *boardStoreStub) ClaimBoard(context.Context, string, bool) (dto.BoardPoll, error) {
	return s.board, nil
}
func (s *boardStoreStub) CompleteBoard(context.Context, dto.BoardSnapshot) error {
	s.state = "completed"
	return nil
}
func (s *boardStoreStub) FailBoard(context.Context, dto.BoardPoll) error {
	s.state = "failed"
	return nil
}

type boardFetcherStub struct {
	jobs []dto.Job
	err  error
}

func (f boardFetcherStub) FetchBoard(context.Context, dto.BoardPoll) ([]dto.Job, error) {
	return f.jobs, f.err
}

type boardIngesterStub struct {
	err  error
	jobs []dto.Job
}

func (i *boardIngesterStub) BulkExport(_ context.Context, jobs []dto.Job) error {
	i.jobs = jobs
	return i.err
}

var (
	errPartial = errors.New("partial page")
	errPersist = errors.New("persistence failed")
)

func TestBoardPollCompletesOnlyAfterFetchAndIngest(t *testing.T) {
	for _, tc := range []struct {
		name      string
		fetchErr  error
		ingestErr error
		want      string
		wantErr   error
	}{
		{"fetch failed", errPartial, nil, "failed", errPartial},
		{"ingest failed", nil, errPersist, "failed", errPersist},
		{"complete", nil, nil, "completed", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &boardStoreStub{board: dto.BoardPoll{ID: "board", CompanyID: "company", Source: "greenhouse", Token: "acme"}}
			poller := scraper.NewBoardPoller(store, boardFetcherStub{jobs: []dto.Job{{Title: "Engineer", URL: "https://example.com/1"}}, err: tc.fetchErr}, &boardIngesterStub{err: tc.ingestErr})
			err := poller.PollBoard(context.Background(), "board", false)
			if tc.wantErr == nil && err != nil || tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Errorf("PollBoard() = %v, want %v", err, tc.wantErr)
			}
			if store.state != tc.want {
				t.Fatalf("board state = %q, want %q", store.state, tc.want)
			}
		})
	}
}

func TestBoardPollCarriesVerifiedCompanyIdentity(t *testing.T) {
	store := &boardStoreStub{board: dto.BoardPoll{ID: "board", CompanyID: "company", CompanySlug: "company-slug", Source: "greenhouse", Token: "regional-token"}}
	ingester := &boardIngesterStub{}
	poller := scraper.NewBoardPoller(store, boardFetcherStub{jobs: []dto.Job{{Title: "Engineer", URL: "https://example.com/1", CompanySlug: "regional-token"}}}, ingester)
	if err := poller.PollBoard(context.Background(), "board", false); err != nil {
		t.Fatal(err)
	}
	if len(ingester.jobs) != 1 || ingester.jobs[0].CompanyID != "company" || ingester.jobs[0].CompanySlug != "company-slug" || ingester.jobs[0].BoardID != "board" {
		t.Fatalf("ingested job identity=%v", ingester.jobs)
	}
}

package scraper

import (
	"context"
	"errors"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type boardStoreStub struct {
	board     dto.BoardPoll
	completed int
	failed    int
}

func (s *boardStoreStub) ListDueBoards(context.Context) ([]dto.BoardPoll, error) {
	return []dto.BoardPoll{s.board}, nil
}
func (s *boardStoreStub) ListActiveBoards(context.Context) ([]dto.BoardPoll, error) {
	return []dto.BoardPoll{s.board}, nil
}
func (s *boardStoreStub) ClaimBoard(context.Context, string, bool) (dto.BoardPoll, error) {
	return s.board, nil
}
func (s *boardStoreStub) CompleteBoard(context.Context, dto.BoardSnapshot) error {
	s.completed++
	return nil
}
func (s *boardStoreStub) FailBoard(context.Context, dto.BoardPoll) error { s.failed++; return nil }

type boardFetcherStub struct {
	jobs []dto.Job
	err  error
}

func (f boardFetcherStub) FetchBoard(context.Context, dto.BoardPoll) ([]dto.Job, error) {
	return f.jobs, f.err
}

type boardIngesterStub struct{ err error }

func (i boardIngesterStub) BulkExport(context.Context, []dto.Job) error { return i.err }

type captureBoardIngester struct{ jobs []dto.Job }

func (i *captureBoardIngester) BulkExport(_ context.Context, jobs []dto.Job) error {
	i.jobs = jobs
	return nil
}

func TestBoardPollCompletesOnlyAfterFetchAndIngest(t *testing.T) {
	for _, tc := range []struct {
		name             string
		fetchErr         error
		ingestErr        error
		complete, failed int
	}{
		{"fetch failed", errors.New("partial page"), nil, 0, 1},
		{"ingest failed", nil, errors.New("persistence failed"), 0, 1},
		{"complete", nil, nil, 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &boardStoreStub{board: dto.BoardPoll{ID: "board", CompanyID: "company", Source: "greenhouse", Token: "acme"}}
			poller := NewBoardPoller(store, boardFetcherStub{jobs: []dto.Job{{Title: "Engineer", URL: "https://example.com/1"}}, err: tc.fetchErr}, boardIngesterStub{err: tc.ingestErr})
			_ = poller.PollDue(context.Background())
			if store.completed != tc.complete || store.failed != tc.failed {
				t.Fatalf("completed=%d failed=%d, want %d/%d", store.completed, store.failed, tc.complete, tc.failed)
			}
		})
	}
}

func TestBoardPollCarriesVerifiedCompanyIdentity(t *testing.T) {
	store := &boardStoreStub{board: dto.BoardPoll{ID: "board", CompanyID: "company", CompanySlug: "company-slug", Source: "greenhouse", Token: "regional-token"}}
	ingester := &captureBoardIngester{}
	poller := NewBoardPoller(store, boardFetcherStub{jobs: []dto.Job{{Title: "Engineer", URL: "https://example.com/1", CompanySlug: "regional-token"}}}, ingester)
	if err := poller.PollDue(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(ingester.jobs) != 1 || ingester.jobs[0].CompanyID != "company" || ingester.jobs[0].CompanySlug != "company-slug" || ingester.jobs[0].BoardID != "board" {
		t.Fatalf("ingested job identity=%v", ingester.jobs)
	}
}

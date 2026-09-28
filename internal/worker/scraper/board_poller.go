package scraper

import (
	"context"
	"errors"
	"fmt"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/builder"
)

type BoardPollStore interface {
	ListDueBoards(context.Context) ([]dto.BoardPoll, error)
	ListActiveBoards(context.Context) ([]dto.BoardPoll, error)
	ClaimBoard(context.Context, string, bool) (dto.BoardPoll, error)
	CompleteBoard(context.Context, dto.BoardSnapshot) error
	FailBoard(context.Context, dto.BoardPoll) error
}

type BoardFetcher interface {
	FetchBoard(context.Context, dto.BoardPoll) ([]dto.Job, error)
}

type BoardIngester interface {
	BulkExport(context.Context, []dto.Job) error
}

type BoardPoller struct {
	store    BoardPollStore
	fetcher  BoardFetcher
	ingester BoardIngester
}

func NewBoardPoller(store BoardPollStore, fetcher BoardFetcher, ingester BoardIngester) *BoardPoller {
	return &BoardPoller{store: store, fetcher: fetcher, ingester: ingester}
}

func (p *BoardPoller) PollDue(ctx context.Context) error {
	boards, err := p.store.ListDueBoards(ctx)
	return p.poll(ctx, boards, false, err)
}

func (p *BoardPoller) PollAll(ctx context.Context) error {
	boards, err := p.store.ListActiveBoards(ctx)
	return p.poll(ctx, boards, true, err)
}

func (p *BoardPoller) poll(ctx context.Context, boards []dto.BoardPoll, manual bool, err error) error {
	if err != nil {
		return err
	}
	var errs []error
	for _, board := range boards {
		if err := p.PollBoard(ctx, board.ID, manual); err != nil {
			errs = append(errs, fmt.Errorf("board %s: %w", board.ID, err))
		}
	}
	return errors.Join(errs...)
}

func (p *BoardPoller) PollBoard(ctx context.Context, id string, manual bool) error {
	claim, err := p.store.ClaimBoard(ctx, id, manual)
	if errors.Is(err, jobsearch.ErrBoardClaimUnavailable) && !manual {
		return nil
	}
	if err != nil {
		return err
	}
	jobs, err := p.fetcher.FetchBoard(ctx, claim)
	if err == nil {
		for idx := range jobs {
			jobs[idx].BoardID = claim.ID
			jobs[idx].CompanyID = claim.CompanyID
			jobs[idx].CompanySlug = claim.CompanySlug
		}
		err = p.ingester.BulkExport(ctx, jobs)
	}
	if err != nil {
		return errors.Join(err, p.store.FailBoard(ctx, claim))
	}
	return p.store.CompleteBoard(ctx, dto.BoardSnapshot{Poll: claim, Jobs: jobs, Complete: true})
}

type SourceBoardFetcher struct{}

func (SourceBoardFetcher) FetchBoard(ctx context.Context, board dto.BoardPoll) ([]dto.Job, error) {
	target := dto.SourceTarget{Source: board.Source, Value: board.Token, Enabled: true}
	src, ok := builder.BuildSource(target)
	if !ok {
		return nil, fmt.Errorf("unsupported board source %q", board.Source)
	}
	jobs, _, err := src.FetchPage(ctx, "")
	return jobs, err
}

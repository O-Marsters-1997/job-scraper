package scraper

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/builder"
)

type BoardPollStore interface {
	ClaimBoard(context.Context, string, bool) (dto.BoardPoll, error)
	CompleteBoard(context.Context, dto.BoardSnapshot) error
	FailBoard(context.Context, dto.BoardPoll) error
}

type BoardFetcher interface {
	FetchBoard(context.Context, dto.BoardPoll) (jobs []dto.Job, nextPollIn time.Duration, err error)
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

func (p *BoardPoller) PollBoard(ctx context.Context, id string, manual bool) error {
	claim, err := p.store.ClaimBoard(ctx, id, manual)
	if errors.Is(err, jobsearch.ErrBoardClaimUnavailable) && !manual {
		return nil
	}
	if err != nil {
		return err
	}
	jobs, nextPollIn, err := p.fetcher.FetchBoard(ctx, claim)
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
	return p.store.CompleteBoard(ctx, dto.BoardSnapshot{Poll: claim, Jobs: jobs, Complete: true, NextPollIn: nextPollIn})
}

type ProfileSaver interface {
	SaveCompanyProfile(ctx context.Context, companyID, source string, profile dto.CompanyProfile) error
}

type SourceBoardFetcher struct {
	Profiles ProfileSaver
}

func (f SourceBoardFetcher) FetchBoard(ctx context.Context, board dto.BoardPoll) ([]dto.Job, time.Duration, error) {
	target := dto.SourceTarget{Source: board.Source, Value: board.Token, Enabled: true}
	src, ok := builder.BuildSource(target)
	if !ok {
		return nil, 0, fmt.Errorf("unsupported board source %q", board.Source)
	}
	bp, ok := src.(sources.BoardPoller)
	if !ok {
		jobs, _, err := src.FetchPage(ctx, "")
		return jobs, 0, err
	}
	res, err := bp.PollBoard(ctx)
	if err != nil {
		return nil, 0, err
	}
	if res.Profile != nil {
		if err := f.Profiles.SaveCompanyProfile(ctx, board.CompanyID, board.Source, *res.Profile); err != nil {
			return nil, 0, err
		}
	}
	return res.Jobs, res.NextPollIn, nil
}

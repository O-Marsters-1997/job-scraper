package scraper

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/telemetry"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/builder"
)

type BoardPollStore interface {
	ClaimBoard(context.Context, string, bool) (dto.BoardPoll, error)
	CompleteBoard(context.Context, dto.BoardSnapshot) error
	FailBoard(context.Context, dto.BoardPoll) error
}

type BoardFetcher interface {
	FetchBoard(context.Context, dto.BoardPoll) (sources.BoardResult, error)
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
	res, err := p.fetcher.FetchBoard(ctx, claim)
	return p.complete(ctx, claim, res, err)
}

// PollPrefetched completes a Board's poll from jobs the caller already fetched.
func (p *BoardPoller) PollPrefetched(ctx context.Context, id string, jobs []dto.Job, nextPollIn time.Duration) error {
	claim, err := p.store.ClaimBoard(ctx, id, false)
	if errors.Is(err, jobsearch.ErrBoardClaimUnavailable) {
		return nil
	}
	if err != nil {
		return err
	}
	return p.complete(ctx, claim, sources.BoardResult{Jobs: jobs, NextPollIn: nextPollIn}, nil)
}

// underparsedRatio is the share of the reported count a poll must parse to count as full.
const underparsedRatio = 0.98

func (p *BoardPoller) complete(ctx context.Context, claim dto.BoardPoll, res sources.BoardResult, err error) error {
	jobs := res.Jobs
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
	if res.Reported > 0 && float64(len(jobs)) < underparsedRatio*float64(res.Reported) {
		slog.WarnContext(ctx, "board parsed fewer jobs than reported",
			slog.String(logger.KeyEvent, telemetry.EventBoardUnderparsed),
			slog.String(logger.KeySource, claim.Source),
			slog.String("token", claim.Token),
			slog.Int("reported", res.Reported),
			slog.Int("parsed", len(jobs)))
	}
	return p.store.CompleteBoard(ctx, dto.BoardSnapshot{Poll: claim, Jobs: jobs, Complete: true, NextPollIn: res.NextPollIn, Reported: res.Reported})
}

type ProfileSaver interface {
	SaveCompanyProfile(ctx context.Context, companyID, source string, profile dto.CompanyProfile) error
}

type SourceBoardFetcher struct {
	Profiles ProfileSaver
}

func (f SourceBoardFetcher) FetchBoard(ctx context.Context, board dto.BoardPoll) (sources.BoardResult, error) {
	target := dto.SourceTarget{Source: board.Source, Value: board.Token, Enabled: true}
	src, ok := builder.BuildSource(target)
	if !ok {
		return sources.BoardResult{}, fmt.Errorf("unsupported board source %q", board.Source)
	}
	bp, ok := src.(sources.BoardPoller)
	if !ok {
		jobs, _, err := src.FetchPage(ctx, "")
		return sources.BoardResult{Jobs: jobs}, err
	}
	res, err := bp.PollBoard(ctx)
	if err != nil {
		return sources.BoardResult{}, err
	}
	if res.Profile != nil {
		if err := f.Profiles.SaveCompanyProfile(ctx, board.CompanyID, board.Source, *res.Profile); err != nil {
			return sources.BoardResult{}, err
		}
	}
	return res, nil
}

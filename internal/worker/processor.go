package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/worker/scraper"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

func (p *Processor) FailRun(ctx context.Context, task queue.Task) error {
	if task.Kind == queue.DetailTask || task.TargetID == "" || task.RunID == "" {
		return nil
	}
	_, err := p.js.Targets().TransitionSourceTargetRun(ctx, task.TargetID, task.RunID, "failed", "Work failed after retries. Try running it again.")
	if errors.Is(err, data.ErrNotFound) {
		return nil
	}
	return err
}

type Publisher interface {
	Publish(ctx context.Context, task queue.Task) error
}

type Deps struct {
	JS           *jobsearch.Module
	Broker       Publisher
	Orchestrator *scraper.Orchestrator
	Boards       *scraper.BoardPoller
	Detailers    map[string]sources.DetailFetcher
	Exporter     *scraper.APIExporter
}

type Processor struct {
	js           *jobsearch.Module
	broker       Publisher
	orchestrator *scraper.Orchestrator
	boards       *scraper.BoardPoller
	detailers    map[string]sources.DetailFetcher
	exporter     *scraper.APIExporter
}

func NewProcessor(d Deps) *Processor {
	return &Processor{js: d.JS, broker: d.Broker, orchestrator: d.Orchestrator, boards: d.Boards, detailers: d.Detailers, exporter: d.Exporter}
}

func (p *Processor) Process(ctx context.Context, task queue.Task) error {
	switch task.Kind {
	case queue.DetailTask:
		if task.Source == "remoteok" || task.Source == "remotive" {
			return p.exporter.Export(ctx, task.Card)
		}
		fetcher := p.detailers[task.Source]
		if fetcher == nil {
			return fmt.Errorf("source %s cannot fetch details", task.Source)
		}
		job, err := fetcher.GetDetails(ctx, task.URL)
		if err != nil {
			return err
		}
		return p.exporter.Export(ctx, job)
	case queue.ListingPageTask:
		return p.processPage(ctx, task)
	case queue.BoardCheckTask:
		return p.processBoard(ctx, task)
	case queue.BoardVerifyTask:
		return p.verifyBoard(ctx, task)
	default:
		return fmt.Errorf("unsupported task kind %s", task.Kind)
	}
}

func (p *Processor) verifyBoard(ctx context.Context, task queue.Task) error {
	if err := scraper.VerifyBoard(ctx, task.Source, task.BoardToken); err != nil {
		slog.WarnContext(ctx, "board verification failed", slog.String(logger.KeyCompanyID, task.CompanyID), slog.String(logger.KeySource, task.Source), slog.String("token", task.BoardToken), slog.Any(logger.KeyErr, err))
		return nil
	}
	_, err := p.js.Boards().VerifyCompanyBoard(ctx, task.CompanyID, task.Source, task.BoardToken, "user_confirmed")
	return err
}

func (p *Processor) currentTarget(ctx context.Context, task queue.Task) (dto.SourceTarget, bool, error) {
	target, err := p.js.Targets().GetSourceTarget(ctx, task.TargetID)
	if errors.Is(err, data.ErrNotFound) {
		return dto.SourceTarget{}, false, nil
	}
	if err != nil {
		return dto.SourceTarget{}, false, err
	}
	if !target.Enabled || target.RunID != task.RunID || target.Source != task.Source || target.RunStatus == "succeeded" || target.RunStatus == "failed" {
		return target, false, nil
	}
	if task.Cursor == "" && !task.Redelivered && !task.Recovery && target.RunStatus == "running" && time.Since(target.UpdatedAt) < 30*time.Minute {
		return target, false, nil
	}
	return target, true, nil
}

func (p *Processor) processPage(ctx context.Context, task queue.Task) error {
	target, active, err := p.currentTarget(ctx, task)
	if err != nil || !active {
		return err
	}
	if target.RunStatus == "queued" || task.Cursor == "" {
		if _, err := p.js.Targets().TransitionSourceTargetRun(ctx, target.ID, task.RunID, "running", ""); err != nil {
			return err
		}
	}
	next, err := p.orchestrator.ScrapePage(ctx, target, task.Cursor)
	if err != nil {
		return err
	}
	if next != "" {
		task.ID, task.Cursor = uuid.NewString(), next
		if err := p.broker.Publish(ctx, task); err != nil {
			return err
		}
		_, err = p.js.Targets().TransitionSourceTargetRun(ctx, target.ID, task.RunID, "running", "")
		return err
	}
	_, err = p.js.Targets().TransitionSourceTargetRun(ctx, target.ID, task.RunID, "succeeded", "")
	return err
}

func (p *Processor) processBoard(ctx context.Context, task queue.Task) error {
	if task.TargetID != "" {
		target, active, err := p.currentTarget(ctx, task)
		if err != nil || !active {
			return err
		}
		if _, err := p.js.Targets().TransitionSourceTargetRun(ctx, target.ID, task.RunID, "running", ""); err != nil {
			return err
		}
	}
	if err := p.boards.PollBoard(ctx, task.BoardID, task.Manual); err != nil {
		return err
	}
	if task.TargetID != "" {
		_, err := p.js.Targets().TransitionSourceTargetRun(ctx, task.TargetID, task.RunID, "succeeded", "")
		return err
	}
	return nil
}

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
	"github.com/ollymarsters/job-scraper/internal/filter"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/slug"
	"github.com/ollymarsters/job-scraper/internal/worker/proxy"
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

type IncludeFilterConfigs interface {
	IncludeFilterConfigs(ctx context.Context) ([]dto.SearchConfig, error)
}

type DiscoverFunc func(ctx context.Context, source, token string) (name string, jobs []dto.Job, err error)

type Deps struct {
	JS           *jobsearch.Module
	Broker       Publisher
	Orchestrator *scraper.Orchestrator
	Boards       *scraper.BoardPoller
	Detailers    map[string]sources.DetailFetcher
	Exporter     *scraper.APIExporter
	MaxPages     int
	Scoring      IncludeFilterConfigs
	Discover     DiscoverFunc
}

type Processor struct {
	js           *jobsearch.Module
	broker       Publisher
	orchestrator *scraper.Orchestrator
	boards       *scraper.BoardPoller
	detailers    map[string]sources.DetailFetcher
	exporter     *scraper.APIExporter
	maxPages     int
	scoring      IncludeFilterConfigs
	discover     DiscoverFunc
}

func NewProcessor(d Deps) *Processor {
	return &Processor{js: d.JS, broker: d.Broker, orchestrator: d.Orchestrator, boards: d.Boards, detailers: d.Detailers, exporter: d.Exporter, maxPages: d.MaxPages, scoring: d.Scoring, discover: d.Discover}
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
		ctx, fetches := proxy.WithCollector(ctx)
		job, err := fetcher.GetDetails(ctx, task.URL)
		if errors.Is(err, sources.ErrGone) {
			slog.WarnContext(ctx, "job is gone", slog.String(logger.KeySource, task.Source), slog.String(logger.KeyURL, task.URL), slog.Any(logger.KeyErr, err))
			return nil
		}
		if err != nil {
			return err
		}
		if err := p.exporter.Export(ctx, job); err != nil {
			return err
		}
		p.forgetFetches(ctx, fetches)
		return nil
	case queue.ListingPageTask:
		ctx, fetches := proxy.WithCollector(ctx)
		if err := p.processPage(ctx, task); err != nil {
			return err
		}
		p.forgetFetches(ctx, fetches)
		return nil
	case queue.BoardCheckTask:
		return p.processBoard(ctx, task)
	case queue.BoardVerifyTask:
		return p.verifyBoard(ctx, task)
	case queue.BoardDiscoverTask:
		return p.discoverBoard(ctx, task)
	default:
		return fmt.Errorf("unsupported task kind %s", task.Kind)
	}
}

func (p *Processor) forgetFetches(ctx context.Context, c *proxy.Collector) {
	if err := p.js.ForgetFetches(ctx, c.URLs()); err != nil {
		slog.ErrorContext(ctx, "forget fetch cache failed", slog.Any(logger.KeyErr, err))
	}
}

func (p *Processor) verifyBoard(ctx context.Context, task queue.Task) error {
	verified, err := scraper.VerifyBoard(ctx, task.Source, task.BoardToken)
	if err != nil {
		slog.WarnContext(ctx, "board verification failed", slog.String(logger.KeyCompanyID, task.CompanyID), slog.String(logger.KeySource, task.Source), slog.String("token", task.BoardToken), slog.Any(logger.KeyErr, err))
		return nil
	}
	if _, err := p.js.Boards().VerifyCompanyBoard(ctx, task.CompanyID, task.Source, task.BoardToken, verified.Method); err != nil {
		return err
	}
	return p.nameCompany(ctx, task.CompanyID, verified.CompanyName)
}

func (p *Processor) nameCompany(ctx context.Context, companyID, name string) error {
	if name == "" {
		return nil
	}
	company, err := p.js.Boards().GetCompany(ctx, companyID)
	if err != nil {
		return err
	}
	if company.Name != slug.Humanize(company.Slug) {
		return nil
	}
	return p.js.Boards().RenameCompany(ctx, companyID, name)
}

func (p *Processor) discoverBoard(ctx context.Context, task queue.Task) error {
	name, jobs, err := p.discover(ctx, task.Source, task.BoardToken)
	if err != nil {
		slog.WarnContext(ctx, "board discovery failed", slog.String(logger.KeySource, task.Source), slog.String("token", task.BoardToken), slog.Any(logger.KeyErr, err))
		return nil
	}
	company, err := p.js.Boards().UpsertCompany(ctx, dto.CompanyUpsert{Slug: slug.Make(name), Name: name})
	if err != nil {
		return err
	}
	if _, err := p.js.Boards().UpsertCandidateBoard(ctx, company.ID, task.Source, task.BoardToken); err != nil {
		return err
	}
	if _, err := p.js.Boards().VerifyCompanyBoard(ctx, company.ID, task.Source, task.BoardToken, "discovered"); err != nil && !errors.Is(err, data.ErrNotFound) {
		return err
	}
	configs, err := p.scoring.IncludeFilterConfigs(ctx)
	if err != nil {
		return err
	}
	for _, cfg := range configs {
		if !anyPasses(jobs, cfg) {
			continue
		}
		if _, err := p.js.TrackDiscoveredCompany(ctx, cfg.UserID, company.ID); err != nil {
			return err
		}
	}
	return nil
}

func anyPasses(jobs []dto.Job, cfg dto.SearchConfig) bool {
	for _, job := range jobs {
		if _, rejected := filter.Reject(job, cfg); !rejected {
			return true
		}
	}
	return false
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
	task.Page++
	if next != "" && (p.maxPages == 0 || task.Page < p.maxPages) {
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

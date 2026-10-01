// Package sourcetargets is the jobsearch context's source-target feature:
// what to scrape, on what cadence, and starting a run (ADR 0011).
package sourcetargets

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
)

type QueuePublisher interface {
	Publish(ctx context.Context, task queue.Task) error
	EnqueueJobs(ctx context.Context, jobs []dto.QueuedJob) error
}

type Store interface {
	CandidateStore
	CreateSourceTarget(ctx context.Context, userID, source, value string, enabled bool, filters map[string]string) (dto.SourceTarget, error)
	CreateSourceTargetWithRun(ctx context.Context, userID, source, value string, enabled bool, filters map[string]string) (dto.SourceTarget, error)
	UpdateSourceTarget(ctx context.Context, id, userID string, enabled *bool, checkIntervalMinutes *int) (dto.SourceTarget, error)
	DeleteSourceTarget(ctx context.Context, id, userID string) error
	ListSourceTargetsByUser(ctx context.Context, userID string) ([]dto.SourceTarget, error)
	StartSourceTargetRun(ctx context.Context, id string) (dto.SourceTarget, error)
	GetSourceTarget(ctx context.Context, id string) (dto.SourceTarget, error)
	TransitionSourceTargetRun(ctx context.Context, id, runID, status, runError string) (dto.SourceTarget, error)
	ListRecoverableSourceTargets(ctx context.Context) ([]dto.SourceTarget, error)
	ClaimRecoverableSourceTarget(ctx context.Context, id, runID string) (dto.SourceTarget, error)
}

// SearchConfigReader reads a user's Search Config from the scoring context.
type SearchConfigReader interface {
	SearchConfig(ctx context.Context, userID string) (dto.SearchConfig, error)
}

type Service struct {
	targets Store
	configs SearchConfigReader
	queue   QueuePublisher
	boards  CardBoards
}

func New(targets Store, configs SearchConfigReader, q QueuePublisher, boards CardBoards) *Service {
	return &Service{targets: targets, configs: configs, queue: q, boards: boards}
}

// Create validates a new source target against the source registry, then
// creates it. An enabled target starts a run immediately and publishes it to
// the queue.
func (s *Service) Create(ctx context.Context, userID string, in dto.CreateSourceTargetInput) (dto.SourceTarget, error) {
	if in.Source == "" || in.Value == "" {
		return dto.SourceTarget{}, apperr.Invalid("source and value are required")
	}

	if role, _ := sourcespec.SourceRole(in.Source); role == sourcespec.RoleATS {
		return dto.SourceTarget{}, apperr.Invalid("ATS boards are tracked as companies — add it under Company boards")
	}
	if role, _ := sourcespec.SourceRole(in.Source); role == sourcespec.RoleDiscovery {
		if t := detect.Detect(in.Value); t != detect.UnknownHTML && t != detect.Aggregator {
			return dto.SourceTarget{}, apperr.Invalid("that looks like an ATS board — add it under Company boards")
		}
	}

	if in.Source == "indeed" {
		if search, ok := detect.ParseSearchURL(in.Value); ok && search.Source == in.Source {
			in.Value = search.Value
		}
	}

	if err := validateSourceValue(in.Source, in.Value, in.Filters); err != nil {
		return dto.SourceTarget{}, err
	}

	filters := in.Filters
	if filters == nil {
		filters = map[string]string{}
	}

	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}

	startNow := enabled
	create := s.targets.CreateSourceTarget
	if startNow {
		create = s.targets.CreateSourceTargetWithRun
	}
	target, err := create(ctx, userID, in.Source, in.Value, enabled, filters)
	if err != nil {
		return dto.SourceTarget{}, err
	}

	if startNow {
		published, err := s.publishRun(ctx, target)
		if err != nil && published.ID == "" {
			return dto.SourceTarget{}, apperr.Unavailable("search saved but could not start; retry from Searches")
		}
		target = published
	}

	return target, nil
}

func validateSourceValue(source, value string, filters map[string]string) error {
	if fields, isFilter := sourcespec.LookupFilterFields(source); isFilter {
		for k := range filters {
			if !isKnownFilterField(k, fields) {
				return apperr.Invalid("unknown filter key: " + k)
			}
		}
		for _, f := range fields {
			if f.Required && filters[f.Name] == "" {
				return apperr.Invalid("missing required filter: " + f.Name)
			}
		}
		return nil
	}

	urlPrefix, isURL, ok := sourcespec.LookupSource(source)
	if !ok {
		return apperr.Invalid("unsupported source")
	}
	if isURL && !strings.HasPrefix(value, urlPrefix) {
		return apperr.Invalid("value must be a URL starting with " + urlPrefix)
	}
	if !isURL && (strings.Contains(value, "://") || strings.Contains(value, "/")) {
		return apperr.Invalid("enter just the board token, e.g. acmecorp, not the full URL")
	}
	return nil
}

func isKnownFilterField(key string, fields []sourcespec.FilterField) bool {
	for _, f := range fields {
		if f.Name == key {
			return true
		}
	}
	return false
}

// Update changes a source target's enabled state and/or check interval. If
// it enables a discovery target, saved candidates are reconsidered against
// the user's current search config.
func (s *Service) Update(ctx context.Context, userID string, in dto.UpdateSourceTargetInput) (dto.SourceTarget, error) {
	if in.Enabled == nil && in.CheckIntervalMinutes == nil {
		return dto.SourceTarget{}, apperr.Invalid("nothing to update")
	}
	if in.CheckIntervalMinutes != nil && *in.CheckIntervalMinutes < 60 {
		return dto.SourceTarget{}, apperr.Invalid("check_interval_minutes must be at least 60")
	}

	target, err := s.targets.UpdateSourceTarget(ctx, in.ID, userID, in.Enabled, in.CheckIntervalMinutes)
	if err != nil {
		return dto.SourceTarget{}, err
	}

	if in.Enabled == nil || !*in.Enabled {
		return target, nil
	}
	if role, _ := sourcespec.SourceRole(target.Source); role != sourcespec.RoleDiscovery {
		return target, nil
	}

	cfg, err := s.configs.SearchConfig(ctx, userID)
	if errors.Is(err, data.ErrNotFound) {
		cfg = dto.SearchConfig{UserID: userID}
	} else if err != nil {
		return dto.SourceTarget{}, apperr.Unavailable("search saved but candidate reconsideration failed")
	}
	if err := s.Reconsider(ctx, cfg); err != nil {
		return dto.SourceTarget{}, apperr.Unavailable("search saved but candidate reconsideration failed")
	}

	return target, nil
}

func (s *Service) List(ctx context.Context, userID string) ([]dto.SourceTarget, error) {
	targets, err := s.targets.ListSourceTargetsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range targets {
		targets[i].URL = detect.BuildSearchURL(targets[i].Source, targets[i].Value, targets[i].Filters)
	}
	return targets, nil
}

func (s *Service) Delete(ctx context.Context, userID, id string) error {
	return s.targets.DeleteSourceTarget(ctx, id, userID)
}

func (s *Service) Scrape(ctx context.Context, userID, id string) (dto.SourceTarget, error) {
	targets, err := s.targets.ListSourceTargetsByUser(ctx, userID)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	for _, target := range targets {
		if target.ID != id {
			continue
		}
		if target.RunStatus == "queued" || target.RunStatus == "running" {
			return dto.SourceTarget{}, apperr.Conflict("search already in progress")
		}
		target.Enabled = true
		queued, err := s.enqueueRun(ctx, target)
		if err != nil {
			return dto.SourceTarget{}, apperr.Unavailable("could not start search")
		}
		return queued, nil
	}
	return dto.SourceTarget{}, apperr.NotFound("not found")
}

func (s *Service) enqueueRun(ctx context.Context, target dto.SourceTarget) (dto.SourceTarget, error) {
	queued, err := s.targets.StartSourceTargetRun(ctx, target.ID)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	return s.publishRun(ctx, queued)
}

func (s *Service) GetSourceTarget(ctx context.Context, id string) (dto.SourceTarget, error) {
	return s.targets.GetSourceTarget(ctx, id)
}

func (s *Service) TransitionSourceTargetRun(ctx context.Context, id, runID, status, runError string) (dto.SourceTarget, error) {
	return s.targets.TransitionSourceTargetRun(ctx, id, runID, status, runError)
}

func (s *Service) ListRecoverableSourceTargets(ctx context.Context) ([]dto.SourceTarget, error) {
	return s.targets.ListRecoverableSourceTargets(ctx)
}

func (s *Service) ClaimRecoverableSourceTarget(ctx context.Context, id, runID string) (dto.SourceTarget, error) {
	return s.targets.ClaimRecoverableSourceTarget(ctx, id, runID)
}

func (s *Service) publishRun(ctx context.Context, queued dto.SourceTarget) (dto.SourceTarget, error) {
	queued.Enabled = true
	task := queue.Task{Version: 1, ID: uuid.NewString(), Source: queued.Source, Kind: queue.ListingPageTask, TargetID: queued.ID, RunID: queued.RunID}
	if err := s.queue.Publish(ctx, task); err != nil {
		return queued, err
	}
	return queued, nil
}

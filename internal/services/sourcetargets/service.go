// Package sourcetargets holds the domain rules for creating, updating and
// (re)running a source target. Persistence goes through
// providers.SourceTargetProvider; queue publishing goes through the
// QueuePublisher port declared here.
package sourcetargets

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/candidates"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/sources/registry"
)

type QueuePublisher interface {
	Publish(ctx context.Context, task queue.Task) error
}

type Service struct {
	targets    providers.SourceTargetProvider
	configs    providers.SearchConfigProvider
	candidates *candidates.Service
	queue      QueuePublisher
}

func New(targets providers.SourceTargetProvider, configs providers.SearchConfigProvider, candidates *candidates.Service, q QueuePublisher) *Service {
	return &Service{targets: targets, configs: configs, candidates: candidates, queue: q}
}

// Create validates a new source target against the source registry, then
// creates it. Discovery sources (and ATS sources with ScrapeNow set) start a
// run immediately and publish it to the queue.
func (s *Service) Create(ctx context.Context, userID string, in dto.CreateSourceTargetInput) (dto.SourceTarget, error) {
	if in.Source == "" || in.Value == "" {
		return dto.SourceTarget{}, apperr.Invalid("source and value are required")
	}

	if role, _ := registry.SourceRole(in.Source); role == registry.RoleDiscovery {
		// An ATS board URL pasted into a discovery source belongs under Tracked
		// companies, not here. (Discovery values are keywords or aggregator URLs.)
		if t := detect.Detect(in.Value); t != detect.UnknownHTML && t != detect.Aggregator {
			return dto.SourceTarget{}, apperr.Invalid("that looks like an ATS board — add it under Tracked companies")
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

	role, _ := registry.SourceRole(in.Source)
	startNow := enabled && (role == registry.RoleDiscovery || in.ScrapeNow)
	if startNow && role == registry.RoleATS {
		if _, err := s.targets.GetVerifiedBoardID(ctx, in.Source, in.Value); err != nil {
			return dto.SourceTarget{}, apperr.Unprocessable("verified Board required to start search")
		}
	}

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
	if fields, isFilter := registry.LookupFilterFields(source); isFilter {
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

	urlPrefix, isURL, ok := registry.LookupSource(source)
	if !ok {
		return apperr.Invalid("unsupported source")
	}
	if isURL && !strings.HasPrefix(value, urlPrefix) {
		return apperr.Invalid("value must be a URL starting with " + urlPrefix)
	}
	// ATS board slot: value must be a bare token, not a URL.
	if !isURL && (strings.Contains(value, "://") || strings.Contains(value, "/")) {
		return apperr.Invalid("enter just the board token, e.g. acmecorp, not the full URL")
	}
	return nil
}

func isKnownFilterField(key string, fields []registry.FilterField) bool {
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
	if role, _ := registry.SourceRole(target.Source); role != registry.RoleDiscovery {
		return target, nil
	}

	cfg, err := s.configs.GetSearchConfig(ctx, userID)
	if errors.Is(err, providers.ErrNotFound) {
		cfg = dto.SearchConfig{UserID: userID}
	} else if err != nil {
		return dto.SourceTarget{}, apperr.Unavailable("search saved but candidate reconsideration failed")
	}
	if err := s.candidates.Reconsider(ctx, cfg); err != nil {
		return dto.SourceTarget{}, apperr.Unavailable("search saved but candidate reconsideration failed")
	}

	return target, nil
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
	if role, _ := registry.SourceRole(target.Source); role == registry.RoleATS {
		if _, err := s.targets.GetVerifiedBoardID(ctx, target.Source, target.Value); err != nil {
			return dto.SourceTarget{}, err
		}
	}
	queued, err := s.targets.StartSourceTargetRun(ctx, target.ID)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	return s.publishRun(ctx, queued)
}

func (s *Service) publishRun(ctx context.Context, queued dto.SourceTarget) (dto.SourceTarget, error) {
	queued.Enabled = true
	task := queue.Task{Version: 1, ID: uuid.NewString(), Source: queued.Source, TargetID: queued.ID, RunID: queued.RunID}
	if role, _ := registry.SourceRole(queued.Source); role == registry.RoleATS {
		boardID, err := s.targets.GetVerifiedBoardID(ctx, queued.Source, queued.Value)
		if err != nil {
			return queued, err
		}
		task.Kind, task.BoardID, task.Manual = queue.BoardCheckTask, boardID, true
	} else {
		task.Kind = queue.ListingPageTask
	}
	if err := s.queue.Publish(ctx, task); err != nil {
		return queued, err
	}
	return queued, nil
}

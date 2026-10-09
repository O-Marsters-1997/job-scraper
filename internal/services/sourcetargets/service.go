// Package sourcetargets is the jobsearch context's source-target feature:
// what to scrape, on what cadence, and starting a run (ADR 0011).
package sourcetargets

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/runwindow"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
)

type QueuePublisher interface {
	Publish(ctx context.Context, task queue.Task) error
	EnqueueJobs(ctx context.Context, jobs []dto.QueuedJob) error
}

type Store interface {
	CandidateStore
	CreateSourceTarget(ctx context.Context, userID, source, value string, enabled bool, filters map[string]string, window dto.RunWindow, nextRunAt *time.Time) (dto.SourceTarget, error)
	CreateSourceTargetWithRun(ctx context.Context, userID, source, value string, enabled bool, filters map[string]string, window dto.RunWindow, nextRunAt *time.Time) (dto.SourceTarget, error)
	UpdateSourceTarget(ctx context.Context, id, userID string, enabled *bool, window *dto.RunWindow, nextRunAt *time.Time) (dto.SourceTarget, error)
	DeleteSourceTarget(ctx context.Context, id, userID string) error
	ListSourceTargetsByUser(ctx context.Context, userID string) ([]dto.SourceTarget, error)
	StartSourceTargetRun(ctx context.Context, id string, nextRunAt *time.Time) (dto.SourceTarget, error)
	GetSourceTarget(ctx context.Context, id string) (dto.SourceTarget, error)
	TransitionSourceTargetRun(ctx context.Context, id, runID, status, runError string) (dto.SourceTarget, error)
	DisableSourceTargets(ctx context.Context, source, reason string) (int64, error)
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
	now     func() time.Time
	jitter  func(time.Duration) time.Duration
}

func New(targets Store, configs SearchConfigReader, q QueuePublisher, boards CardBoards) *Service {
	return &Service{targets: targets, configs: configs, queue: q, boards: boards, now: time.Now, jitter: runwindow.Jitter}
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
			in.Filters = search.Filters
		}
	}

	if err := validateSourceValue(in.Source, in.Value, in.Filters); err != nil {
		return dto.SourceTarget{}, err
	}

	filters := in.Filters
	if filters == nil {
		filters = map[string]string{}
	}
	if in.Source == "linkedin" && filters["recency"] == "" {
		filters["recency"] = "r604800"
	}

	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}

	window, err := createWindow(in.RunWindow)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	var nextRunAt *time.Time
	if enabled {
		nextRunAt = s.nextRun(window)
	}

	startNow := enabled
	create := s.targets.CreateSourceTarget
	if startNow {
		create = s.targets.CreateSourceTargetWithRun
	}
	target, err := create(ctx, userID, in.Source, in.Value, enabled, filters, window, nextRunAt)
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

func createWindow(in dto.RunWindowInput) (dto.RunWindow, error) {
	var window dto.RunWindow
	switch {
	case !in.Set:
		window = runwindow.Default()
	case in.Window == nil:
		window = manualWindow()
	default:
		window = *in.Window
	}
	if err := runwindow.Validate(window); err != nil {
		return dto.RunWindow{}, apperr.Invalid("run_window: " + err.Error())
	}
	return window, nil
}

func manualWindow() dto.RunWindow {
	w := runwindow.Default()
	w.IntervalMinutes = nil
	return w
}

func (s *Service) nextRun(w dto.RunWindow) *time.Time {
	next, ok := runwindow.Next(w, s.now(), s.jitter)
	if !ok {
		return nil
	}
	return &next
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

// Update changes a source target's enabled state and/or Run Window. A
// run_window of null makes the target manual. If it enables a discovery
// target, saved candidates are reconsidered against the user's current search
// config.
func (s *Service) Update(ctx context.Context, userID string, in dto.UpdateSourceTargetInput) (dto.SourceTarget, error) {
	if in.Enabled == nil && !in.RunWindow.Set {
		return dto.SourceTarget{}, apperr.Invalid("nothing to update")
	}

	var window *dto.RunWindow
	var nextRunAt *time.Time
	if in.RunWindow.Set {
		w, err := s.updatedWindow(ctx, userID, in)
		if err != nil {
			return dto.SourceTarget{}, err
		}
		window = &w
		nextRunAt = s.nextRun(w)
	} else if in.Enabled != nil && *in.Enabled {
		current, err := s.owned(ctx, userID, in.ID)
		if err != nil {
			return dto.SourceTarget{}, err
		}
		window = &current.RunWindow
		nextRunAt = s.nextRun(current.RunWindow)
	}

	target, err := s.targets.UpdateSourceTarget(ctx, in.ID, userID, in.Enabled, window, nextRunAt)
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

func (s *Service) updatedWindow(ctx context.Context, userID string, in dto.UpdateSourceTargetInput) (dto.RunWindow, error) {
	if in.RunWindow.Window != nil {
		if err := runwindow.Validate(*in.RunWindow.Window); err != nil {
			return dto.RunWindow{}, apperr.Invalid("run_window: " + err.Error())
		}
		return *in.RunWindow.Window, nil
	}
	current, err := s.owned(ctx, userID, in.ID)
	if err != nil {
		return dto.RunWindow{}, err
	}
	current.RunWindow.IntervalMinutes = nil
	return current.RunWindow, nil
}

func (s *Service) owned(ctx context.Context, userID, id string) (dto.SourceTarget, error) {
	targets, err := s.targets.ListSourceTargetsByUser(ctx, userID)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	for _, target := range targets {
		if target.ID == id {
			return target, nil
		}
	}
	return dto.SourceTarget{}, apperr.NotFound("not found")
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
	target, err := s.owned(ctx, userID, id)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	if target.RunStatus == "queued" || target.RunStatus == "running" {
		return dto.SourceTarget{}, apperr.Conflict("search already in progress")
	}
	queued, err := s.enqueueRun(ctx, target)
	if err != nil {
		return dto.SourceTarget{}, apperr.Unavailable("could not start search")
	}
	return queued, nil
}

func (s *Service) enqueueRun(ctx context.Context, target dto.SourceTarget) (dto.SourceTarget, error) {
	queued, err := s.targets.StartSourceTargetRun(ctx, target.ID, s.nextRun(target.RunWindow))
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

// DisableSource turns off every enabled target of source for all users and
// returns how many it changed. In-flight runs are failed with the reason.
func (s *Service) DisableSource(ctx context.Context, source, reason string) (int64, error) {
	disabled, err := s.targets.DisableSourceTargets(ctx, source, reason)
	if err != nil {
		return 0, err
	}
	TargetsDisabled.WithLabelValues(source, reason).Add(float64(disabled))
	return disabled, nil
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

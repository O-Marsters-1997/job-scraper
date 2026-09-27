package scoring

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/extract"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
	"github.com/ollymarsters/job-scraper/internal/services/notify"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/store"
	"github.com/ollymarsters/job-scraper/internal/services/scoringconfig"
)

// ErrNotFound is the scoring store's not-found sentinel, re-exported for
// contexts that must match it (ADR 0011).
var ErrNotFound = store.ErrNotFound

// Reconsiderer re-evaluates a user's discovered-job candidates against an
// updated Search Config.
type Reconsiderer interface {
	Reconsider(ctx context.Context, config dto.SearchConfig) error
}

type noopAlerter struct{}

func (noopAlerter) NotifyNewJob(context.Context, dto.Job, string) error { return nil }

type Module struct {
	store         *store.Store
	scoring       *Service
	scoringConfig *scoringconfig.Service
}

// New wires the scoring context: its own store, the answer-effect loop and
// Search Config orchestration. notifyAPIKey empty disables new-job email
// alerts (dev default).
func New(pool *pgxpool.Pool, credentials Credentials, profiles ProfileReader, candidates Reconsiderer, notifyAPIKey, notifyFrom string) *Module {
	st := store.New(pool)

	var alerter Alerter = noopAlerter{}
	if notifyAPIKey != "" {
		renderer, err := notify.NewRenderer()
		if err != nil {
			slog.Error("notify templates unavailable", slog.Any("err", err))
		} else {
			alerter = notify.NewNotificationService(notify.NewResendNotifier(notifyAPIKey, notifyFrom), renderer)
		}
	}

	mainService := NewService(st, jev.NewClient(), credentials, alerter, profiles)
	scoringConfig := scoringconfig.New(st, candidates, mainService, extract.NewClient(), credentials)

	return &Module{store: st, scoring: mainService, scoringConfig: scoringConfig}
}

// Run drains the answer-effect queue until ctx is cancelled.
func (m *Module) Run(ctx context.Context) error {
	return m.scoring.Run(ctx)
}

// SearchConfig returns userID's Search Config, satisfied by the scoring
// store; the worker's reject filter and admin's option commands read it
// through this facade (ADR 0011).
func (m *Module) SearchConfig(ctx context.Context, userID string) (dto.SearchConfig, error) {
	return m.store.GetSearchConfig(ctx, userID)
}

// OpsState satisfies telemetry.StateReader.
func (m *Module) OpsState(ctx context.Context) (dto.OpsState, error) {
	return m.store.OpsState(ctx)
}

func (m *Module) AddOption(ctx context.Context, id, dimension, label, question string) error {
	return m.store.AddScoringOption(ctx, id, dimension, label, question)
}

func (m *Module) RewordOption(ctx context.Context, id, question string) error {
	return m.store.RewordScoringOption(ctx, id, question)
}

func (m *Module) RetireOption(ctx context.Context, id string) error {
	return m.store.RetireScoringOption(ctx, id)
}

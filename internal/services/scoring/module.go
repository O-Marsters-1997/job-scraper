package scoring

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/extract"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
	"github.com/ollymarsters/job-scraper/internal/services/notify"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/store"
	"github.com/ollymarsters/job-scraper/internal/services/scoringconfig"
	"github.com/ollymarsters/job-scraper/internal/telemetry"
)

var _ telemetry.StateReader = (*Module)(nil)

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
func New(pool *pgxpool.Pool, credentials Credentials, profiles ProfileReader, candidates scoringconfig.Reconsiderer, notifyAPIKey, notifyFrom string) *Module {
	st := store.New(pool)

	var alerter Alerter = noopAlerter{}
	if notifyAPIKey != "" {
		renderer, err := notify.NewRenderer()
		if err != nil {
			slog.Error("notify templates unavailable", slog.Any(logger.KeyErr, err))
		} else {
			alerter = notify.NewNotificationService(notify.NewResendNotifier(notifyAPIKey, notifyFrom), renderer)
		}
	}

	mainService := NewService(st, jev.NewClient(), credentials, alerter, profiles)
	scoringConfig := scoringconfig.New(st, candidates, mainService, extract.NewClient(), credentials, mainService)

	return &Module{store: st, scoring: mainService, scoringConfig: scoringConfig}
}

// NewFacade wires only the scoring store, for cmd/admin's option commands
// and the worker's reject filter (ADR 0011). Run and the scoringConfig
// routes panic on a Module built this way.
func NewFacade(pool *pgxpool.Pool) *Module {
	return &Module{store: store.New(pool)}
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

// JobsChanged is scoring's tx-scoped port for a job's content changing:
// it drops stale cached answers and queues a fresh answer effect, within
// the caller's own transaction (ADR 0011).
func (m *Module) JobsChanged(ctx context.Context, tx pgx.Tx, jobIDs []string, firstDiscovery bool) error {
	return m.store.JobsChanged(ctx, tx, jobIDs, firstDiscovery)
}

// JobsClosed is scoring's tx-scoped port for jobs closing: it drops their
// cached answers within the caller's own transaction (ADR 0011).
func (m *Module) JobsClosed(ctx context.Context, tx pgx.Tx, jobIDs []string) error {
	return m.store.JobsClosed(ctx, tx, jobIDs)
}

// CompanyTracked is scoring's tx-scoped port for a user tracking a company:
// it queues scores for the company's known open Jobs within the caller's
// own transaction, without alerting (ADR 0011).
func (m *Module) CompanyTracked(ctx context.Context, tx pgx.Tx, userID, companyID string) error {
	return m.store.CompanyTracked(ctx, tx, userID, companyID)
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

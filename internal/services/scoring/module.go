package scoring

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/extract"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
	"github.com/ollymarsters/job-scraper/internal/services/notify"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/store"
	"github.com/ollymarsters/job-scraper/internal/telemetry"
)

var _ telemetry.StateReader = (*Module)(nil)

type noopAlerter struct{}

func (noopAlerter) NotifyNewJob(context.Context, dto.Job, string) error { return nil }

type Module struct {
	store Store
	svc   *Service
}

// Deps are the stores and collaborators Build wires into the Module; New
// builds the real ones and calls Build. Tests call Build directly with
// fakes (ADR 0012).
type Deps struct {
	Store        Store
	Answerer     Answerer
	Credentials  Credentials
	Alerter      Alerter
	Profiles     ProfileReader
	Candidates   Reconsiderer
	Extractor    Extractor
	TickInterval time.Duration
}

func Build(deps Deps) *Module {
	svc := NewService(deps)
	return &Module{store: deps.Store, svc: svc}
}

// New wires the scoring context: its own store, the answer-effect loop and
// Search Config orchestration. notifyAPIKey empty disables new-job email
// alerts (dev default).
func New(pool *pgxpool.Pool, credentials Credentials, profiles ProfileReader, candidates Reconsiderer, notifyAPIKey, notifyFrom string) *Module {
	st := store.New(pool)

	var alerter Alerter = noopAlerter{}
	if notifyAPIKey != "" {
		alerter = notify.NewNotificationService(notify.NewResendNotifier(notifyAPIKey, notifyFrom))
	}

	return Build(Deps{
		Store: st, Answerer: jev.NewClient(), Credentials: credentials,
		Alerter: alerter, Profiles: profiles, Candidates: candidates, Extractor: extract.NewClient(),
	})
}

func NewFacade(pool *pgxpool.Pool) *Module {
	return &Module{store: store.New(pool)}
}

// Run drains the answer-effect queue until ctx is cancelled.
func (m *Module) Run(ctx context.Context) error {
	return m.svc.Run(ctx)
}

// SearchConfig returns userID's Search Config, satisfied by the scoring
// store; the worker's reject filter and admin's option commands read it
// through this facade (ADR 0011).
func (m *Module) SearchConfig(ctx context.Context, userID string) (dto.SearchConfig, error) {
	return m.store.GetSearchConfig(ctx, userID)
}

// IncludeFilterConfigs returns the Search Config of every User with at least
// one include filter set; Users without one never match a Discovered Board.
func (m *Module) IncludeFilterConfigs(ctx context.Context) ([]dto.SearchConfig, error) {
	return m.store.ListIncludeFilterConfigs(ctx)
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

// Ask answers arbitrary questions about a job, reusing cached Jev answers
// and billing the rest to userID's OpenRouter key.
func (m *Module) Ask(ctx context.Context, userID, jobID string, questions []string) (map[string]dto.Answer, error) {
	return m.svc.Ask(ctx, userID, jobID, questions)
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

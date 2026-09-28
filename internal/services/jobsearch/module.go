package jobsearch

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/services/candidates"
	"github.com/ollymarsters/job-scraper/internal/services/companies"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
	"github.com/ollymarsters/job-scraper/internal/services/sources"
	"github.com/ollymarsters/job-scraper/internal/services/sourcetargets"
)

// ErrBoardClaimUnavailable is the jobsearch store's sentinel, re-exported
// for the worker to match (ADR 0011).
var ErrBoardClaimUnavailable = store.ErrBoardClaimUnavailable

type Module struct {
	store         Store
	jobs          *Service
	companies     *companies.Service
	sourceTargets *sourcetargets.Service
	sources       *sources.Service
	candidates    *candidates.Service
	ingest        *Ingester
}

// ScoringPort is scoring's facade as jobsearch needs it: Search Config
// reads for source-target filtering, plus the tx-scoped write ports jobsearch
// calls instead of writing scoring's tables directly (ADR 0011).
type ScoringPort interface {
	sourcetargets.SearchConfigReader
	store.ScoringWriter
}

// QueuePublisher is the queue as jobsearch's sibling feature services need
// it: companies and sourcetargets publish worker tasks, candidates enqueues
// detail-fetch jobs.
type QueuePublisher interface {
	Publish(ctx context.Context, task queue.Task) error
	EnqueueJobs(ctx context.Context, jobs []dto.QueuedJob) error
}

// Store is the full surface jobsearch's own service, ingester and sibling
// feature services (companies, sourcetargets, candidates) read and write.
// store.Store satisfies it; jobsearchtest.FakeStore proves it via
// RunStoreContract (ADR 0012).
type Store interface {
	candidates.Store
	companies.Store
	companies.SourceTargets
	sourcetargets.Store

	Page(ctx context.Context, userID string, options dto.JobPageOptions) (dto.JobPage, error)
	GetJob(ctx context.Context, jobID, userID string) (dto.Job, error)
	ListJobs(ctx context.Context, userID string) ([]dto.Job, error)
	NewURLs(ctx context.Context, urls []string) ([]string, error)
	SaveCanonical(ctx context.Context, job dto.Job) (dto.Job, string, error)
	ListCompaniesForUser(ctx context.Context, userID string) ([]dto.Company, error)
}

// Deps are the store and collaborators Build wires into the Module; New
// builds the real store and calls Build. Tests call Build directly with
// fakes (ADR 0012).
type Deps struct {
	Store   Store
	Scoring ScoringPort
	Queue   QueuePublisher
}

func Build(deps Deps) *Module {
	cand := candidates.New(deps.Store, deps.Queue)
	return &Module{
		store:         deps.Store,
		jobs:          NewService(deps.Store),
		companies:     companies.New(deps.Store, deps.Store, deps.Queue),
		sourceTargets: sourcetargets.New(deps.Store, deps.Scoring, cand, deps.Queue),
		sources:       sources.New(),
		candidates:    cand,
		ingest:        newIngester(deps.Store, deps.Store),
	}
}

func New(pool *pgxpool.Pool, q *queue.Broker, scoring ScoringPort) *Module {
	return Build(Deps{Store: store.New(pool, scoring), Scoring: scoring, Queue: q})
}

// Reconsider re-evaluates the caller's saved candidates against cfg;
// legacy scoringconfig calls this through its own Reconsiderer interface
// (ADR 0011).
func (m *Module) Reconsider(ctx context.Context, cfg dto.SearchConfig) error {
	return m.candidates.Reconsider(ctx, cfg)
}

// Boards exposes company and board management to other contexts, the
// worker's board poller and crawler, and cmd/admin (ADR 0011 Phase 9).
func (m *Module) Boards() *companies.Service { return m.companies }

// Targets exposes source-target management to other contexts, the worker's
// run recovery, and cmd/admin (ADR 0011 Phase 9).
func (m *Module) Targets() *sourcetargets.Service { return m.sourceTargets }

// Catalog exposes job lookups to other contexts, the worker's new-URL
// check, and cmd/admin (ADR 0011 Phase 9).
func (m *Module) Catalog() *Service { return m.jobs }

// Candidates exposes candidate capture to the worker's scrape orchestrator
// (ADR 0011 Phase 9).
func (m *Module) Candidates() *candidates.Service { return m.candidates }

// DeleteExpiredCandidates is called by the worker's daily cleanup.
func (m *Module) DeleteExpiredCandidates(ctx context.Context) error {
	return m.candidates.DeleteExpired(ctx)
}

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
	store         *store.Store
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

func New(pool *pgxpool.Pool, q *queue.Broker, scoring ScoringPort) *Module {
	st := store.New(pool, scoring)
	cand := candidates.New(st, q)
	return &Module{
		store:         st,
		jobs:          NewService(st),
		companies:     companies.New(st, st, q),
		sourceTargets: sourcetargets.New(st, scoring, cand, q),
		sources:       sources.New(),
		candidates:    cand,
		ingest:        newIngester(st, st),
	}
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

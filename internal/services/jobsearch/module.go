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

type Module struct {
	store         *store.Store
	jobs          *Service
	companies     *companies.Service
	sourceTargets *sourcetargets.Service
	sources       *sources.Service
	candidates    *candidates.Service
	ingest        *Ingester
}

// New builds the jobsearch context. configs is scoring's search config
// reader (ADR 0011 migration order).
func New(pool *pgxpool.Pool, q *queue.Broker, configs sourcetargets.SearchConfigReader) *Module {
	st := store.New(pool)
	cand := candidates.New(st, q)
	return &Module{
		store:         st,
		jobs:          NewService(st),
		companies:     companies.New(st, st, q),
		sourceTargets: sourcetargets.New(st, configs, cand, q),
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

// Boards exposes company and board management to other contexts and
// cmd/admin (ADR 0011 Phase 9; the worker doesn't consume this yet — #268).
func (m *Module) Boards() *companies.Service { return m.companies }

// Targets exposes source-target management to other contexts and cmd/admin
// (ADR 0011 Phase 9; the worker doesn't consume this yet — #268).
func (m *Module) Targets() *sourcetargets.Service { return m.sourceTargets }

// Catalog exposes job lookups to other contexts and cmd/admin (ADR 0011
// Phase 9; the worker doesn't consume this yet — #268).
func (m *Module) Catalog() *Service { return m.jobs }

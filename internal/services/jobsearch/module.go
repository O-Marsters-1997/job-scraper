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

type ScoringPort interface {
	sourcetargets.SearchConfigReader
	store.ScoringWriter
}

type QueuePublisher interface {
	Publish(ctx context.Context, task queue.Task) error
	EnqueueJobs(ctx context.Context, jobs []dto.QueuedJob) error
}

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

func (m *Module) Reconsider(ctx context.Context, cfg dto.SearchConfig) error {
	return m.candidates.Reconsider(ctx, cfg)
}

func (m *Module) Boards() *companies.Service { return m.companies }

func (m *Module) Targets() *sourcetargets.Service { return m.sourceTargets }

func (m *Module) Catalog() *Service { return m.jobs }

func (m *Module) Candidates() *candidates.Service { return m.candidates }

func (m *Module) DeleteExpiredCandidates(ctx context.Context) error {
	return m.candidates.DeleteExpired(ctx)
}

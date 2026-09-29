package jobsearch

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
	"github.com/ollymarsters/job-scraper/internal/services/sourcetargets"
)

var ErrBoardClaimUnavailable = store.ErrBoardClaimUnavailable

type Module struct {
	store         Store
	jobs          *Service
	sourceTargets *sourcetargets.Service
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
	Page(ctx context.Context, userID string, options dto.JobPageOptions) (dto.JobPage, error)
	GetJob(ctx context.Context, jobID, userID string) (dto.Job, error)
	ListJobs(ctx context.Context, userID string) ([]dto.Job, error)
	NewURLs(ctx context.Context, urls []string) ([]string, error)
	SaveCanonical(ctx context.Context, job dto.Job) (dto.Job, string, error)
	ListCompaniesForUser(ctx context.Context, userID string) ([]dto.Company, error)
	UpsertCompany(ctx context.Context, c dto.CompanyUpsert) (dto.Company, error)
	GetCompany(ctx context.Context, id string) (dto.Company, error)
	ListCompanyBoards(ctx context.Context, companyID string) ([]dto.CompanyBoard, error)
	UpsertCandidateBoard(ctx context.Context, companyID, source, token string) (dto.CompanyBoard, error)
	SetCompanyTracking(ctx context.Context, userID, companyID string, enabled bool, checkIntervalMinutes int) (dto.CompanyTracking, error)
	VerifyCompanyBoard(ctx context.Context, companyID, source, token, method string) (dto.CompanyBoard, error)
	ListCompaniesToCrawl(ctx context.Context, limit int) ([]dto.Company, error)
	TouchCompanyCrawled(ctx context.Context, id string) error
	ListDueBoards(ctx context.Context) ([]dto.BoardPoll, error)
	ListActiveBoards(ctx context.Context) ([]dto.BoardPoll, error)
	ClaimBoard(ctx context.Context, id string, manual bool) (dto.BoardPoll, error)
	CompleteBoard(ctx context.Context, snapshot dto.BoardSnapshot) error
	FailBoard(ctx context.Context, poll dto.BoardPoll) error
	GetVerifiedBoardID(ctx context.Context, source, token string) (string, error)
	GetLastScraped(ctx context.Context, source string) (time.Time, bool, error)
	SetLastScraped(ctx context.Context, source string) error
}

type Deps struct {
	Store         Store
	SourceTargets sourcetargets.Store
	Scoring       ScoringPort
	Queue         QueuePublisher
}

func Build(deps Deps) *Module {
	return &Module{
		store:         deps.Store,
		jobs:          NewService(deps.Store, deps.Queue),
		sourceTargets: sourcetargets.New(deps.SourceTargets, deps.Scoring, deps.Queue),
		ingest:        newIngester(deps.Store, deps.Store),
	}
}

func New(pool *pgxpool.Pool, q *queue.Broker, scoring ScoringPort) *Module {
	st := store.New(pool, scoring)
	return Build(Deps{
		Store:         st,
		SourceTargets: st,
		Scoring:       scoring,
		Queue:         q,
	})
}

func (m *Module) Reconsider(ctx context.Context, cfg dto.SearchConfig) error {
	return m.sourceTargets.Reconsider(ctx, cfg)
}

func (m *Module) Boards() *Service { return m.jobs }

func (m *Module) Targets() *sourcetargets.Service { return m.sourceTargets }

func (m *Module) Catalog() *Service { return m.jobs }

func (m *Module) DeleteExpiredCandidates(ctx context.Context) error {
	return m.sourceTargets.DeleteExpired(ctx)
}

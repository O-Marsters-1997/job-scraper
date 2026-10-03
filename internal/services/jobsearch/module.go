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
	queue         QueuePublisher
	scoring       ScoringPort
}

type ScoringPort interface {
	sourcetargets.SearchConfigReader
	CompanyProfiles(ctx context.Context, userID string, companyIDs []string) (map[string][]dto.CompanyProfileEntry, error)
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
	PageCompaniesForUser(ctx context.Context, userID string, options dto.CompanyPageOptions) (dto.CompanyPage, error)
	GetCompanyForUser(ctx context.Context, userID, id string) (dto.Company, error)
	ListTrackedCompaniesForUser(ctx context.Context, userID string) ([]dto.TrackedCompany, error)
	ListNewCompanies(ctx context.Context, userID string) ([]dto.NewCompany, error)
	ListNewCompanyJobs(ctx context.Context, userID string) ([]dto.Job, error)
	SaveCompanyProfile(ctx context.Context, companyID, source string, profile dto.CompanyProfile) error
	DeleteCompanyTracking(ctx context.Context, userID, companyID string) error
	UpsertCompany(ctx context.Context, c dto.CompanyUpsert) (dto.Company, error)
	GetCompany(ctx context.Context, id string) (dto.Company, error)
	ListCompanyBoards(ctx context.Context, companyID string) ([]dto.CompanyBoard, error)
	UpsertCandidateBoard(ctx context.Context, companyID, source, token string) (dto.CompanyBoard, error)
	SetCompanyTracking(ctx context.Context, userID, companyID string, enabled bool, checkIntervalMinutes int) (dto.CompanyTracking, error)
	TrackDiscoveredCompany(ctx context.Context, userID, companyID string) (bool, error)
	SetCompanyReviewState(ctx context.Context, userID, companyID, state string) (dto.CompanyTracking, error)
	VerifyCompanyBoard(ctx context.Context, companyID, source, token, method, via string) (dto.CompanyBoard, error)
	ListCompaniesToCrawl(ctx context.Context, limit int) ([]dto.Company, error)
	TouchCompanyCrawled(ctx context.Context, id string) error
	RenameCompany(ctx context.Context, id, name string) error
	ListDueBoards(ctx context.Context) ([]dto.BoardPoll, error)
	ListActiveBoards(ctx context.Context) ([]dto.BoardPoll, error)
	ClaimBoard(ctx context.Context, id string, manual bool) (dto.BoardPoll, error)
	CompleteBoard(ctx context.Context, snapshot dto.BoardSnapshot) error
	FailBoard(ctx context.Context, poll dto.BoardPoll) error
	ListUntrackedDiscoveredBoards(ctx context.Context) ([]dto.CompanyBoard, error)
	ListVerifiedBoardsBySlug(ctx context.Context, slugs []string) ([]dto.CardBoard, error)
	ListVerifiedCompanySlugs(ctx context.Context, slugs []string) ([]string, error)
	GetBoardCompanyID(ctx context.Context, source, token string) (string, error)
	GetVerifiedBoardID(ctx context.Context, source, token string) (string, error)
	GetLastScraped(ctx context.Context, source string) (time.Time, bool, error)
	SetLastScraped(ctx context.Context, source string) error
	LookupFetch(ctx context.Context, url string) (dto.CachedResponse, bool, error)
	PutFetch(ctx context.Context, resp dto.CachedResponse) error
	ForgetFetches(ctx context.Context, urls []string) error
	DeleteExpiredFetches(ctx context.Context) error
}

type Deps struct {
	Store         Store
	SourceTargets sourcetargets.Store
	Scoring       ScoringPort
	Queue         QueuePublisher
}

func Build(deps Deps) *Module {
	jobs := NewService(deps.Store, deps.Queue)
	return &Module{
		store:         deps.Store,
		jobs:          jobs,
		sourceTargets: sourcetargets.New(deps.SourceTargets, deps.Scoring, deps.Queue, jobs),
		queue:         deps.Queue,
		scoring:       deps.Scoring,
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

func (m *Module) PublishBoardChecks(ctx context.Context, manual bool) error {
	return m.jobs.PublishBoardChecks(ctx, manual)
}

// TrackDiscoveredCompany tracks the Company for userID as `new`, reporting
// whether it did; an existing row of any review state is left untouched.
func (m *Module) TrackDiscoveredCompany(ctx context.Context, userID, companyID string) (bool, error) {
	return m.store.TrackDiscoveredCompany(ctx, userID, companyID)
}

// SaveCompanyProfile upserts the profile source reports for companyID.
func (m *Module) SaveCompanyProfile(ctx context.Context, companyID, source string, profile dto.CompanyProfile) error {
	return m.store.SaveCompanyProfile(ctx, companyID, source, profile)
}

func (m *Module) Boards() Store { return m.store }

func (m *Module) Targets() *sourcetargets.Service { return m.sourceTargets }

func (m *Module) DeleteExpiredCandidates(ctx context.Context) error {
	return m.sourceTargets.DeleteExpired(ctx)
}

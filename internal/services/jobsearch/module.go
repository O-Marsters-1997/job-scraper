package jobsearch

import (
	"cmp"
	"context"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
	"github.com/ollymarsters/job-scraper/internal/services/sourcetargets"
)

var ErrBoardClaimUnavailable = store.ErrBoardClaimUnavailable

// RegisterMetrics registers the jobsearch context's Prometheus metrics.
func RegisterMetrics(reg prometheus.Registerer) {
	sourcetargets.RegisterMetrics(reg)
}

type Module struct {
	store         Store
	jobs          *Service
	sourceTargets *sourcetargets.Service
	queue         QueuePublisher
	scoring       ScoringPort
	claimLimit    int
}

const (
	defaultClaimLimit = 10
	maxClaimLimit     = 1000
)

type ScoringPort interface {
	sourcetargets.SearchConfigReader
	CompanyProfiles(ctx context.Context, userID string, companyIDs []string) (map[string][]dto.CompanyProfileEntry, error)
	store.ScoringWriter
	ExcludeCompany(ctx context.Context, userID, name string) (bool, error)
	UnexcludeCompany(ctx context.Context, userID, name string) error
}

type QueuePublisher interface {
	Publish(ctx context.Context, task queue.Task) error
	EnqueueJobs(ctx context.Context, jobs []dto.QueuedJob) error
}

type Store interface {
	Page(ctx context.Context, userID string, options dto.JobPageOptions) (dto.JobPage, error)
	GetJob(ctx context.Context, jobID, userID string) (dto.Job, error)
	ListJobListings(ctx context.Context, jobID string) ([]dto.JobListing, error)
	ListJobs(ctx context.Context, userID string, excludedCompanySlugs []string) ([]dto.Job, error)
	MarkJobsSeen(ctx context.Context, userID string, jobIDs []string, seen bool) error
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
	SetCompanyFavourite(ctx context.Context, userID, companyID string, favourite bool) error
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
	ClaimLimit    int

	MaxAutomaticTargets int
}

func Build(deps Deps) *Module {
	jobs := NewService(deps.Store, deps.Queue, deps.Scoring)
	maxAutomatic := deps.MaxAutomaticTargets
	if maxAutomatic == 0 {
		maxAutomatic = sourcetargets.DefaultMaxAutomatic
	}
	return &Module{
		store:         deps.Store,
		jobs:          jobs,
		sourceTargets: sourcetargets.New(deps.SourceTargets, deps.Scoring, deps.Queue, jobs, maxAutomatic),
		queue:         deps.Queue,
		scoring:       deps.Scoring,
		claimLimit:    cmp.Or(deps.ClaimLimit, defaultClaimLimit),
	}
}

func New(pool *pgxpool.Pool, q *queue.Broker, scoring ScoringPort, maxAutomaticTargets int) *Module {
	st := store.New(pool, scoring)
	return Build(Deps{
		Store:         st,
		SourceTargets: st,
		Scoring:       scoring,
		Queue:         q,
		ClaimLimit:    claimLimitFromEnv(),

		MaxAutomaticTargets: maxAutomaticTargets,
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

func claimLimitFromEnv() int {
	limit, err := strconv.Atoi(os.Getenv("DISCOVERY_CLAIM_LIMIT"))
	if err != nil || limit < 1 || limit > maxClaimLimit {
		return defaultClaimLimit
	}
	return limit
}

// PublishDueTargets starts and publishes a run for each saved search whose
// Run Window has come due, at most DISCOVERY_CLAIM_LIMIT per call.
func (m *Module) PublishDueTargets(ctx context.Context) error {
	return m.sourceTargets.PublishDue(ctx, m.claimLimit)
}

package jobsearch

import (
	"cmp"
	"context"
	"log/slog"
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
	proxyUsage    *ProxyUsage
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
	JobStats(ctx context.Context, userID string, options dto.JobStatsOptions) (dto.JobStats, error)
	JobInsightGroups(ctx context.Context, userID string, options dto.JobInsightsOptions) ([]dto.JobInsightGroup, error)
	Page(ctx context.Context, userID string, options dto.JobPageOptions) (dto.JobPage, error)
	GetJob(ctx context.Context, jobID, userID string) (dto.Job, error)
	ListJobListings(ctx context.Context, jobID string) ([]dto.JobListing, error)
	FindJobByPosting(ctx context.Context, url, source, token, postingID string) (dto.Job, error)
	MarkJobsSeen(ctx context.Context, userID string, jobIDs []string, seen bool) error
	NewURLs(ctx context.Context, urls []string) ([]string, error)
	SaveCanonical(ctx context.Context, job dto.Job) (dto.Job, string, error)
	PageCompaniesForUser(ctx context.Context, userID string, options dto.CompanyPageOptions) (dto.CompanyPage, error)
	CountCompaniesByATS(ctx context.Context) ([]dto.ATSCount, error)
	GetCompanyForUser(ctx context.Context, userID, id string) (dto.Company, error)
	ListTrackedCompaniesForUser(ctx context.Context, userID string) ([]dto.TrackedCompany, error)
	ListNewCompanies(ctx context.Context, userID string, limit, offset int32) (dto.NewCompanyPage, error)
	ListNewCompanyJobs(ctx context.Context, userID string, companyIDs []string) ([]dto.Job, error)
	SaveCompanyProfile(ctx context.Context, companyID, source string, profile dto.CompanyProfile) error
	DeleteCompanyTracking(ctx context.Context, userID, companyID string) error
	UpsertCompany(ctx context.Context, c dto.CompanyUpsert) (dto.Company, error)
	GetCompany(ctx context.Context, id string) (dto.Company, error)
	ListCompanyBoards(ctx context.Context, companyID string) ([]dto.CompanyBoard, error)
	UpsertCandidateBoard(ctx context.Context, companyID, source, token string) (dto.CompanyBoard, error)
	SetCompanyTracking(ctx context.Context, userID, companyID string, enabled bool, checkIntervalMinutes int) (dto.CompanyTracking, error)
	TrackDiscoveredCompany(ctx context.Context, userID, companyID string) (bool, error)
	SetCompanyFavourite(ctx context.Context, userID, companyID string, favourite bool) error
	SetCompanyKind(ctx context.Context, companyID string, kind dto.CompanyKind) error
	SuggestCompanyKind(ctx context.Context, companyID string, kind dto.CompanyKind) (bool, error)
	SetCompanyReviewState(ctx context.Context, userID, companyID, state string) (dto.CompanyTracking, error)
	VerifyCompanyBoard(ctx context.Context, companyID, source, token, method, via string) (dto.CompanyBoard, error)
	ListCompaniesToCrawl(ctx context.Context, limit int) ([]dto.Company, error)
	TouchCompanyCrawled(ctx context.Context, id string) error
	RenameCompany(ctx context.Context, id, name string) error
	ListDueBoards(ctx context.Context) ([]dto.BoardPoll, error)
	MarkBoardCheckPublished(ctx context.Context, id string) error
	ClearBoardCheckPublished(ctx context.Context, id string) error
	ListActiveBoards(ctx context.Context) ([]dto.BoardPoll, error)
	ClaimBoard(ctx context.Context, id string, manual bool) (dto.BoardPoll, error)
	CompleteBoard(ctx context.Context, snapshot dto.BoardSnapshot) ([]dto.JobChange, error)
	FailBoard(ctx context.Context, poll dto.BoardPoll) error
	ListUntrackedDiscoveredBoards(ctx context.Context, uk bool, ukSince time.Time) ([]dto.CompanyBoard, error)
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
	ScrapeRunPageDone(ctx context.Context, runID string) error
	ScrapeRunDeferred(ctx context.Context, runID string) error
	ScrapeRunAttemptFailed(ctx context.Context, runID string, page, attempt int, message string) error
	FinishScrapeRun(ctx context.Context, runID string, stop dto.StopReason) error
	BoardRunAttemptFailed(ctx context.Context, run dto.BoardRunStart, attempt int, message string) error
	FinishBoardRun(ctx context.Context, run dto.BoardRunStart, stop dto.StopReason, tally dto.BoardTally) error
	DeleteExpiredScrapeRuns(ctx context.Context) error
	ListScrapeRuns(ctx context.Context, userID string, opts dto.ScrapeRunListOptions) ([]dto.ScrapeRun, error)
	ListScrapeRunJobSnapshots(ctx context.Context, userID, runID string) ([]dto.ScrapeRunJobSnapshot, error)
	ListSourceHealth(ctx context.Context, userID string) ([]dto.SourceHealth, error)
	ListRecentNewJobs(ctx context.Context, userID string, since time.Time) ([]dto.SourceNewJob, error)
	ScrapeRunSummary(ctx context.Context, userID string, opts dto.ScrapeRunSummaryOptions) (dto.ScrapeRunSummary, error)
	MarkBoardUKJobsSeen(ctx context.Context, source, token string) error
	RecordDiscoverMiss(ctx context.Context, source, token string) error
	DiscoverMissedSince(ctx context.Context, source, token string, since time.Time) (bool, error)
	HarvestSeenIDs(ctx context.Context, source string) ([]string, error)
	RecordHarvestSeenIDs(ctx context.Context, source, outcome string, ids []string) error
	RetireHarvestSeenIDs(ctx context.Context, source string, ids []string) error
}

type Deps struct {
	Store         Store
	SourceTargets sourcetargets.Store
	Scoring       ScoringPort
	Queue         QueuePublisher
	ProxyQuota    QuotaFetcher
	Now           func() time.Time
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
		proxyUsage:    newProxyUsage(deps.ProxyQuota, deps.Now),
		claimLimit:    cmp.Or(deps.ClaimLimit, defaultClaimLimit),
	}
}

func New(pool *pgxpool.Pool, q *queue.Broker, scoring ScoringPort, proxyQuota QuotaFetcher, maxAutomaticTargets int, log *slog.Logger) *Module {
	st := store.New(pool, scoring, store.WithLogger(log))
	return Build(Deps{
		Store:         st,
		SourceTargets: st,
		Scoring:       scoring,
		Queue:         q,
		ProxyQuota:    proxyQuota,
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

func (m *Module) SuggestCompanyKind(ctx context.Context, companyID string, kind dto.CompanyKind) (bool, error) {
	return m.store.SuggestCompanyKind(ctx, companyID, kind)
}

// SaveCompanyProfile upserts the profile source reports for companyID.
func (m *Module) SaveCompanyProfile(ctx context.Context, companyID, source string, profile dto.CompanyProfile) error {
	return m.store.SaveCompanyProfile(ctx, companyID, source, profile)
}

// RefreshProxyUsage refreshes the in-memory proxy quota snapshot.
func (m *Module) RefreshProxyUsage(ctx context.Context) error {
	return m.proxyUsage.Refresh(ctx)
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

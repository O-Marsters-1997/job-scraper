package app

import (
	"os"

	"github.com/ollymarsters/job-scraper/internal/candidates"
	"github.com/ollymarsters/job-scraper/internal/credstore"
	jobsdb "github.com/ollymarsters/job-scraper/internal/data/db"
	igoogle "github.com/ollymarsters/job-scraper/internal/google"
	"github.com/ollymarsters/job-scraper/internal/ingest"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/services/aicredentials"
	"github.com/ollymarsters/job-scraper/internal/services/aiprefs"
	"github.com/ollymarsters/job-scraper/internal/services/applications"
	"github.com/ollymarsters/job-scraper/internal/services/applicationstatuses"
	authsvc "github.com/ollymarsters/job-scraper/internal/services/auth"
	"github.com/ollymarsters/job-scraper/internal/services/companies"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates"
	googlesvc "github.com/ollymarsters/job-scraper/internal/services/google"
	"github.com/ollymarsters/job-scraper/internal/services/jobs"
	"github.com/ollymarsters/job-scraper/internal/services/profile"
	"github.com/ollymarsters/job-scraper/internal/services/scoringconfig"
	"github.com/ollymarsters/job-scraper/internal/services/sources"
	"github.com/ollymarsters/job-scraper/internal/services/sourcetargets"
)

type services struct {
	auth                *authsvc.Service
	google              *googlesvc.Service
	googleClient        *igoogle.Client
	ingest              *ingest.Ingester
	jobs                *jobs.Service
	applicationStatuses *applicationstatuses.Service
	applications        *applications.Service
	sources             *sources.Service
	profile             *profile.Service
	scoringConfig       *scoringconfig.Service
	aiPrefs             *aiprefs.Service
	aiCredentials       *aicredentials.Service
	sourceTargets       *sourcetargets.Service
	companies           *companies.Service
	cvTemplates         *cvtemplates.Service
}

func newServices(db *jobsdb.DB, q *queue.Broker, creds credstore.CredentialStore) *services {
	candidateService := candidates.New(db, q)
	tokenStore := jobsdb.NewGoogleTokenStore(db)
	googleClient := igoogle.NewClient(
		os.Getenv("GOOGLE_CLIENT_ID"),
		os.Getenv("GOOGLE_CLIENT_SECRET"),
		os.Getenv("GOOGLE_REDIRECT_URL"),
		tokenStore,
	)

	return &services{
		auth:                authsvc.New(db),
		google:              googlesvc.New(googleClient),
		googleClient:        googleClient,
		ingest:              ingest.New(db, db),
		jobs:                jobs.New(db),
		applicationStatuses: applicationstatuses.New(db),
		applications:        applications.New(db),
		sources:             sources.New(),
		profile:             profile.New(db),
		scoringConfig:       scoringconfig.New(db, candidateService, db),
		aiPrefs:             aiprefs.New(creds),
		aiCredentials:       aicredentials.New(creds),
		sourceTargets:       sourcetargets.New(db, db, candidateService, q),
		companies:           companies.New(db, db, q),
		cvTemplates:         cvtemplates.NewService(googleClient, db),
	}
}

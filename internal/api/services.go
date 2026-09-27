package api

import (
	"os"

	"github.com/ollymarsters/job-scraper/internal/api/credstore"
	"github.com/ollymarsters/job-scraper/internal/api/extract"
	igoogle "github.com/ollymarsters/job-scraper/internal/api/google"
	"github.com/ollymarsters/job-scraper/internal/api/ingest"
	"github.com/ollymarsters/job-scraper/internal/api/services/aicredentials"
	"github.com/ollymarsters/job-scraper/internal/api/services/aiprefs"
	"github.com/ollymarsters/job-scraper/internal/api/services/applications"
	"github.com/ollymarsters/job-scraper/internal/api/services/applicationstatuses"
	authsvc "github.com/ollymarsters/job-scraper/internal/api/services/auth"
	"github.com/ollymarsters/job-scraper/internal/api/services/companies"
	"github.com/ollymarsters/job-scraper/internal/api/services/cvtemplates"
	googlesvc "github.com/ollymarsters/job-scraper/internal/api/services/google"
	"github.com/ollymarsters/job-scraper/internal/api/services/jobs"
	"github.com/ollymarsters/job-scraper/internal/api/services/profile"
	"github.com/ollymarsters/job-scraper/internal/api/services/scoringconfig"
	"github.com/ollymarsters/job-scraper/internal/api/services/sources"
	"github.com/ollymarsters/job-scraper/internal/api/services/sourcetargets"
	"github.com/ollymarsters/job-scraper/internal/api/services/suitability"
	"github.com/ollymarsters/job-scraper/internal/candidates"
	jobsdb "github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/queue"
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
	suitability         *suitability.Service
	aiPrefs             *aiprefs.Service
	aiCredentials       *aicredentials.Service
	sourceTargets       *sourcetargets.Service
	companies           *companies.Service
	cvTemplates         *cvtemplates.Service
}

func newServices(db *jobsdb.DB, q *queue.Broker, creds credstore.CredentialStore, suitabilitySvc *suitability.Service) *services {
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
		scoringConfig:       scoringconfig.New(db, candidateService, db, suitabilitySvc, extract.NewClient(), creds),
		suitability:         suitabilitySvc,
		aiPrefs:             aiprefs.New(creds),
		aiCredentials:       aicredentials.New(creds),
		sourceTargets:       sourcetargets.New(db, db, candidateService, q),
		companies:           companies.New(db, db, q),
		cvTemplates:         cvtemplates.NewService(googleClient, db),
	}
}

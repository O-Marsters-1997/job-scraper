package api

import (
	"os"

	"github.com/ollymarsters/job-scraper/internal/api/credstore"
	"github.com/ollymarsters/job-scraper/internal/api/extract"
	"github.com/ollymarsters/job-scraper/internal/api/services/aicredentials"
	"github.com/ollymarsters/job-scraper/internal/api/services/aiprefs"
	googlesvc "github.com/ollymarsters/job-scraper/internal/api/services/google"
	"github.com/ollymarsters/job-scraper/internal/api/services/profile"
	"github.com/ollymarsters/job-scraper/internal/api/services/scoringconfig"
	"github.com/ollymarsters/job-scraper/internal/api/services/suitability"
	jobsdb "github.com/ollymarsters/job-scraper/internal/data/db"
	igoogle "github.com/ollymarsters/job-scraper/internal/google"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
)

type services struct {
	google        *googlesvc.Service
	googleClient  *igoogle.Client
	profile       *profile.Service
	scoringConfig *scoringconfig.Service
	suitability   *suitability.Service
	aiPrefs       *aiprefs.Service
	aiCredentials *aicredentials.Service
}

func newServices(db *jobsdb.DB, q *queue.Broker, creds credstore.CredentialStore, suitabilitySvc *suitability.Service, js *jobsearch.Module) *services {
	tokenStore := jobsdb.NewGoogleTokenStore(db)
	googleClient := igoogle.NewClient(
		os.Getenv("GOOGLE_CLIENT_ID"),
		os.Getenv("GOOGLE_CLIENT_SECRET"),
		os.Getenv("GOOGLE_REDIRECT_URL"),
		tokenStore,
	)

	return &services{
		google:        googlesvc.New(googleClient),
		googleClient:  googleClient,
		profile:       profile.New(db),
		scoringConfig: scoringconfig.New(db, js, db, suitabilitySvc, extract.NewClient(), creds),
		suitability:   suitabilitySvc,
		aiPrefs:       aiprefs.New(creds),
		aiCredentials: aicredentials.New(creds),
	}
}

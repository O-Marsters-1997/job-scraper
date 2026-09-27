package api

import (
	"os"

	"github.com/ollymarsters/job-scraper/internal/api/credstore"
	"github.com/ollymarsters/job-scraper/internal/api/services/aicredentials"
	"github.com/ollymarsters/job-scraper/internal/api/services/aiprefs"
	googlesvc "github.com/ollymarsters/job-scraper/internal/api/services/google"
	"github.com/ollymarsters/job-scraper/internal/api/services/profile"
	jobsdb "github.com/ollymarsters/job-scraper/internal/data/db"
	igoogle "github.com/ollymarsters/job-scraper/internal/google"
)

type services struct {
	google        *googlesvc.Service
	googleClient  *igoogle.Client
	profile       *profile.Service
	aiPrefs       *aiprefs.Service
	aiCredentials *aicredentials.Service
}

func newServices(db *jobsdb.DB, creds credstore.CredentialStore) *services {
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
		aiPrefs:       aiprefs.New(creds),
		aiCredentials: aicredentials.New(creds),
	}
}

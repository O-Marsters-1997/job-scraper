package app

import (
	"context"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"github.com/ollymarsters/job-scraper/internal/applications"
	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/candidates"
	"github.com/ollymarsters/job-scraper/internal/credstore"
	"github.com/ollymarsters/job-scraper/internal/cvtemplates"
	jobsdb "github.com/ollymarsters/job-scraper/internal/data/db"
	igoogle "github.com/ollymarsters/job-scraper/internal/google"
	"github.com/ollymarsters/job-scraper/internal/handlers"
	"github.com/ollymarsters/job-scraper/internal/ingest"
	"github.com/ollymarsters/job-scraper/internal/queue"
)

func NewRouter(db *jobsdb.DB, q *queue.Broker, creds credstore.CredentialStore) http.Handler {
	allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:3000"
	}

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{allowedOrigin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	jobH := handlers.NewJobHandler(db)
	jobReasoningH := handlers.NewJobReasoningHandler(db, creds)
	authH := handlers.NewAuthHandler(db)
	appH := handlers.NewApplicationHandler(db)
	applicationsSvc := applications.New(db)
	statusH := handlers.NewApplicationStatusHandler(db)
	candidateService := candidates.New(db, q)
	stH := handlers.NewSourceTargetHandler(db, q).WithCandidates(candidateService, db)
	compH := handlers.NewCompaniesHandler(db, db, handlers.ATSBoardVerifier{})
	scoringCfgH := handlers.NewScoringConfigHandler(db).WithCandidates(candidateService)
	scoresH := handlers.NewScoresHandler(db)
	aiPrefsH := handlers.NewAIPrefsHandler(db, creds)
	aiCredsH := handlers.NewAICredentialsHandler(creds)
	profileH := handlers.NewProfileHandler(db)

	tokenStore := jobsdb.NewGoogleTokenStore(db)
	googleClient := igoogle.NewClient(
		os.Getenv("GOOGLE_CLIENT_ID"),
		os.Getenv("GOOGLE_CLIENT_SECRET"),
		os.Getenv("GOOGLE_REDIRECT_URL"),
		tokenStore,
	)
	googleH := handlers.NewGoogleHandler(googleClient)

	cvSvc := cvtemplates.NewService(googleClient, db)
	cvH := handlers.NewCVTemplatesHandler(cvSvc, googleClient)

	ingestH := handlers.NewIngestHandler(ingest.New(db, db))

	r.Route("/auth", func(r chi.Router) {
		r.Post("/login", authH.Login)
		r.Post("/signup", authH.Signup)
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(db))
			r.Post("/logout", authH.Logout)
			r.Get("/me", authH.Me)
		})
	})

	r.Route("/google", func(r chi.Router) {
		// /start is public so the OAuth redirect URL stays clean.
		r.Get("/oauth/start", googleH.OAuthStart)
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(db))
			r.Get("/oauth/callback", googleH.OAuthCallback)
			r.Get("/status", googleH.GetStatus)
			r.Delete("/link", googleH.Disconnect)
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(db))

		r.Get("/jobs", jobH.ListJobs)
		r.Get("/jobs/all", jobH.ListAllJobs)
		r.Get("/jobs/{id}", jobH.GetJob)
		r.Post("/jobs/{id}/reasoning", jobReasoningH.PostJobReasoning)

		r.Route("/application-statuses", func(r chi.Router) {
			r.Get("/", statusH.ListApplicationStatuses)
			r.Post("/", statusH.CreateApplicationStatus)
			r.Patch("/{id}", statusH.UpdateApplicationStatus)
			r.Delete("/{id}", statusH.DeleteApplicationStatus)
		})

		r.Route("/applications", func(r chi.Router) {
			r.Get("/", appH.ListApplications)
			r.Post("/", handlers.Body(applicationsSvc.Create, http.StatusCreated))
			r.Patch("/{id}", handlers.BodyID(applicationsSvc.Update, http.StatusOK))
			r.Delete("/{id}", handlers.ID(func(ctx context.Context, userID, id string) (struct{}, error) {
				return struct{}{}, db.DeleteApplication(ctx, userID, id)
			}, http.StatusNoContent))
			r.Get("/for-jobs", appH.GetApplicationsForJobs)
		})

		r.Get("/sources", stH.Sources)
		r.Get("/sources/resolve", stH.ResolveBoard)

		r.Get("/profile", profileH.GetProfile)
		r.Put("/profile", profileH.UpdateProfile)

		r.Get("/scoring-config", scoringCfgH.GetScoringConfig)
		r.Put("/scoring-config", scoringCfgH.UpdateScoringConfig)
		r.Get("/scores/status", scoresH.Status)
		r.Post("/scores/rescore", scoresH.Rescore)

		r.Get("/ai-prefs", aiPrefsH.GetAIPrefs)
		r.Put("/ai-prefs", aiPrefsH.UpdateAIPrefs)

		r.Put("/ai-credentials", aiCredsH.UpsertCredential)

		r.Route("/source-targets", func(r chi.Router) {
			r.Get("/", stH.List)
			r.Post("/", stH.Create)
			r.Patch("/{id}", stH.Update)
			r.Post("/{id}/scrape", stH.Scrape)
			r.Delete("/{id}", stH.Delete)
		})

		r.Route("/companies", func(r chi.Router) {
			r.Get("/", compH.List)
			r.Post("/", compH.Create)
			r.Put("/{id}/tracking", compH.SetTracking)
			r.Get("/{id}/boards", compH.ListBoards)
			r.Post("/{id}/boards", compH.AddBoard)
		})

		r.Route("/cv-templates", func(r chi.Router) {
			r.Get("/", cvH.ListCVTemplates)
			r.Get("/{docId}/{tabId}/pdf", cvH.ExportCV)
		})

		r.Route("/tracked-docs", func(r chi.Router) {
			r.Post("/", cvH.AddTrackedDoc)
			r.Delete("/{docId}", cvH.RemoveTrackedDoc)
			r.Post("/{docId}/tabs/{tabId}/hide", cvH.HideTab)
			r.Post("/{docId}/tabs/{tabId}/show", cvH.ShowTab)
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.ServiceTokenMiddleware)
		r.Post("/ingest", ingestH.Ingest)
		r.Post("/ingest/batch", ingestH.IngestBatch)
	})

	return r
}

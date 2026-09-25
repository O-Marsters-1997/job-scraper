package app

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/credstore"
	jobsdb "github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/handlers"
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

	svc := newServices(db, q, creds)

	r.Route("/auth", func(r chi.Router) {
		r.Post("/login", handlers.Login(svc.auth))
		r.Post("/signup", handlers.Signup(svc.auth))
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(db))
			r.Post("/logout", handlers.Logout(svc.auth))
			r.Get("/me", handlers.Me)
		})
	})

	r.Route("/google", func(r chi.Router) {
		r.Get("/oauth/start", handlers.OAuthStart(svc.google))
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(db))
			r.Get("/oauth/callback", handlers.OAuthCallback(svc.google))
			r.Get("/status", handlers.GetAll(svc.google.Status))
			r.Delete("/link", handlers.Delete(svc.google.Disconnect))
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(db))

		r.Get("/jobs", handlers.Query(svc.jobs.List))
		r.Get("/jobs/all", handlers.GetAll(db.List))
		r.Get("/jobs/{id}", handlers.GetByID(svc.jobs.Get))

		r.Route("/application-statuses", func(r chi.Router) {
			r.Get("/", handlers.GetAll(db.ListApplicationStatusesByUser))
			r.Post("/", handlers.Create(svc.applicationStatuses.Create))
			r.Patch("/{id}", handlers.Update(svc.applicationStatuses.Update))
			r.Delete("/{id}", handlers.Delete(svc.applicationStatuses.Delete))
		})

		r.Route("/applications", func(r chi.Router) {
			r.Get("/", handlers.Query(svc.applications.List))
			r.Post("/", handlers.Create(svc.applications.Create))
			r.Patch("/{id}", handlers.Update(svc.applications.Update))
			r.Delete("/{id}", handlers.Delete(svc.applications.Delete))
			r.Get("/for-jobs", handlers.Query(svc.applications.ForJobs))
		})

		r.Get("/sources", handlers.GetAll(svc.sources.List))
		r.Get("/sources/resolve", handlers.Query(svc.sources.Resolve))

		r.Get("/profile", handlers.GetAll(svc.profile.Get))
		r.Put("/profile", handlers.Update(svc.profile.Update))

		r.Get("/scoring-config", handlers.GetAll(svc.scoringConfig.Get))
		r.Put("/scoring-config", handlers.Update(svc.scoringConfig.Update))
		r.Get("/scores/status", handlers.GetAll(db.GetScoringStatus))
		r.Post("/scores/rescore", handlers.GetAll(svc.scoringConfig.Rescore))

		r.Get("/ai-prefs", handlers.GetAll(svc.aiPrefs.Get))

		r.Put("/ai-credentials", handlers.Update(svc.aiCredentials.Update))

		r.Route("/source-targets", func(r chi.Router) {
			r.Get("/", handlers.GetAll(db.ListSourceTargetsByUser))
			r.Post("/", handlers.Create(svc.sourceTargets.Create))
			r.Patch("/{id}", handlers.Update(svc.sourceTargets.Update))
			r.Post("/{id}/scrape", handlers.GetByID(svc.sourceTargets.Scrape))
			r.Delete("/{id}", handlers.Delete(svc.sourceTargets.Delete))
		})

		r.Route("/companies", func(r chi.Router) {
			r.Get("/", handlers.GetAll(db.ListCompaniesForUser))
			r.Post("/", handlers.Create(svc.companies.Create))
			r.Put("/{id}/tracking", handlers.Update(svc.companies.SetTracking))
			r.Get("/{id}/boards", handlers.GetByID(svc.companies.ListBoards))
			r.Post("/{id}/boards", handlers.Create(svc.companies.AddBoard))
		})

		r.Route("/cv-templates", func(r chi.Router) {
			r.Get("/", handlers.GetAll(svc.cvTemplates.List))
			r.Get("/{docId}/{tabId}/pdf", handlers.ExportCV(svc.cvTemplates))
		})

		r.Route("/tracked-docs", func(r chi.Router) {
			r.Post("/", handlers.Create(svc.cvTemplates.AddDoc))
			r.Delete("/{id}", handlers.Delete(svc.cvTemplates.RemoveDoc))
			r.Post("/{docId}/tabs/{tabId}/hide", handlers.Update(svc.cvTemplates.HideTab))
			r.Post("/{docId}/tabs/{tabId}/show", handlers.Update(svc.cvTemplates.ShowTab))
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.ServiceTokenMiddleware)
		r.Post("/ingest", handlers.Ingest(svc.ingest))
		r.With(middleware.RequestSize(2<<20)).Post("/ingest/batch", handlers.IngestBatch(svc.ingest))
	})

	return r
}

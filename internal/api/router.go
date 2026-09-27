package api

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/ollymarsters/job-scraper/internal/api/apihandlers"
	"github.com/ollymarsters/job-scraper/internal/api/auth"
	"github.com/ollymarsters/job-scraper/internal/api/credstore"
	"github.com/ollymarsters/job-scraper/internal/api/services/suitability"
	jobsdb "github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/handlers"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/services/applications"
	"github.com/ollymarsters/job-scraper/internal/telemetry"
)

// Module mounts a context's session-protected routes onto the router.
type Module interface {
	Routes(chi.Router)
}

// NewRouter takes apps separately from modules because legacy signup also
// wires it in as a StatusSeeder (ADR 0011); every other moved context can
// go through modules alone.
func NewRouter(db *jobsdb.DB, q *queue.Broker, creds credstore.CredentialStore, suitabilitySvc *suitability.Service, apps *applications.Module, modules ...Module) http.Handler {
	allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:3000"
	}

	r := chi.NewRouter()
	r.Use(telemetry.AccessLog)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{allowedOrigin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	svc := newServices(db, q, creds, suitabilitySvc, apps)

	allModules := append([]Module{apps}, modules...)
	for _, m := range allModules {
		if pm, ok := m.(interface{ PublicRoutes(chi.Router) }); ok {
			pm.PublicRoutes(r)
		}
	}

	r.Route("/auth", func(r chi.Router) {
		r.Post("/login", apihandlers.Login(svc.auth))
		r.Post("/signup", apihandlers.Signup(svc.auth))
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(db))
			r.Post("/logout", apihandlers.Logout(svc.auth))
			r.Get("/me", apihandlers.Me)
		})
	})

	r.Route("/google", func(r chi.Router) {
		r.Get("/oauth/start", apihandlers.OAuthStart(svc.google))
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(db))
			r.Get("/oauth/callback", apihandlers.OAuthCallback(svc.google))
			r.Get("/status", handlers.GetAll(svc.google.Status))
			r.Delete("/link", handlers.Delete(svc.google.Disconnect))
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(db))

		r.Get("/jobs", handlers.Query(svc.jobs.List))
		r.Get("/jobs/all", handlers.GetAll(db.List))
		r.Get("/jobs/{id}", handlers.GetByID(svc.jobs.Get))

		r.Get("/sources", handlers.GetAll(svc.sources.List))
		r.Get("/sources/resolve", handlers.Query(svc.sources.Resolve))

		r.Get("/profile", handlers.GetAll(svc.profile.Get))
		r.Put("/profile", handlers.Update(svc.profile.Update))

		r.Get("/scoring-config", handlers.GetAll(svc.scoringConfig.Get))
		r.Put("/scoring-config", handlers.Update(svc.scoringConfig.Update))
		r.Get("/scoring-options", handlers.GetAll(svc.scoringConfig.Options))
		r.Get("/scores/status", handlers.GetAll(db.GetScoringStatus))
		r.Post("/scores/recompute", handlers.GetAll(svc.suitability.Recompute))

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
			r.Get("/{docId}/{tabId}/pdf", apihandlers.ExportCV(svc.cvTemplates))
		})

		r.Route("/tracked-docs", func(r chi.Router) {
			r.Post("/", handlers.Create(svc.cvTemplates.AddDoc))
			r.Delete("/{id}", handlers.Delete(svc.cvTemplates.RemoveDoc))
			r.Post("/{docId}/tabs/{tabId}/hide", handlers.Update(svc.cvTemplates.HideTab))
			r.Post("/{docId}/tabs/{tabId}/show", handlers.Update(svc.cvTemplates.ShowTab))
		})

		for _, m := range allModules {
			m.Routes(r)
		}
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.ServiceTokenMiddleware)
		r.Post("/ingest", apihandlers.Ingest(svc.ingest))
		r.With(middleware.RequestSize(2<<20)).Post("/ingest/batch", apihandlers.IngestBatch(svc.ingest))
	})

	return r
}

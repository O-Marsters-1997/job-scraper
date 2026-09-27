package api

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"github.com/ollymarsters/job-scraper/internal/api/auth"
	"github.com/ollymarsters/job-scraper/internal/services/identity"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/telemetry"
)

// Module mounts a context's session-protected routes onto the router.
type Module interface {
	Routes(chi.Router)
}

// NewRouter takes idm and js separately from modules: idm's session
// middleware wraps every protected route, and js's PublicRoutes (ingest)
// needs its own ServiceTokenMiddleware group here, since depguard's "shared"
// rule keeps internal/api out of internal/services/**, so jobsearch itself
// can't apply that middleware (ADR 0011). Every other moved context goes
// through modules alone.
func NewRouter(idm *identity.Module, js *jobsearch.Module, modules ...Module) http.Handler {
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

	allModules := append([]Module{idm}, modules...)
	for _, m := range allModules {
		if pm, ok := m.(interface{ PublicRoutes(chi.Router) }); ok {
			pm.PublicRoutes(r)
		}
	}

	r.Group(func(r chi.Router) {
		r.Use(auth.ServiceTokenMiddleware)
		js.PublicRoutes(r)
	})

	r.Group(func(r chi.Router) {
		r.Use(idm.Middleware())

		js.Routes(r)
		for _, m := range allModules {
			m.Routes(r)
		}
	})

	return r
}

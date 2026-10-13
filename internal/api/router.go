package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/ollymarsters/job-scraper/internal/api/auth"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/identity"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/telemetry"
)

// Module mounts a context's session-protected routes onto the router.
type Module interface {
	Routes(chi.Router)
}

// NewRouter takes idm and js separately from modules so js's PublicRoutes
// (ingest) can get its own ServiceTokenMiddleware group here, since
// depguard bars internal/api from internal/services/** (ADR 0011).
func NewRouter(idm *identity.Module, js *jobsearch.Module, modules ...Module) (http.Handler, error) {
	allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:3000"
	}

	cop := http.NewCrossOriginProtection()
	if err := cop.AddTrustedOrigin(allowedOrigin); err != nil {
		return nil, fmt.Errorf("add trusted origin %q: %w", allowedOrigin, err)
	}

	r := chi.NewRouter()
	r.Use(otelhttp.NewMiddleware("api"))
	r.Use(chimw.RequestID)
	r.Use(logger.Middleware)
	r.Use(telemetry.AccessLog)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{allowedOrigin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))
	r.Use(requireJSON)

	allModules := append([]Module{idm}, modules...)
	r.Group(func(r chi.Router) {
		r.Use(limitBody(maxBodyBytes))
		for _, m := range allModules {
			if pm, ok := m.(interface{ PublicRoutes(chi.Router) }); ok {
				pm.PublicRoutes(r)
			}
		}
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.ServiceTokenMiddleware)
		js.PublicRoutes(r)
	})

	r.Group(func(r chi.Router) {
		r.Use(limitBody(maxBodyBytes))
		r.Use(idm.Middleware())

		js.Routes(r)
		for _, m := range allModules {
			m.Routes(r)
		}
	})

	return cop.Handler(r), nil
}

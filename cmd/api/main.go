package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"github.com/ollymarsters/job-scraper/internal/auth"
	jobsdb "github.com/ollymarsters/job-scraper/internal/data/db"
	igoogle "github.com/ollymarsters/job-scraper/internal/google"
	"github.com/ollymarsters/job-scraper/internal/handlers"
	"github.com/ollymarsters/job-scraper/internal/logger"
)

func main() {
	slog.SetDefault(logger.New())

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	connStr, err := jobsdb.ConnString()
	if err != nil {
		slog.Error("db config invalid", slog.Any("err", err))
		os.Exit(1)
	}
	db, err := jobsdb.New(ctx, connStr)
	if err != nil {
		slog.Error("db init failed",
			slog.Any("err", err),
		)
		os.Exit(1)
	}
	defer db.Close()

	allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:3000"
	}

	port := os.Getenv("API_PORT")
	if port == "" {
		port = ":8080"
	}

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{allowedOrigin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	jobH := handlers.NewJobHandler(db)
	authH := handlers.NewAuthHandler(db, db, db)
	appH := handlers.NewApplicationHandler(db)
	statusH := handlers.NewApplicationStatusHandler(db)

	tokenStore := jobsdb.NewGoogleTokenStore(db)
	googleClient := igoogle.NewClient(
		os.Getenv("GOOGLE_CLIENT_ID"),
		os.Getenv("GOOGLE_CLIENT_SECRET"),
		os.Getenv("GOOGLE_REDIRECT_URL"),
		tokenStore,
	)
	googleH := handlers.NewGoogleHandler(googleClient, db)

	r.Post("/auth/login", authH.Login)
	r.Post("/auth/signup", authH.Signup)

	// Google OAuth — start is accessible without auth so the redirect URL is clean,
	// but callback and status/disconnect require the session cookie.
	r.Get("/google/oauth/start", googleH.OAuthStart)

	// Protected routes — auth middleware applied to all.
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(db))
		r.Get("/jobs", jobH.ListJobs)
		r.Post("/auth/logout", authH.Logout)
		r.Get("/auth/me", authH.Me)

		r.Get("/application-statuses", statusH.ListApplicationStatuses)
		r.Post("/application-statuses", statusH.CreateApplicationStatus)
		r.Patch("/application-statuses/{id}", statusH.UpdateApplicationStatus)
		r.Delete("/application-statuses/{id}", statusH.DeleteApplicationStatus)

		r.Get("/applications", appH.ListApplications)
		r.Post("/applications", appH.CreateApplication)
		r.Patch("/applications/{id}", appH.UpdateApplication)
		r.Delete("/applications/{id}", appH.DeleteApplication)
		r.Get("/applications/for-jobs", appH.GetApplicationsForJobs)

		r.Get("/google/oauth/callback", googleH.OAuthCallback)
		r.Get("/google/status", googleH.GetStatus)
		r.Delete("/google/link", googleH.Disconnect)
	})

	srv := &http.Server{Addr: port, Handler: r}

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	slog.Info("api server starting", slog.String("addr", port))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", slog.Any("err", err))
		os.Exit(1)
	}
}

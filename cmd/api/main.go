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
	"github.com/ollymarsters/job-scraper/internal/handlers"
	"github.com/ollymarsters/job-scraper/internal/logger"
)

func main() {
	slog.SetDefault(logger.New())

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	db, err := jobsdb.New(ctx, jobsdb.ConnString())
	if err != nil {
		slog.Error("db init failed", slog.Any("err", err))
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

	h := handlers.New(db, db, db, db, db)

	r.Post("/auth/login", h.Login)
	r.Post("/auth/signup", h.Signup)

	// Protected routes — auth middleware applied to all.
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(db))
		r.Get("/jobs", h.ListJobs)
		r.Post("/auth/logout", h.Logout)
		r.Get("/auth/me", h.Me)

		r.Get("/application-statuses", h.ListApplicationStatuses)
		r.Post("/application-statuses", h.CreateApplicationStatus)
		r.Patch("/application-statuses/{id}", h.UpdateApplicationStatus)
		r.Delete("/application-statuses/{id}", h.DeleteApplicationStatus)

		r.Get("/applications", h.ListApplications)
		r.Post("/applications", h.CreateApplication)
		r.Patch("/applications/{id}", h.UpdateApplication)
		r.Delete("/applications/{id}", h.DeleteApplication)
		r.Get("/applications/for-jobs", h.GetApplicationsForJobs)
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

package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/cvtemplates"
	jobsdb "github.com/ollymarsters/job-scraper/internal/data/db"
	igoogle "github.com/ollymarsters/job-scraper/internal/google"
	"github.com/ollymarsters/job-scraper/internal/handlers"
	"github.com/ollymarsters/job-scraper/internal/ingest"
	"github.com/ollymarsters/job-scraper/internal/notify"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/score"
)

func NewRouter(ctx context.Context, db *jobsdb.DB, q *queue.Queue) http.Handler {
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
	authH := handlers.NewAuthHandler(db)
	appH := handlers.NewApplicationHandler(db)
	statusH := handlers.NewApplicationStatusHandler(db)
	stH := handlers.NewSourceTargetHandler(db, q)

	tokenStore := jobsdb.NewGoogleTokenStore(db)
	googleClient := igoogle.NewClient(
		os.Getenv("GOOGLE_CLIENT_ID"),
		os.Getenv("GOOGLE_CLIENT_SECRET"),
		os.Getenv("GOOGLE_REDIRECT_URL"),
		tokenStore,
	)
	googleH := handlers.NewGoogleHandler(googleClient, db)

	cvSvc := cvtemplates.NewService(googleClient, db)
	cvH := handlers.NewCVTemplatesHandler(cvSvc, googleClient)

	scoringUserID := os.Getenv("SCORING_USER_ID")
	ingestSvc := buildIngestSvc(ctx, db, scoringUserID)
	ingestH := handlers.NewIngestHandler(ingestSvc)

	r.Post("/auth/login", authH.Login)
	r.Post("/auth/signup", authH.Signup)

	// Google OAuth start is accessible without auth so the redirect URL is clean.
	r.Get("/google/oauth/start", googleH.OAuthStart)

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

		r.Get("/source-targets", stH.List)
		r.Post("/source-targets", stH.Create)
		r.Patch("/source-targets/{id}", stH.Update)
		r.Delete("/source-targets/{id}", stH.Delete)

		r.Get("/cv-templates", cvH.ListCVTemplates)
		r.Post("/tracked-docs", cvH.AddTrackedDoc)
		r.Delete("/tracked-docs/{docId}", cvH.RemoveTrackedDoc)
		r.Post("/tracked-docs/{docId}/tabs/{tabId}/hide", cvH.HideTab)
		r.Post("/tracked-docs/{docId}/tabs/{tabId}/show", cvH.ShowTab)
		r.Get("/cv-templates/{docId}/{tabId}/pdf", cvH.ExportCV)
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.ServiceTokenMiddleware)
		r.Post("/ingest", ingestH.Ingest)
	})

	return r
}

func buildIngestSvc(ctx context.Context, db *jobsdb.DB, scoringUserID string) *ingest.Ingester {
	var scorer ingest.Scorer
	if scoringUserID != "" {
		if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
			claudeScorer := score.NewClaudeScorer(score.ClaudeScorerConfig{APIKey: apiKey})
			scorer = score.NewIngestScorer(claudeScorer, db, db, scoringUserID)
		}
	}
	var notifier ingest.Notifier
	if notifSvc := setupNotifications(ctx, db, scoringUserID); notifSvc != nil {
		notifier = notifSvc
	}
	return ingest.New(db, scorer, notifier)
}

func setupNotifications(ctx context.Context, db *jobsdb.DB, userID string) *notify.NotificationService {
	apiKey := os.Getenv("RESEND_API_KEY")
	to := os.Getenv("NOTIFY_EMAIL_TO")
	from := os.Getenv("NOTIFY_EMAIL_FROM")
	if from == "" {
		from = "onboarding@resend.dev"
	}

	if apiKey == "" || to == "" {
		slog.Info("notifications disabled: RESEND_API_KEY or NOTIFY_EMAIL_TO not set")
		return nil
	}

	renderer, err := notify.NewRenderer()
	if err != nil {
		slog.Error("notify: failed to load templates", slog.Any("err", err))
		return nil
	}

	var threshold int
	if userID != "" {
		if cfg, err := db.GetSearchConfig(ctx, userID); err == nil {
			threshold = cfg.NotifyThreshold
		} else {
			slog.Warn("notify: could not load search config for threshold", slog.Any("err", err))
		}
	}

	return notify.NewNotificationService(
		notify.NewResendNotifier(apiKey, from),
		renderer,
		notify.Config{
			To:              to,
			OnIngestEnabled: os.Getenv("NOTIFY_ON_INGEST") == "true",
			DigestEnabled:   os.Getenv("NOTIFY_DIGEST_ENABLED") != "false",
			NotifyThreshold: threshold,
		},
	)
}

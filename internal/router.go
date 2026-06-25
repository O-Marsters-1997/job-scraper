package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/credstore"
	"github.com/ollymarsters/job-scraper/internal/cvtemplates"
	jobsdb "github.com/ollymarsters/job-scraper/internal/data/db"
	igoogle "github.com/ollymarsters/job-scraper/internal/google"
	"github.com/ollymarsters/job-scraper/internal/handlers"
	"github.com/ollymarsters/job-scraper/internal/ingest"
	"github.com/ollymarsters/job-scraper/internal/notify"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/score"
)

func NewRouter(ctx context.Context, db *jobsdb.DB, q *queue.Queue, creds credstore.CredentialStore) http.Handler {
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
	scoringCfgH := handlers.NewScoringConfigHandler(db)
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
	googleH := handlers.NewGoogleHandler(googleClient, db)

	cvSvc := cvtemplates.NewService(googleClient, db)
	cvH := handlers.NewCVTemplatesHandler(cvSvc, googleClient)

	scoringUserID := os.Getenv("SCORING_USER_ID")
	ingestSvc := buildIngestSvc(ctx, db, scoringUserID)
	ingestH := handlers.NewIngestHandler(ingestSvc)

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

		r.Route("/application-statuses", func(r chi.Router) {
			r.Get("/", statusH.ListApplicationStatuses)
			r.Post("/", statusH.CreateApplicationStatus)
			r.Patch("/{id}", statusH.UpdateApplicationStatus)
			r.Delete("/{id}", statusH.DeleteApplicationStatus)
		})

		r.Route("/applications", func(r chi.Router) {
			r.Get("/", appH.ListApplications)
			r.Post("/", appH.CreateApplication)
			r.Patch("/{id}", appH.UpdateApplication)
			r.Delete("/{id}", appH.DeleteApplication)
			r.Get("/for-jobs", appH.GetApplicationsForJobs)
		})

		r.Get("/sources", stH.Sources)

		r.Get("/profile", profileH.Get)
		r.Put("/profile", profileH.Put)

		r.Get("/scoring-config", scoringCfgH.Get)
		r.Put("/scoring-config", scoringCfgH.Put)

		r.Get("/ai-prefs", aiPrefsH.Get)
		r.Put("/ai-prefs", aiPrefsH.Put)

		r.Put("/ai-credentials", aiCredsH.Put)

		r.Route("/source-targets", func(r chi.Router) {
			r.Get("/", stH.List)
			r.Post("/", stH.Create)
			r.Patch("/{id}", stH.Update)
			r.Delete("/{id}", stH.Delete)
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
	})

	return r
}

func buildIngestSvc(ctx context.Context, db *jobsdb.DB, scoringUserID string) *ingest.Ingester {
	var scorer ingest.Scorer
	if scoringUserID != "" {
		if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
			claudeScorer := score.NewClaudeScorer(score.ClaudeScorerConfig{APIKey: apiKey})
			scorer = score.NewIngestScorerWithPrefs(claudeScorer, db, db, db, scoringUserID)
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

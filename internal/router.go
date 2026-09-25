package app

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"github.com/ollymarsters/job-scraper/internal/aiprefs"
	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/applications"
	"github.com/ollymarsters/job-scraper/internal/auth"
	"github.com/ollymarsters/job-scraper/internal/candidates"
	"github.com/ollymarsters/job-scraper/internal/companies"
	"github.com/ollymarsters/job-scraper/internal/credstore"
	"github.com/ollymarsters/job-scraper/internal/cvtemplates"
	jobsdb "github.com/ollymarsters/job-scraper/internal/data/db"
	"github.com/ollymarsters/job-scraper/internal/dto"
	igoogle "github.com/ollymarsters/job-scraper/internal/google"
	"github.com/ollymarsters/job-scraper/internal/handlers"
	"github.com/ollymarsters/job-scraper/internal/ingest"
	"github.com/ollymarsters/job-scraper/internal/jobreasoning"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/score"
	"github.com/ollymarsters/job-scraper/internal/scoringconfig"
	"github.com/ollymarsters/job-scraper/internal/sourcetargets"
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
	authH := handlers.NewAuthHandler(db)
	appH := handlers.NewApplicationHandler(db)
	applicationsSvc := applications.New(db)
	statusH := handlers.NewApplicationStatusHandler(db)
	candidateService := candidates.New(db, q)
	sourceTargetsSvc := sourcetargets.New(db, db, candidateService, q)
	companiesSvc := companies.New(db, db, handlers.ATSBoardVerifier{})
	scoringConfigSvc := scoringconfig.New(db, candidateService)
	aiPrefsSvc := aiprefs.New(db, creds)
	jobReasoningSvc := jobreasoning.New(db, db, db, db, creds, func(apiKey string) score.SuitabilityScorer {
		return score.NewClaudeScorer(apiKey)
	})

	tokenStore := jobsdb.NewGoogleTokenStore(db)
	googleClient := igoogle.NewClient(
		os.Getenv("GOOGLE_CLIENT_ID"),
		os.Getenv("GOOGLE_CLIENT_SECRET"),
		os.Getenv("GOOGLE_REDIRECT_URL"),
		tokenStore,
	)
	googleH := handlers.NewGoogleHandler(googleClient)

	cvSvc := cvtemplates.NewService(googleClient, db)
	cvH := handlers.NewCVTemplatesHandler(googleClient)

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
		r.Post("/jobs/{id}/reasoning", handlers.ID(jobReasoningSvc.Generate, http.StatusOK))

		r.Route("/application-statuses", func(r chi.Router) {
			r.Get("/", handlers.User(db.ListApplicationStatusesByUser, http.StatusOK))
			r.Post("/", handlers.Body(func(ctx context.Context, userID string, in dto.ApplicationStatusInput) (dto.ApplicationStatus, error) {
				if in.Name == "" || in.Colour == "" {
					return dto.ApplicationStatus{}, apperr.Invalid("name and colour are required")
				}
				return db.CreateApplicationStatus(ctx, userID, in.Name, in.Colour)
			}, http.StatusCreated))
			r.Patch("/{id}", handlers.BodyID(func(ctx context.Context, userID, id string, in dto.ApplicationStatusInput) (dto.ApplicationStatus, error) {
				if in.Name == "" || in.Colour == "" {
					return dto.ApplicationStatus{}, apperr.Invalid("name and colour are required")
				}
				return db.UpdateApplicationStatus(ctx, id, userID, in.Name, in.Colour)
			}, http.StatusOK))
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

		r.Get("/sources", handlers.Sources)
		r.Get("/sources/resolve", handlers.ResolveBoard)

		r.Get("/profile", handlers.User(func(ctx context.Context, userID string) (dto.ProfileView, error) {
			p, err := db.GetProfile(ctx, userID)
			return dto.ProfileView(p), err
		}, http.StatusOK))
		r.Put("/profile", handlers.Body(func(ctx context.Context, userID string, in dto.UpdateProfileInput) (struct{}, error) {
			_, err := db.UpdateEmail(ctx, userID, in.Email)
			return struct{}{}, err
		}, http.StatusNoContent))

		r.Get("/scoring-config", handlers.User(scoringConfigSvc.Get, http.StatusOK))
		r.Put("/scoring-config", handlers.Body(scoringConfigSvc.Update, http.StatusOK))
		r.Get("/scores/status", handlers.User(db.GetScoringStatus, http.StatusOK))
		r.Post("/scores/rescore", handlers.User(func(ctx context.Context, userID string) (dto.RescoreResult, error) {
			queued, err := db.QueueRescore(ctx, userID)
			return dto.RescoreResult{Queued: queued}, err
		}, http.StatusOK))

		r.Get("/ai-prefs", handlers.User(aiPrefsSvc.Get, http.StatusOK))
		r.Put("/ai-prefs", handlers.Body(aiPrefsSvc.Update, http.StatusOK))

		r.Put("/ai-credentials", handlers.Body(func(ctx context.Context, userID string, in dto.UpsertCredentialInput) (struct{}, error) {
			if in.Provider == "" {
				return struct{}{}, apperr.Invalid("provider required")
			}
			if in.APIKey == nil {
				return struct{}{}, creds.Delete(ctx, userID, in.Provider)
			}
			return struct{}{}, creds.Save(ctx, userID, in.Provider, strings.Trim(*in.APIKey, `"`))
		}, http.StatusNoContent))

		r.Route("/source-targets", func(r chi.Router) {
			r.Get("/", handlers.User(db.ListSourceTargetsByUser, http.StatusOK))
			r.Post("/", handlers.Body(sourceTargetsSvc.Create, http.StatusCreated))
			r.Patch("/{id}", handlers.BodyID(sourceTargetsSvc.Update, http.StatusOK))
			r.Post("/{id}/scrape", handlers.ID(sourceTargetsSvc.Scrape, http.StatusAccepted))
			r.Delete("/{id}", handlers.ID(func(ctx context.Context, userID, id string) (struct{}, error) {
				return struct{}{}, db.DeleteSourceTarget(ctx, id, userID)
			}, http.StatusNoContent))
		})

		r.Route("/companies", func(r chi.Router) {
			r.Get("/", handlers.User(db.ListCompaniesForUser, http.StatusOK))
			r.Post("/", handlers.Body(companiesSvc.Create, http.StatusCreated))
			r.Put("/{id}/tracking", handlers.BodyID(companiesSvc.SetTracking, http.StatusOK))
			r.Get("/{id}/boards", handlers.ID(companiesSvc.ListBoards, http.StatusOK))
			r.Post("/{id}/boards", handlers.BodyID(companiesSvc.AddBoard, http.StatusOK))
		})

		r.Route("/cv-templates", func(r chi.Router) {
			r.Get("/", handlers.User(cvSvc.List, http.StatusOK))
			r.Get("/{docId}/{tabId}/pdf", cvH.ExportCV)
		})

		r.Route("/tracked-docs", func(r chi.Router) {
			r.Post("/", handlers.Body(func(ctx context.Context, userID string, in struct {
				URL string `json:"url"`
			}) (struct{}, error) {
				return struct{}{}, cvSvc.AddDoc(ctx, userID, in.URL)
			}, http.StatusCreated))
			r.Delete("/{id}", handlers.ID(func(ctx context.Context, userID, id string) (struct{}, error) {
				return struct{}{}, cvSvc.RemoveDoc(ctx, userID, id)
			}, http.StatusNoContent))
			r.Post("/{docId}/tabs/{tabId}/hide", handlers.ID2(func(ctx context.Context, userID, docID, tabID string) (struct{}, error) {
				return struct{}{}, cvSvc.HideTab(ctx, userID, docID, tabID)
			}, http.StatusNoContent))
			r.Post("/{docId}/tabs/{tabId}/show", handlers.ID2(func(ctx context.Context, userID, docID, tabID string) (struct{}, error) {
				return struct{}{}, cvSvc.ShowTab(ctx, userID, docID, tabID)
			}, http.StatusNoContent))
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.ServiceTokenMiddleware)
		r.Post("/ingest", ingestH.Ingest)
		r.Post("/ingest/batch", ingestH.IngestBatch)
	})

	return r
}

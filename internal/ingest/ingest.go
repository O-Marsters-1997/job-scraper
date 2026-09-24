package ingest

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"strings"

	"github.com/google/uuid"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sources/registry"
)

type Saver interface {
	Save(ctx context.Context, jobs []dto.Job) ([]dto.Job, error)
}

type CanonicalSaver interface {
	SaveCanonical(ctx context.Context, job dto.Job) (dto.Job, string, error)
}

type Result struct {
	Status string `json:"status"`
	JobID  string `json:"job_id,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// Scorer runs suitability scoring for a batch of jobs for a specific user after they are saved.
// Implementations handle their own errors internally and never block ingest.
type Scorer interface {
	ScoreAndSaveBatch(ctx context.Context, jobs []dto.Job, userID string)
}

// TargetUserLister lists users with an enabled source_target matching (source,
// value). Pass value="" to match any enabled target for that source (used for
// discovery-sourced jobs, where the job doesn't carry the specific target).
type TargetUserLister interface {
	ListUserIDsForTarget(ctx context.Context, source, value string) ([]string, error)
}

// CompanyUpserter records a company seen during ingest into the shared catalog.
type CompanyUpserter interface {
	UpsertCompany(ctx context.Context, c dto.CompanyUpsert) (dto.Company, error)
}

type CredentialGetter interface {
	Get(ctx context.Context, userID, provider string) (string, error)
}

// Config wires all Ingester dependencies. Users, Creds, and ScorerFor are all
// required together; omitting any one disables per-user suitability scoring.
// Companies is optional; omitting it disables the company catalog upsert.
type Config struct {
	DB        Saver
	Provider  string // AI provider to fan out to, e.g. "anthropic"
	Users     TargetUserLister
	Creds     CredentialGetter
	ScorerFor func(apiKey string) Scorer
	Companies CompanyUpserter
}

// Ingester saves valid jobs and fans out scoring to interested users.
type Ingester struct {
	db        Saver
	provider  string
	users     TargetUserLister
	creds     CredentialGetter
	scorerFor func(apiKey string) Scorer
	companies CompanyUpserter
}

// New creates an Ingester. Scoring is skipped when Config.Users, Config.Creds,
// or Config.ScorerFor is nil. Companies may also be nil.
func New(cfg Config) *Ingester {
	return &Ingester{
		db:        cfg.DB,
		provider:  cfg.Provider,
		users:     cfg.Users,
		creds:     cfg.Creds,
		scorerFor: cfg.ScorerFor,
		companies: cfg.Companies,
	}
}

// Ingest saves valid jobs, upserts their companies into the shared catalog,
// then fans out suitability scoring to users tracking each job's company or
// search surface. Save errors are returned; everything after Save is
// fire-and-forget.
func (i *Ingester) Ingest(ctx context.Context, jobs []dto.Job) error {
	valid := make([]dto.Job, 0, len(jobs))
	for _, j := range jobs {
		if j.Title == "" || j.URL == "" {
			slog.Warn("skipping invalid job", slog.String("title", j.Title), slog.String("url", j.URL))
			continue
		}
		valid = append(valid, j)
	}
	if len(valid) == 0 {
		return nil
	}

	saved, err := i.db.Save(ctx, valid)
	if err != nil {
		return err
	}
	slog.Info("jobs ingested", slog.Int("count", len(saved)))

	i.upsertCompanies(ctx, saved)
	i.scoreForTrackingUsers(ctx, saved)

	return nil
}

func (i *Ingester) IngestJobs(ctx context.Context, jobs []dto.Job) ([]Result, error) {
	canonical, ok := i.db.(CanonicalSaver)
	if !ok {
		return nil, errors.New("canonical job persistence unavailable")
	}
	results := make([]Result, len(jobs))
	for idx, job := range jobs {
		if strings.TrimSpace(job.Title) == "" || strings.TrimSpace(job.URL) == "" {
			results[idx] = Result{Status: "rejected", Reason: "title and url are required"}
			continue
		}
		parsedURL, parseErr := url.Parse(job.URL)
		if parseErr != nil || parsedURL.Hostname() == "" || (parsedURL.Scheme != "https" && parsedURL.Scheme != "http") || parsedURL.User != nil {
			results[idx] = Result{Status: "rejected", Reason: "invalid job url"}
			continue
		}
		if (job.BoardID != "" && uuid.Validate(job.BoardID) != nil) ||
			(job.CompanyID != "" && uuid.Validate(job.CompanyID) != nil) {
			results[idx] = Result{Status: "rejected", Reason: "invalid job identity"}
			continue
		}
		saved, status, err := canonical.SaveCanonical(ctx, job)
		if err != nil {
			if errors.Is(err, providers.ErrCanonicalConflict) {
				results[idx] = Result{Status: "rejected", Reason: err.Error()}
				continue
			}
			return nil, err
		}
		results[idx] = Result{Status: status, JobID: saved.ID}
		if status == "unchanged" {
			continue
		}
		i.upsertCompanies(ctx, []dto.Job{saved})
	}
	return results, nil
}

// upsertCompanies records each distinct (source, company_slug) pair among
// saved jobs into the shared company catalog. Never blocks ingest.
func (i *Ingester) upsertCompanies(ctx context.Context, jobs []dto.Job) {
	if i.companies == nil {
		return
	}
	seen := make(map[string]bool)
	for _, j := range jobs {
		if j.CompanySlug == "" || seen[j.CompanySlug] {
			continue
		}
		seen[j.CompanySlug] = true

		atsSource, atsToken := "", ""
		if role, _ := registry.SourceRole(j.Source); role == registry.RoleATS {
			atsSource, atsToken = j.Source, j.CompanySlug
		}
		upsert := dto.CompanyUpsert{
			Slug:      j.CompanySlug,
			Name:      humanizeSlug(j.CompanySlug),
			ATSSource: atsSource,
			ATSToken:  atsToken,
		}
		if _, err := i.companies.UpsertCompany(ctx, upsert); err != nil {
			slog.Warn("ingest: could not upsert company",
				slog.String("slug", j.CompanySlug), slog.Any("err", err))
		}
	}
}

// humanizeSlug turns a slug like "acme-corp" into a display name "Acme Corp".
// ponytail: name derived from slug; good enough until a source carries a real company name.
func humanizeSlug(slug string) string {
	words := strings.Split(slug, "-")
	for idx, w := range words {
		if w == "" {
			continue
		}
		words[idx] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

// scoreForTrackingUsers fans out suitability scoring to users with an enabled
// source_target for each job's company or search surface — not every
// credentialed user. ATS jobs score only for users tracking that exact board;
// discovery jobs score for users with any enabled target on that source.
func (i *Ingester) scoreForTrackingUsers(ctx context.Context, jobs []dto.Job) {
	if i.users == nil || i.creds == nil || i.scorerFor == nil || i.provider == "" {
		return
	}

	type groupKey struct{ source, value string }
	groups := make(map[groupKey][]dto.Job)
	for _, j := range jobs {
		value := ""
		if role, _ := registry.SourceRole(j.Source); role == registry.RoleATS {
			value = j.CompanySlug
		}
		key := groupKey{j.Source, value}
		groups[key] = append(groups[key], j)
	}

	for key, groupJobs := range groups {
		userIDs, err := i.users.ListUserIDsForTarget(ctx, key.source, key.value)
		if err != nil {
			slog.Error("ingest: could not list users for target",
				slog.String("source", key.source), slog.String("value", key.value), slog.Any("err", err))
			continue
		}
		for _, uid := range userIDs {
			apiKey, err := i.creds.Get(ctx, uid, i.provider)
			if err != nil {
				slog.Warn("ingest: could not get credential for user",
					slog.String("user_id", uid), slog.String("provider", i.provider), slog.Any("err", err))
				continue
			}
			scorer := i.scorerFor(apiKey)
			scorer.ScoreAndSaveBatch(ctx, groupJobs, uid)
		}
	}
}

// ProviderForModel resolves the AI provider name for a model ID.
// ponytail: claude-* only; extend when other providers are added.
func ProviderForModel(modelID string) string {
	if strings.HasPrefix(modelID, "claude-") {
		return "anthropic"
	}
	return ""
}

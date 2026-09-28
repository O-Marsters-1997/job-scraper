package jobsearch

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"strings"

	"github.com/google/uuid"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
)

type canonicalSaver interface {
	SaveCanonical(ctx context.Context, job dto.Job) (dto.Job, string, error)
}

type companyUpserter interface {
	UpsertCompany(ctx context.Context, c dto.CompanyUpsert) (dto.Company, error)
}

// IngestResult is one job's ingest outcome.
type IngestResult struct {
	Status string `json:"status"`
	JobID  string `json:"job_id,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// Ingester saves jobs delivered by the worker over POST /ingest, keeping one
// canonical Job per posting.
type Ingester struct {
	jobs      canonicalSaver
	companies companyUpserter
}

func newIngester(jobs canonicalSaver, companies companyUpserter) *Ingester {
	return &Ingester{jobs: jobs, companies: companies}
}

func (i *Ingester) IngestJobs(ctx context.Context, jobs []dto.Job) ([]IngestResult, error) {
	results := make([]IngestResult, len(jobs))
	for idx, job := range jobs {
		if strings.TrimSpace(job.Title) == "" || strings.TrimSpace(job.URL) == "" {
			results[idx] = IngestResult{Status: "rejected", Reason: "title and url are required"}
			continue
		}
		parsedURL, parseErr := url.Parse(job.URL)
		if parseErr != nil || parsedURL.Hostname() == "" || (parsedURL.Scheme != "https" && parsedURL.Scheme != "http") || parsedURL.User != nil {
			results[idx] = IngestResult{Status: "rejected", Reason: "invalid job url"}
			continue
		}
		if (job.BoardID != "" && uuid.Validate(job.BoardID) != nil) ||
			(job.CompanyID != "" && uuid.Validate(job.CompanyID) != nil) {
			results[idx] = IngestResult{Status: "rejected", Reason: "invalid job identity"}
			continue
		}
		saved, status, err := i.jobs.SaveCanonical(ctx, job)
		if err != nil {
			if errors.Is(err, store.ErrCanonicalConflict) {
				results[idx] = IngestResult{Status: "rejected", Reason: err.Error()}
				continue
			}
			return nil, err
		}
		results[idx] = IngestResult{Status: status, JobID: saved.ID}
		if status == "unchanged" {
			continue
		}
		i.upsertCompanies(ctx, []dto.Job{saved})
	}
	if slog.Default().Enabled(ctx, slog.LevelDebug) {
		slog.DebugContext(ctx, "ingest jobs", slog.Int(logger.KeyCount, len(jobs)), slog.Any("by_status", countByStatus(results)))
	}
	return results, nil
}

func countByStatus(results []IngestResult) map[string]int {
	counts := make(map[string]int)
	for _, r := range results {
		counts[r.Status]++
	}
	return counts
}

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
		if role, _ := sourcespec.SourceRole(j.Source); role == sourcespec.RoleATS {
			atsSource, atsToken = j.Source, j.CompanySlug
		}
		upsert := dto.CompanyUpsert{
			Slug:      j.CompanySlug,
			Name:      humanizeSlug(j.CompanySlug),
			ATSSource: atsSource,
			ATSToken:  atsToken,
		}
		if _, err := i.companies.UpsertCompany(ctx, upsert); err != nil {
			slog.WarnContext(ctx, "ingest: could not upsert company",
				slog.String(logger.KeyCompanySlug, j.CompanySlug), slog.Any(logger.KeyErr, err))
		}
	}
}

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

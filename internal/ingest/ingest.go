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

type CanonicalSaver interface {
	SaveCanonical(ctx context.Context, job dto.Job) (dto.Job, string, error)
}

type Result struct {
	Status string `json:"status"`
	JobID  string `json:"job_id,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type CompanyUpserter interface {
	UpsertCompany(ctx context.Context, c dto.CompanyUpsert) (dto.Company, error)
}

type Ingester struct {
	db        CanonicalSaver
	companies CompanyUpserter
}

func New(db CanonicalSaver, companies CompanyUpserter) *Ingester {
	return &Ingester{db: db, companies: companies}
}

func (i *Ingester) IngestJobs(ctx context.Context, jobs []dto.Job) ([]Result, error) {
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
		saved, status, err := i.db.SaveCanonical(ctx, job)
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

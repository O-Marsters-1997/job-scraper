package jobsearch

import (
	"cmp"
	"context"
	"errors"
	"log/slog"
	"net/url"
	"strings"

	"github.com/google/uuid"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
	"github.com/ollymarsters/job-scraper/internal/slug"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
)

// IngestResult is one job's ingest outcome.
type IngestResult struct {
	Status string `json:"status"`
	JobID  string `json:"job_id,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// IngestJobs saves jobs delivered by the worker over POST /ingest, keeping one
// canonical Job per posting.
func (s *Service) IngestJobs(ctx context.Context, jobs []dto.Job) ([]IngestResult, error) {
	results := make([]IngestResult, len(jobs))
	discovered := make(map[string]bool)
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
		saved, status, err := s.store.SaveCanonical(ctx, job)
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
		s.upsertCompany(ctx, saved)
		s.discoverBoard(ctx, job, discovered)
	}
	if slog.Default().Enabled(ctx, slog.LevelDebug) {
		slog.DebugContext(ctx, "ingest jobs", slog.Int(logger.KeyCount, len(jobs)), slog.Any("by_status", countByStatus(results)))
	}
	return results, nil
}

func (s *Service) discoverBoard(ctx context.Context, j dto.Job, seen map[string]bool) {
	target := cmp.Or(j.ApplyURL, j.URL)
	source, token, ok := detect.ResolveBoard(target)
	if !ok {
		var host string
		if u, err := url.Parse(target); err == nil {
			host = u.Hostname()
		}
		if !strings.Contains(host, j.Source) {
			slog.InfoContext(ctx, "ingest: unresolved external host",
				slog.String("host", host), slog.String(logger.KeySource, j.Source))
		}
		return
	}
	key := source + "/" + token
	if source == j.Source || seen[key] {
		return
	}
	if _, err := s.store.GetVerifiedBoardID(ctx, source, token); !errors.Is(err, data.ErrNotFound) {
		if err != nil {
			slog.WarnContext(ctx, "ingest: could not look up board", slog.Any(logger.KeyErr, err))
		}
		return
	}
	seen[key] = true
	task := queue.Task{Version: 1, ID: uuid.NewString(), Source: source, Kind: queue.BoardDiscoverTask, BoardToken: token}
	if err := s.queue.Publish(ctx, task); err != nil {
		slog.WarnContext(ctx, "ingest: could not publish board discover",
			slog.String(logger.KeySource, source), slog.Any(logger.KeyErr, err))
	}
}

func countByStatus(results []IngestResult) map[string]int {
	counts := make(map[string]int)
	for _, r := range results {
		counts[r.Status]++
	}
	return counts
}

func (s *Service) upsertCompany(ctx context.Context, j dto.Job) {
	if j.CompanySlug == "" {
		return
	}
	upsert := dto.CompanyUpsert{Slug: j.CompanySlug, Name: slug.Humanize(j.CompanySlug)}
	if role, _ := sourcespec.SourceRole(j.Source); role == sourcespec.RoleATS {
		upsert.ATSSource, upsert.ATSToken = j.Source, j.CompanySlug
	}
	if _, err := s.store.UpsertCompany(ctx, upsert); err != nil {
		slog.WarnContext(ctx, "ingest: could not upsert company",
			slog.String(logger.KeyCompanySlug, j.CompanySlug), slog.Any(logger.KeyErr, err))
	}
}

package jobsearch

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func (s *Service) SubmitJobURL(ctx context.Context, userID string, in dto.SubmitJobURLInput) (dto.SubmitJobURLResult, error) {
	raw := strings.TrimSpace(in.URL)
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return dto.SubmitJobURLResult{}, apperr.Invalid("enter a valid job URL")
	}

	listingSource, canonical, isListing := detect.ResolveListing(raw)
	candidates := []string{raw}
	if isListing && canonical != raw {
		candidates = append(candidates, canonical)
	}
	var postingID string
	for _, candidate := range candidates {
		postSource, postToken, id, _ := detect.ResolvePosting(candidate)
		if candidate == raw {
			postingID = id
		}
		held, err := s.store.FindJobByPosting(ctx, candidate, postSource, postToken, id)
		switch {
		case err == nil:
			return dto.SubmitJobURLResult{Status: dto.SubmissionHeld, JobID: held.ID}, nil
		case !errors.Is(err, data.ErrNotFound):
			return dto.SubmitJobURLResult{}, err
		}
	}

	source, token, isATS := detect.ResolveBoard(raw)
	if !isATS && isListing {
		queued := dto.QueuedJob{URL: canonical, Card: dto.Job{URL: canonical, Source: listingSource}}
		if err := s.queue.EnqueueJobs(ctx, []dto.QueuedJob{queued}); err != nil {
			return dto.SubmitJobURLResult{}, err
		}
		return dto.SubmitJobURLResult{Status: dto.SubmissionQueued}, nil
	}
	if !isATS {
		return dto.SubmitJobURLResult{}, apperr.Unprocessable(fmt.Sprintf("%s isn't a supported job board. Use Add application instead", parsed.Hostname()))
	}
	var job dto.Job
	if err := s.trackATSCompany(ctx, userID, &job, source, token, postingID, raw, ""); err != nil {
		return dto.SubmitJobURLResult{}, err
	}
	if err := s.syncBoard(ctx, source, token, job.CompanyID); err != nil {
		return dto.SubmitJobURLResult{}, err
	}
	return dto.SubmitJobURLResult{Status: dto.SubmissionQueued}, nil
}

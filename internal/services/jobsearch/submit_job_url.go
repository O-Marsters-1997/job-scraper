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

	postSource, postToken, postingID, _ := detect.ResolvePosting(raw)
	held, err := s.store.FindJobByPosting(ctx, raw, postSource, postToken, postingID)
	switch {
	case err == nil:
		return dto.SubmitJobURLResult{Status: dto.SubmissionHeld, JobID: held.ID}, nil
	case !errors.Is(err, data.ErrNotFound):
		return dto.SubmitJobURLResult{}, err
	}

	source, token, isATS := detect.ResolveBoard(raw)
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

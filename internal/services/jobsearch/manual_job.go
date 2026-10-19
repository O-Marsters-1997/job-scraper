package jobsearch

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/slug"
)

const manualSource = "other"

func (s *Service) AddManualJob(ctx context.Context, userID string, in dto.AddManualJobInput) (dto.ManualJobResult, error) {
	in.URL = strings.TrimSpace(in.URL)
	in.Title = strings.TrimSpace(in.Title)
	in.Company = strings.TrimSpace(in.Company)
	if in.URL == "" || in.Title == "" {
		return dto.ManualJobResult{}, apperr.Invalid("url and title are required")
	}

	postSource, postToken, postingID, _ := detect.ResolvePosting(in.URL)
	held, err := s.store.FindJobByPosting(ctx, in.URL, postSource, postToken, postingID)
	switch {
	case err == nil:
		return s.manualResult(ctx, userID, held.ID, true)
	case !errors.Is(err, data.ErrNotFound):
		return dto.ManualJobResult{}, err
	}

	job := dto.Job{URL: in.URL, Title: in.Title, Description: in.Description}
	source, token, isATS := detect.ResolveBoard(in.URL)
	if isATS {
		if err := s.trackATSCompany(ctx, userID, &job, source, token, postingID, in); err != nil {
			return dto.ManualJobResult{}, err
		}
	} else {
		if in.Company == "" {
			return dto.ManualJobResult{}, apperr.Invalid("company is required")
		}
		job.Source, job.CompanySlug = manualSource, slug.Company(in.Company)
	}

	results, err := s.IngestJobs(ctx, []dto.Job{job})
	if err != nil {
		return dto.ManualJobResult{}, err
	}
	res := results[0]
	if res.Status == "rejected" {
		return dto.ManualJobResult{}, apperr.Invalid(res.Reason)
	}
	saved := res.Status == "new" || res.Status == "upgraded"
	if isATS && saved {
		if err := s.syncBoard(ctx, source, token, job.CompanyID); err != nil {
			return dto.ManualJobResult{}, err
		}
	}
	return s.manualResult(ctx, userID, res.JobID, !saved)
}

func (s *Service) syncBoard(ctx context.Context, source, token, companyID string) error {
	task := queue.Task{Version: 1, ID: uuid.NewString(), Source: source, Urgent: true}
	boardID, err := s.store.GetVerifiedBoardID(ctx, source, token)
	switch {
	case err == nil:
		task.Kind, task.BoardID, task.Manual = queue.BoardCheckTask, boardID, true
	case errors.Is(err, data.ErrNotFound):
		task.Kind, task.CompanyID, task.BoardToken, task.Poll = queue.BoardVerifyTask, companyID, token, true
	default:
		return err
	}
	return s.queue.Publish(ctx, task)
}

func (s *Service) trackATSCompany(ctx context.Context, userID string, job *dto.Job, source, token, postingID string, in dto.AddManualJobInput) error {
	_, err := s.store.GetBoardCompanyID(ctx, source, token)
	newCompany := errors.Is(err, data.ErrNotFound)
	if err != nil && !newCompany {
		return err
	}
	track := true
	company, err := s.CreateCompany(ctx, userID, dto.CreateCompanyInput{URL: in.URL, Track: &track})
	if err != nil {
		return err
	}
	if newCompany && in.Company != "" {
		if err := s.store.RenameCompany(ctx, company.ID, in.Company); err != nil {
			return err
		}
	}
	boards, err := s.store.ListCompanyBoards(ctx, company.ID)
	if err != nil {
		return err
	}
	for _, b := range boards {
		if b.Source == source && b.BoardToken == token {
			job.BoardID = b.ID
		}
	}
	job.Source, job.CompanySlug, job.CompanyID, job.ProviderPostingID = source, company.Slug, company.ID, postingID
	return nil
}

func (s *Service) manualResult(ctx context.Context, userID, jobID string, existing bool) (dto.ManualJobResult, error) {
	job, err := s.Get(ctx, userID, jobID)
	if err != nil {
		return dto.ManualJobResult{}, err
	}
	return dto.ManualJobResult{Job: job, Existing: existing}, nil
}

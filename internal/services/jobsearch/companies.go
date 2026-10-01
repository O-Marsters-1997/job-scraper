package jobsearch

import (
	"context"

	"github.com/google/uuid"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/slug"
)

const defaultCheckIntervalMinutes = 360

func (s *Service) CreateCompany(ctx context.Context, userID string, in dto.CreateCompanyInput) (dto.Company, error) {
	if in.URL == "" {
		return dto.Company{}, apperr.Invalid("url is required")
	}
	source, token, ok := detect.ResolveBoard(in.URL)
	if !ok {
		return dto.Company{}, apperr.Unprocessable("could not resolve an ATS board from that URL")
	}

	company, err := s.store.UpsertCompany(ctx, dto.CompanyUpsert{
		Slug: token,
		Name: slug.Humanize(token),
	})
	if err != nil {
		return dto.Company{}, err
	}
	if _, err := s.store.UpsertCandidateBoard(ctx, company.ID, source, token); err != nil {
		return dto.Company{}, err
	}

	if in.Track == nil || *in.Track {
		if _, err := s.store.SetCompanyTracking(ctx, userID, company.ID, true, defaultCheckIntervalMinutes); err != nil {
			return dto.Company{}, err
		}
	}
	return company, nil
}

func (s *Service) SetCompanyTracking(ctx context.Context, userID string, in dto.SetCompanyTrackingInput) (dto.CompanyTracking, error) {
	if in.Enabled == nil {
		return dto.CompanyTracking{}, apperr.Invalid("enabled is required")
	}
	interval := 0
	if in.CheckIntervalMinutes != nil {
		interval = *in.CheckIntervalMinutes
		if interval < 60 {
			return dto.CompanyTracking{}, apperr.Invalid("check_interval_minutes must be at least 60")
		}
	}

	company, err := s.store.GetCompany(ctx, in.CompanyID)
	if err != nil {
		return dto.CompanyTracking{}, err
	}
	return s.store.SetCompanyTracking(ctx, userID, company.ID, *in.Enabled, interval)
}

func (s *Service) ListTrackedCompanies(ctx context.Context, userID string) ([]dto.TrackedCompany, error) {
	companies, err := s.store.ListTrackedCompaniesForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range companies {
		for j := range companies[i].Boards {
			b := &companies[i].Boards[j]
			b.URL = detect.BoardURL(b.Source, b.BoardToken)
		}
	}
	return companies, nil
}

func (s *Service) SetCompanyReview(ctx context.Context, userID string, in dto.SetCompanyReviewInput) (dto.CompanyTracking, error) {
	switch in.State {
	case "new", "kept", "dismissed":
	default:
		return dto.CompanyTracking{}, apperr.Invalid("state must be new, kept or dismissed")
	}
	return s.store.SetCompanyReviewState(ctx, userID, in.CompanyID, in.State)
}

func (s *Service) UntrackCompany(ctx context.Context, userID, companyID string) error {
	return s.store.DeleteCompanyTracking(ctx, userID, companyID)
}

func (s *Service) ListCompanyBoards(ctx context.Context, _, companyID string) ([]dto.CompanyBoard, error) {
	if _, err := s.store.GetCompany(ctx, companyID); err != nil {
		return nil, err
	}
	return s.store.ListCompanyBoards(ctx, companyID)
}

func (s *Service) AddCompanyBoard(ctx context.Context, _ string, in dto.AddCompanyBoardInput) (dto.CompanyBoard, error) {
	companyID := in.CompanyID
	if _, err := s.store.GetCompany(ctx, companyID); err != nil {
		return dto.CompanyBoard{}, err
	}
	source, token, ok := detect.ResolveBoard(in.URL)
	if !ok {
		return dto.CompanyBoard{}, apperr.Unprocessable("could not resolve an ATS board from that URL")
	}
	board, err := s.store.UpsertCandidateBoard(ctx, companyID, source, token)
	if err != nil {
		return dto.CompanyBoard{}, err
	}
	if in.Confirm && board.Status == dto.BoardCandidate {
		task := queue.Task{Version: 1, ID: uuid.NewString(), Source: source, Kind: queue.BoardVerifyTask, CompanyID: companyID, BoardToken: token}
		if err := s.queue.Publish(ctx, task); err != nil {
			return dto.CompanyBoard{}, err
		}
	}
	return board, nil
}

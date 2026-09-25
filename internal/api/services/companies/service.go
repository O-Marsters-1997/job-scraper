package companies

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
)

const defaultCheckIntervalMinutes = 360

type QueuePublisher interface {
	Publish(ctx context.Context, task queue.Task) error
}

type Service struct {
	companies providers.CompanyProvider
	targets   providers.SourceTargetProvider
	queue     QueuePublisher
}

func New(companies providers.CompanyProvider, targets providers.SourceTargetProvider, q QueuePublisher) *Service {
	return &Service{companies: companies, targets: targets, queue: q}
}

// Create resolves an ATS URL into a company and, unless Track is explicitly
// false, tracks it for the caller.
func (s *Service) Create(ctx context.Context, userID string, in dto.CreateCompanyInput) (dto.Company, error) {
	if in.URL == "" {
		return dto.Company{}, apperr.Invalid("url is required")
	}
	source, token, ok := detect.ResolveBoard(in.URL)
	if !ok {
		return dto.Company{}, apperr.Unprocessable("could not resolve an ATS board from that URL")
	}

	company, err := s.companies.UpsertCompany(ctx, dto.CompanyUpsert{
		Slug: token,
		Name: humanizeToken(token),
	})
	if err != nil {
		return dto.Company{}, err
	}
	if _, err := s.companies.UpsertCandidateBoard(ctx, company.ID, source, token); err != nil {
		return dto.Company{}, err
	}

	if in.Track == nil || *in.Track {
		if _, err := s.companies.SetCompanyTracking(ctx, userID, company.ID, true, defaultCheckIntervalMinutes); err != nil {
			return dto.Company{}, err
		}
	}
	return company, nil
}

// SetTracking stores the caller's company interest and requested check
// frequency, keeping a legacy source target in sync for ATS companies.
func (s *Service) SetTracking(ctx context.Context, userID string, in dto.SetCompanyTrackingInput) (dto.CompanyTracking, error) {
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

	company, err := s.companies.GetCompany(ctx, in.CompanyID)
	if err != nil {
		return dto.CompanyTracking{}, err
	}
	tracking, err := s.companies.SetCompanyTracking(ctx, userID, company.ID, *in.Enabled, interval)
	if err != nil {
		return dto.CompanyTracking{}, err
	}
	if company.ATSSource != "" {
		if _, err := s.targets.UpsertSourceTargetForCompany(ctx, userID, company.ATSSource, company.ATSToken, company.ID, *in.Enabled, tracking.CheckIntervalMinutes); err != nil {
			return dto.CompanyTracking{}, err
		}
	}
	return tracking, nil
}

func (s *Service) ListBoards(ctx context.Context, _, companyID string) ([]dto.CompanyBoard, error) {
	if _, err := s.companies.GetCompany(ctx, companyID); err != nil {
		return nil, err
	}
	return s.companies.ListCompanyBoards(ctx, companyID)
}

func (s *Service) AddBoard(ctx context.Context, _ string, in dto.AddCompanyBoardInput) (dto.CompanyBoard, error) {
	companyID := in.CompanyID
	if _, err := s.companies.GetCompany(ctx, companyID); err != nil {
		return dto.CompanyBoard{}, err
	}
	source, token, ok := detect.ResolveBoard(in.URL)
	if !ok {
		return dto.CompanyBoard{}, apperr.Unprocessable("could not resolve an ATS board from that URL")
	}
	board, err := s.companies.UpsertCandidateBoard(ctx, companyID, source, token)
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

func humanizeToken(token string) string {
	words := strings.Split(token, "-")
	for i, w := range words {
		if w == "" {
			continue
		}
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

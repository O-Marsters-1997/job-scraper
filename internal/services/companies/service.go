// Package companies is the jobsearch context's company and board feature
// (ADR 0011).
package companies

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
)

const defaultCheckIntervalMinutes = 360

type QueuePublisher interface {
	Publish(ctx context.Context, task queue.Task) error
}

type Store interface {
	UpsertCompany(ctx context.Context, c dto.CompanyUpsert) (dto.Company, error)
	GetCompany(ctx context.Context, id string) (dto.Company, error)
	ListCompanyBoards(ctx context.Context, companyID string) ([]dto.CompanyBoard, error)
	UpsertCandidateBoard(ctx context.Context, companyID, source, token string) (dto.CompanyBoard, error)
	SetCompanyTracking(ctx context.Context, userID, companyID string, enabled bool, checkIntervalMinutes int) (dto.CompanyTracking, error)
	VerifyCompanyBoard(ctx context.Context, companyID, source, token, method string) (dto.CompanyBoard, error)
	ListCompaniesToCrawl(ctx context.Context, limit int) ([]dto.Company, error)
	TouchCompanyCrawled(ctx context.Context, id string) error
	ListDueBoards(ctx context.Context) ([]dto.BoardPoll, error)
	ListActiveBoards(ctx context.Context) ([]dto.BoardPoll, error)
	ClaimBoard(ctx context.Context, id string, manual bool) (dto.BoardPoll, error)
	CompleteBoard(ctx context.Context, snapshot dto.BoardSnapshot) error
	FailBoard(ctx context.Context, poll dto.BoardPoll) error
	GetVerifiedBoardID(ctx context.Context, source, token string) (string, error)
	GetLastScraped(ctx context.Context, source string) (time.Time, bool, error)
	SetLastScraped(ctx context.Context, source string) error
}

// SourceTargets keeps a legacy source target in sync when a tracked
// company's ATS board changes; sourcetargets is the jobsearch context's own
// sibling feature.
type SourceTargets interface {
	UpsertSourceTargetForCompany(ctx context.Context, userID, source, value, companyID string, enabled bool, interval int) (dto.SourceTarget, error)
}

type Service struct {
	companies Store
	targets   SourceTargets
	queue     QueuePublisher
}

func New(companies Store, targets SourceTargets, q QueuePublisher) *Service {
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

func (s *Service) UpsertCompany(ctx context.Context, c dto.CompanyUpsert) (dto.Company, error) {
	return s.companies.UpsertCompany(ctx, c)
}

func (s *Service) ListCompaniesToCrawl(ctx context.Context, limit int) ([]dto.Company, error) {
	return s.companies.ListCompaniesToCrawl(ctx, limit)
}

func (s *Service) TouchCompanyCrawled(ctx context.Context, id string) error {
	return s.companies.TouchCompanyCrawled(ctx, id)
}

func (s *Service) VerifyCompanyBoard(ctx context.Context, companyID, source, token, method string) (dto.CompanyBoard, error) {
	return s.companies.VerifyCompanyBoard(ctx, companyID, source, token, method)
}

func (s *Service) ListDueBoards(ctx context.Context) ([]dto.BoardPoll, error) {
	return s.companies.ListDueBoards(ctx)
}

func (s *Service) ListActiveBoards(ctx context.Context) ([]dto.BoardPoll, error) {
	return s.companies.ListActiveBoards(ctx)
}

func (s *Service) ClaimBoard(ctx context.Context, id string, manual bool) (dto.BoardPoll, error) {
	return s.companies.ClaimBoard(ctx, id, manual)
}

func (s *Service) CompleteBoard(ctx context.Context, snapshot dto.BoardSnapshot) error {
	return s.companies.CompleteBoard(ctx, snapshot)
}

func (s *Service) FailBoard(ctx context.Context, poll dto.BoardPoll) error {
	return s.companies.FailBoard(ctx, poll)
}

func (s *Service) GetVerifiedBoardID(ctx context.Context, source, token string) (string, error) {
	return s.companies.GetVerifiedBoardID(ctx, source, token)
}

func (s *Service) GetLastScraped(ctx context.Context, source string) (time.Time, bool, error) {
	return s.companies.GetLastScraped(ctx, source)
}

func (s *Service) SetLastScraped(ctx context.Context, source string) error {
	return s.companies.SetLastScraped(ctx, source)
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

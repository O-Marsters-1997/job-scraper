// Package companies holds the domain rules for resolving ATS boards into
// companies, tracking them for a user, and managing their candidate boards.
// Persistence goes through providers.CompanyProvider and
// providers.SourceTargetProvider.
package companies

import (
	"context"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

const defaultCheckIntervalMinutes = 360

// BoardVerifier checks whether a resolved ATS board actually exists and is
// scrapeable.
type BoardVerifier interface {
	Verify(ctx context.Context, source, token string) error
}

type Service struct {
	companies providers.CompanyProvider
	targets   providers.SourceTargetProvider
	verifier  BoardVerifier
}

func New(companies providers.CompanyProvider, targets providers.SourceTargetProvider, verifier BoardVerifier) *Service {
	return &Service{companies: companies, targets: targets, verifier: verifier}
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
func (s *Service) SetTracking(ctx context.Context, userID, companyID string, in dto.SetCompanyTrackingInput) (dto.CompanyTracking, error) {
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

	company, err := s.companies.GetCompany(ctx, companyID)
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

func (s *Service) ListBoards(ctx context.Context, _ string, companyID string) ([]dto.CompanyBoard, error) {
	if _, err := s.companies.GetCompany(ctx, companyID); err != nil {
		return nil, err
	}
	return s.companies.ListCompanyBoards(ctx, companyID)
}

// AddBoard resolves a URL into an ATS board and links it to the company,
// verifying it immediately when the caller confirms and the verifier
// succeeds.
func (s *Service) AddBoard(ctx context.Context, _ string, companyID string, in dto.AddCompanyBoardInput) (dto.CompanyBoard, error) {
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
	if in.Confirm && board.Status == dto.BoardCandidate && s.verifier.Verify(ctx, source, token) == nil {
		board, err = s.companies.VerifyCompanyBoard(ctx, companyID, source, token, "user_confirmed")
		if err != nil {
			return dto.CompanyBoard{}, err
		}
	}
	return board, nil
}

// humanizeToken turns a board token like "acme-corp" into "Acme Corp".
// ponytail: name derived from the token; good enough until a source carries a real company name.
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

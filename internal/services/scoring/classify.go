package scoring

import (
	"context"
	"log/slog"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
)

func (s *Service) classifyCompany(ctx context.Context, job dto.Job, surviving []dto.SearchConfig) {
	if s.classifier == nil || s.companyKinds == nil || job.CompanyID == "" || job.CompanyKind != dto.KindUnclassified {
		return
	}
	for _, cfg := range surviving {
		key, err := s.credentials.Get(ctx, cfg.UserID, jev.Provider)
		if err != nil {
			continue
		}
		kind, _, err := s.classifier.ClassifyRecruiter(ctx, key, job.CompanySlug, job)
		if err != nil {
			slog.WarnContext(ctx, "classify recruiter failed", slog.String("company_id", job.CompanyID), slog.Any(logger.KeyErr, err))
			return
		}
		if _, err := s.companyKinds.SuggestCompanyKind(ctx, job.CompanyID, kind); err != nil {
			slog.WarnContext(ctx, "suggest company kind failed", slog.String("company_id", job.CompanyID), slog.Any(logger.KeyErr, err))
		}
		return
	}
}

package cvtailor

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/store"
)

const (
	maxTemplateName = 100
	maxTemplateBody = 2000
)

// SeedDefaults seeds userID's default Outreach templates inside tx; identity's
// signup calls this through its Seeder interface (ADR 0011).
func (m *Module) SeedDefaults(ctx context.Context, tx pgx.Tx, userID string) error {
	return m.store.SeedOutreachTemplates(ctx, tx, userID)
}

func (s *Service) CreateOutreachTemplate(ctx context.Context, userID string, in dto.OutreachTemplateInput) (dto.OutreachTemplate, error) {
	switch in.Kind {
	case dto.OutreachRecruiter, dto.OutreachHiringManager, dto.OutreachReferral, dto.OutreachFollowUp:
	default:
		return dto.OutreachTemplate{}, apperr.Invalid("unknown template kind")
	}
	if err := validateTemplate(&in); err != nil {
		return dto.OutreachTemplate{}, err
	}
	out, err := s.store.CreateOutreachTemplate(ctx, userID, in)
	return out, templateErr(err)
}

func (s *Service) UpdateOutreachTemplate(ctx context.Context, userID string, in dto.OutreachTemplateInput) (dto.OutreachTemplate, error) {
	if err := validateTemplate(&in); err != nil {
		return dto.OutreachTemplate{}, err
	}
	out, err := s.store.UpdateOutreachTemplate(ctx, userID, in)
	return out, templateErr(err)
}

func (s *Service) MakeOutreachTemplateDefault(ctx context.Context, userID string, q dto.OutreachTemplateQuery) (dto.OutreachTemplate, error) {
	out, err := s.store.MakeOutreachTemplateDefault(ctx, userID, q.ID)
	return out, templateErr(err)
}

func (s *Service) DeleteOutreachTemplate(ctx context.Context, userID, id string) error {
	return templateErr(s.store.DeleteOutreachTemplate(ctx, userID, id))
}

func templateErr(err error) error {
	switch {
	case errors.Is(err, store.ErrTemplateNotFound):
		return apperr.NotFound("template not found")
	case errors.Is(err, store.ErrTemplateNameTaken):
		return apperr.Conflict("a template of that kind already has that name")
	case errors.Is(err, store.ErrTemplateDefaultSet):
		return apperr.Conflict("make another template the default before deleting this one")
	}
	return err
}

func validateTemplate(in *dto.OutreachTemplateInput) error {
	in.Name = strings.TrimSpace(in.Name)
	in.Body = strings.TrimSpace(in.Body)
	if in.Name == "" || len([]rune(in.Name)) > maxTemplateName {
		return apperr.Invalid("name must be 1 to 100 characters")
	}
	if in.Body == "" || len([]rune(in.Body)) > maxTemplateBody {
		return apperr.Invalid("body must be 1 to 2000 characters")
	}
	return nil
}

package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/store/sqlc"
)

var (
	ErrTemplateNotFound   = errors.New("template not found")
	ErrTemplateNameTaken  = errors.New("template name taken")
	ErrTemplateDefaultSet = errors.New("template is its kind's default")
)

// DefaultOutreachTemplates is the one default per kind a new User starts with.
func DefaultOutreachTemplates() []dto.OutreachTemplateInput {
	return []dto.OutreachTemplateInput{
		{Kind: dto.OutreachRecruiter, Name: "Agency intro", Body: "Hi {first_name}, I'm a backend engineer looking at {company}'s {role} roles. Recently I {achievement}. Open to a quick chat about what you're hiring for?"},
		{Kind: dto.OutreachHiringManager, Name: "Short and direct", Body: "Hi {first_name}, {hook} I {achievement}, and I'd love to bring that to {company}. Worth 15 minutes?"},
		{Kind: dto.OutreachReferral, Name: "Referral ask", Body: "Hi {first_name}, I'm applying to {company} and saw you're on the team. {hook} Would you be open to referring me, or telling me what the team's like?"},
		{Kind: dto.OutreachFollowUp, Name: "Gentle bump", Body: "Hi {first_name}, just bumping this in case it got buried. Still keen on {company}, happy to work around you."},
	}
}

func toOutreachTemplate(t sqlc.OutreachTemplate) dto.OutreachTemplate {
	return dto.OutreachTemplate{ID: t.ID.String(), Kind: dto.OutreachKind(t.Kind), Name: t.Name, Body: t.Body, IsDefault: t.IsDefault}
}

func (s *Store) SeedOutreachTemplates(ctx context.Context, tx pgx.Tx, userID string) error {
	uid, err := data.UUID(userID)
	if err != nil {
		return err
	}
	q := s.queries.WithTx(tx)
	for _, t := range DefaultOutreachTemplates() {
		if err := q.SeedOutreachTemplate(ctx, sqlc.SeedOutreachTemplateParams{UserID: uid, Kind: string(t.Kind), Name: t.Name, Body: t.Body}); err != nil {
			return fmt.Errorf("store.SeedOutreachTemplates: %w", err)
		}
	}
	return nil
}

func (s *Store) ListOutreachTemplates(ctx context.Context, userID string) ([]dto.OutreachTemplate, error) {
	uid, err := parseID(userID, ErrTemplateNotFound)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListOutreachTemplates(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListOutreachTemplates: %w", err)
	}
	out := make([]dto.OutreachTemplate, len(rows))
	for i, r := range rows {
		out[i] = toOutreachTemplate(r)
	}
	return out, nil
}

func (s *Store) CreateOutreachTemplate(ctx context.Context, userID string, in dto.OutreachTemplateInput) (dto.OutreachTemplate, error) {
	uid, err := parseID(userID, ErrTemplateNotFound)
	if err != nil {
		return dto.OutreachTemplate{}, err
	}
	row, err := s.queries.CreateOutreachTemplate(ctx, sqlc.CreateOutreachTemplateParams{UserID: uid, Kind: string(in.Kind), Name: in.Name, Body: in.Body})
	if err != nil {
		return dto.OutreachTemplate{}, templateWriteErr("CreateOutreachTemplate", err)
	}
	return toOutreachTemplate(row), nil
}

func (s *Store) UpdateOutreachTemplate(ctx context.Context, userID string, in dto.OutreachTemplateInput) (dto.OutreachTemplate, error) {
	uid, err := parseID(userID, ErrTemplateNotFound)
	if err != nil {
		return dto.OutreachTemplate{}, err
	}
	id, err := parseID(in.ID, ErrTemplateNotFound)
	if err != nil {
		return dto.OutreachTemplate{}, err
	}
	row, err := s.queries.UpdateOutreachTemplate(ctx, sqlc.UpdateOutreachTemplateParams{UserID: uid, ID: id, Name: in.Name, Body: in.Body})
	if err != nil {
		return dto.OutreachTemplate{}, templateWriteErr("UpdateOutreachTemplate", err)
	}
	return toOutreachTemplate(row), nil
}

func (s *Store) MakeOutreachTemplateDefault(ctx context.Context, userID, id string) (dto.OutreachTemplate, error) {
	uid, err := parseID(userID, ErrTemplateNotFound)
	if err != nil {
		return dto.OutreachTemplate{}, err
	}
	tid, err := parseID(id, ErrTemplateNotFound)
	if err != nil {
		return dto.OutreachTemplate{}, err
	}
	var out dto.OutreachTemplate
	err = s.inTx(ctx, func(q *sqlc.Queries) error {
		current, err := q.GetOutreachTemplate(ctx, sqlc.GetOutreachTemplateParams{UserID: uid, ID: tid})
		if err != nil {
			return templateWriteErr("MakeOutreachTemplateDefault get", err)
		}
		if err := q.ClearOutreachDefault(ctx, sqlc.ClearOutreachDefaultParams{UserID: uid, Kind: current.Kind}); err != nil {
			return fmt.Errorf("store.MakeOutreachTemplateDefault clear: %w", err)
		}
		row, err := q.SetOutreachDefault(ctx, sqlc.SetOutreachDefaultParams{UserID: uid, ID: tid})
		if err != nil {
			return fmt.Errorf("store.MakeOutreachTemplateDefault set: %w", err)
		}
		out = toOutreachTemplate(row)
		return nil
	})
	return out, err
}

// DeleteOutreachTemplate refuses a kind's default while the kind has other
// templates, so every kind that has templates keeps exactly one default.
func (s *Store) DeleteOutreachTemplate(ctx context.Context, userID, id string) error {
	uid, err := parseID(userID, ErrTemplateNotFound)
	if err != nil {
		return err
	}
	tid, err := parseID(id, ErrTemplateNotFound)
	if err != nil {
		return err
	}
	return s.inTx(ctx, func(q *sqlc.Queries) error {
		current, err := q.GetOutreachTemplate(ctx, sqlc.GetOutreachTemplateParams{UserID: uid, ID: tid})
		if err != nil {
			return templateWriteErr("DeleteOutreachTemplate get", err)
		}
		if current.IsDefault {
			n, err := q.CountOutreachTemplatesOfKind(ctx, sqlc.CountOutreachTemplatesOfKindParams{UserID: uid, Kind: current.Kind})
			if err != nil {
				return fmt.Errorf("store.DeleteOutreachTemplate count: %w", err)
			}
			if n > 1 {
				return ErrTemplateDefaultSet
			}
		}
		if err := q.DeleteOutreachTemplate(ctx, sqlc.DeleteOutreachTemplateParams{UserID: uid, ID: tid}); err != nil {
			return fmt.Errorf("store.DeleteOutreachTemplate: %w", err)
		}
		return nil
	})
}

func templateWriteErr(op string, err error) error {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return ErrTemplateNotFound
	case isNameViolation(err):
		return ErrTemplateNameTaken
	}
	return fmt.Errorf("store.%s: %w", op, err)
}

func isNameViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "outreach_templates_user_id_kind_name_key"
}

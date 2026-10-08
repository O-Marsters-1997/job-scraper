package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/store/sqlc"
)

var (
	ErrBankSkillNotFound = apperr.NotFound("bank skill not found")
	ErrBankSkillExists   = apperr.Conflict("a bank skill with that name already exists")
)

func toBankSkill(s sqlc.BankSkill) dto.BankSkill {
	return dto.BankSkill{ID: s.ID.String(), Name: s.Name, Category: s.Category, SortOrder: int(s.SortOrder)}
}

func (s *Store) ListBankSkills(ctx context.Context, userID string) ([]dto.BankSkill, error) {
	uid, err := parseID(userID, ErrBankSkillNotFound)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListBankSkills(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListBankSkills: %w", err)
	}
	out := make([]dto.BankSkill, len(rows))
	for i, r := range rows {
		out[i] = toBankSkill(r)
	}
	return out, nil
}

// CreateBankSkill appends a Bank Skill; a name already in the Bank, ignoring
// case, is ErrBankSkillExists.
func (s *Store) CreateBankSkill(ctx context.Context, userID string, in dto.BankSkillInput) (dto.BankSkill, error) {
	uid, err := parseID(userID, ErrBankSkillNotFound)
	if err != nil {
		return dto.BankSkill{}, err
	}
	row, err := s.queries.CreateBankSkill(ctx, sqlc.CreateBankSkillParams{UserID: uid, Name: in.Name, Category: in.Category})
	if data.IsUniqueViolation(err) {
		return dto.BankSkill{}, ErrBankSkillExists
	}
	if err != nil {
		return dto.BankSkill{}, fmt.Errorf("store.CreateBankSkill: %w", err)
	}
	return toBankSkill(row), nil
}

// UpdateBankSkill renames or recategorises a Bank Skill; a name taken by
// another, ignoring case, is ErrBankSkillExists.
func (s *Store) UpdateBankSkill(ctx context.Context, userID string, in dto.BankSkillInput) (dto.BankSkill, error) {
	uid, err := parseID(userID, ErrBankSkillNotFound)
	if err != nil {
		return dto.BankSkill{}, err
	}
	id, err := parseID(in.ID, ErrBankSkillNotFound)
	if err != nil {
		return dto.BankSkill{}, err
	}
	row, err := s.queries.UpdateBankSkill(ctx, sqlc.UpdateBankSkillParams{UserID: uid, ID: id, Name: in.Name, Category: in.Category})
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.BankSkill{}, ErrBankSkillNotFound
	}
	if data.IsUniqueViolation(err) {
		return dto.BankSkill{}, ErrBankSkillExists
	}
	if err != nil {
		return dto.BankSkill{}, fmt.Errorf("store.UpdateBankSkill: %w", err)
	}
	return toBankSkill(row), nil
}

func (s *Store) DeleteBankSkill(ctx context.Context, userID, id string) error {
	uid, err := parseID(userID, ErrBankSkillNotFound)
	if err != nil {
		return err
	}
	sid, err := parseID(id, ErrBankSkillNotFound)
	if err != nil {
		return err
	}
	n, err := s.queries.DeleteBankSkill(ctx, sqlc.DeleteBankSkillParams{UserID: uid, ID: sid})
	if err != nil {
		return fmt.Errorf("store.DeleteBankSkill: %w", err)
	}
	if n == 0 {
		return ErrBankSkillNotFound
	}
	return nil
}

// ReorderBankSkills sets the sort order to ids, which must be every one of
// userID's Bank Skills exactly once.
func (s *Store) ReorderBankSkills(ctx context.Context, userID string, ids []string) error {
	uid, err := parseID(userID, ErrBankSkillNotFound)
	if err != nil {
		return err
	}
	uuids, err := data.UUIDs(ids)
	if err != nil {
		return err
	}
	return s.inTx(ctx, func(q *sqlc.Queries) error {
		current, err := q.ListBankSkills(ctx, uid)
		if err != nil {
			return fmt.Errorf("store.ReorderBankSkills list: %w", err)
		}
		if len(current) != len(uuids) {
			return ErrIncompleteOrder
		}
		n, err := q.ReorderBankSkills(ctx, sqlc.ReorderBankSkillsParams{UserID: uid, Ids: uuids})
		if err != nil {
			return fmt.Errorf("store.ReorderBankSkills: %w", err)
		}
		if int(n) != len(uuids) {
			return ErrIncompleteOrder
		}
		return nil
	})
}

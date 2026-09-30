// Package store is the cvtailor context's Postgres store: positions and
// achievements.
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/store/sqlc"
)

const dateLayout = time.DateOnly

var (
	ErrPositionNotFound    = apperr.NotFound("position not found")
	ErrAchievementNotFound = apperr.NotFound("achievement not found")
	ErrIncompleteOrder     = apperr.Invalid("ids must list every item exactly once")
)

type Store struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, queries: sqlc.New(pool)}
}

func parseID(s string, notFound error) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(s); err != nil {
		return pgtype.UUID{}, notFound
	}
	return id, nil
}

func parseDate(s *string) (pgtype.Date, error) {
	if s == nil {
		return pgtype.Date{}, nil
	}
	t, err := time.Parse(dateLayout, *s)
	if err != nil {
		return pgtype.Date{}, apperr.Invalid("dates must be YYYY-MM-DD")
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}

func formatDate(d pgtype.Date) *string {
	if !d.Valid {
		return nil
	}
	s := d.Time.Format(dateLayout)
	return &s
}

func toPosition(p sqlc.Position) dto.Position {
	return dto.Position{
		ID:           p.ID.String(),
		Employer:     p.Employer,
		Title:        p.Title,
		StartDate:    formatDate(p.StartDate),
		EndDate:      formatDate(p.EndDate),
		Achievements: []dto.Achievement{},
	}
}

func toAchievement(a sqlc.Achievement) dto.Achievement {
	return dto.Achievement{ID: a.ID.String(), PositionID: a.PositionID.String(), Text: a.Text}
}

// ListPositions returns userID's Positions in sort order, each with its
// Achievements in sort order.
func (s *Store) ListPositions(ctx context.Context, userID string) ([]dto.Position, error) {
	uid, err := parseID(userID, ErrPositionNotFound)
	if err != nil {
		return nil, err
	}
	return listPositions(ctx, s.queries, uid)
}

func listPositions(ctx context.Context, q *sqlc.Queries, uid pgtype.UUID) ([]dto.Position, error) {
	positions, err := q.ListPositions(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListPositions: %w", err)
	}
	achievements, err := q.ListAchievements(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListPositions achievements: %w", err)
	}
	out := make([]dto.Position, len(positions))
	byID := make(map[pgtype.UUID]int, len(positions))
	for i, p := range positions {
		out[i] = toPosition(p)
		byID[p.ID] = i
	}
	for _, a := range achievements {
		if i, ok := byID[a.PositionID]; ok {
			out[i].Achievements = append(out[i].Achievements, toAchievement(a))
		}
	}
	return out, nil
}

// CreatePosition adds a Position above the user's existing ones.
func (s *Store) CreatePosition(ctx context.Context, userID string, in dto.PositionInput) (dto.Position, error) {
	uid, err := parseID(userID, ErrPositionNotFound)
	if err != nil {
		return dto.Position{}, err
	}
	start, err := parseDate(in.StartDate)
	if err != nil {
		return dto.Position{}, err
	}
	end, err := parseDate(in.EndDate)
	if err != nil {
		return dto.Position{}, err
	}
	p, err := s.queries.CreatePosition(ctx, sqlc.CreatePositionParams{
		UserID: uid, Employer: in.Employer, Title: in.Title, StartDate: start, EndDate: end,
	})
	if err != nil {
		return dto.Position{}, fmt.Errorf("store.CreatePosition: %w", err)
	}
	return toPosition(p), nil
}

func (s *Store) UpdatePosition(ctx context.Context, userID string, in dto.PositionInput) (dto.Position, error) {
	uid, err := parseID(userID, ErrPositionNotFound)
	if err != nil {
		return dto.Position{}, err
	}
	id, err := parseID(in.ID, ErrPositionNotFound)
	if err != nil {
		return dto.Position{}, err
	}
	start, err := parseDate(in.StartDate)
	if err != nil {
		return dto.Position{}, err
	}
	end, err := parseDate(in.EndDate)
	if err != nil {
		return dto.Position{}, err
	}
	p, err := s.queries.UpdatePosition(ctx, sqlc.UpdatePositionParams{
		UserID: uid, ID: id, Employer: in.Employer, Title: in.Title, StartDate: start, EndDate: end,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.Position{}, ErrPositionNotFound
	}
	if err != nil {
		return dto.Position{}, fmt.Errorf("store.UpdatePosition: %w", err)
	}
	full, err := s.queries.ListAchievements(ctx, uid)
	if err != nil {
		return dto.Position{}, fmt.Errorf("store.UpdatePosition achievements: %w", err)
	}
	out := toPosition(p)
	for _, a := range full {
		if a.PositionID == p.ID {
			out.Achievements = append(out.Achievements, toAchievement(a))
		}
	}
	return out, nil
}

// DeletePosition removes the Position; its Achievements cascade.
func (s *Store) DeletePosition(ctx context.Context, userID, id string) error {
	uid, err := parseID(userID, ErrPositionNotFound)
	if err != nil {
		return err
	}
	pid, err := parseID(id, ErrPositionNotFound)
	if err != nil {
		return err
	}
	n, err := s.queries.DeletePosition(ctx, sqlc.DeletePositionParams{UserID: uid, ID: pid})
	if err != nil {
		return fmt.Errorf("store.DeletePosition: %w", err)
	}
	if n == 0 {
		return ErrPositionNotFound
	}
	return nil
}

// ReorderPositions sets the sort order to ids, which must be every one of
// userID's Positions exactly once.
func (s *Store) ReorderPositions(ctx context.Context, userID string, ids []string) error {
	uid, err := parseID(userID, ErrPositionNotFound)
	if err != nil {
		return err
	}
	uuids, err := data.UUIDs(ids)
	if err != nil {
		return err
	}
	return s.inTx(ctx, func(q *sqlc.Queries) error {
		current, err := q.ListPositions(ctx, uid)
		if err != nil {
			return fmt.Errorf("store.ReorderPositions list: %w", err)
		}
		if len(current) != len(uuids) {
			return ErrIncompleteOrder
		}
		n, err := q.ReorderPositions(ctx, sqlc.ReorderPositionsParams{UserID: uid, Ids: uuids})
		if err != nil {
			return fmt.Errorf("store.ReorderPositions: %w", err)
		}
		if int(n) != len(uuids) {
			return ErrIncompleteOrder
		}
		return nil
	})
}

// CreateAchievement appends an Achievement to the end of the Position.
func (s *Store) CreateAchievement(ctx context.Context, userID string, in dto.AchievementInput) (dto.Achievement, error) {
	uid, err := parseID(userID, ErrPositionNotFound)
	if err != nil {
		return dto.Achievement{}, err
	}
	pid, err := parseID(in.PositionID, ErrPositionNotFound)
	if err != nil {
		return dto.Achievement{}, err
	}
	a, err := s.queries.CreateAchievement(ctx, sqlc.CreateAchievementParams{UserID: uid, PositionID: pid, Text: in.Text})
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.Achievement{}, ErrPositionNotFound
	}
	if err != nil {
		return dto.Achievement{}, fmt.Errorf("store.CreateAchievement: %w", err)
	}
	return toAchievement(a), nil
}

func (s *Store) UpdateAchievement(ctx context.Context, userID string, in dto.AchievementInput) (dto.Achievement, error) {
	uid, err := parseID(userID, ErrAchievementNotFound)
	if err != nil {
		return dto.Achievement{}, err
	}
	id, err := parseID(in.ID, ErrAchievementNotFound)
	if err != nil {
		return dto.Achievement{}, err
	}
	a, err := s.queries.UpdateAchievement(ctx, sqlc.UpdateAchievementParams{UserID: uid, ID: id, Text: in.Text})
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.Achievement{}, ErrAchievementNotFound
	}
	if err != nil {
		return dto.Achievement{}, fmt.Errorf("store.UpdateAchievement: %w", err)
	}
	return toAchievement(a), nil
}

func (s *Store) DeleteAchievement(ctx context.Context, userID, id string) error {
	uid, err := parseID(userID, ErrAchievementNotFound)
	if err != nil {
		return err
	}
	aid, err := parseID(id, ErrAchievementNotFound)
	if err != nil {
		return err
	}
	n, err := s.queries.DeleteAchievement(ctx, sqlc.DeleteAchievementParams{UserID: uid, ID: aid})
	if err != nil {
		return fmt.Errorf("store.DeleteAchievement: %w", err)
	}
	if n == 0 {
		return ErrAchievementNotFound
	}
	return nil
}

// ReorderAchievements sets the sort order to ids, which must be every
// Achievement of the Position exactly once.
func (s *Store) ReorderAchievements(ctx context.Context, userID, positionID string, ids []string) error {
	uid, err := parseID(userID, ErrPositionNotFound)
	if err != nil {
		return err
	}
	pid, err := parseID(positionID, ErrPositionNotFound)
	if err != nil {
		return err
	}
	uuids, err := data.UUIDs(ids)
	if err != nil {
		return err
	}
	return s.inTx(ctx, func(q *sqlc.Queries) error {
		if _, err := q.GetPosition(ctx, sqlc.GetPositionParams{UserID: uid, ID: pid}); errors.Is(err, pgx.ErrNoRows) {
			return ErrPositionNotFound
		} else if err != nil {
			return fmt.Errorf("store.ReorderAchievements position: %w", err)
		}
		all, err := q.ListAchievements(ctx, uid)
		if err != nil {
			return fmt.Errorf("store.ReorderAchievements list: %w", err)
		}
		count := 0
		for _, a := range all {
			if a.PositionID == pid {
				count++
			}
		}
		if count != len(uuids) {
			return ErrIncompleteOrder
		}
		n, err := q.ReorderAchievements(ctx, sqlc.ReorderAchievementsParams{UserID: uid, PositionID: pid, Ids: uuids})
		if err != nil {
			return fmt.Errorf("store.ReorderAchievements: %w", err)
		}
		if int(n) != len(uuids) {
			return ErrIncompleteOrder
		}
		return nil
	})
}

func (s *Store) inTx(ctx context.Context, fn func(q *sqlc.Queries) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(s.queries.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

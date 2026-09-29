package store

import (
	"context"
	"fmt"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/store/sqlc"
)

func (s *Store) ImportPositions(ctx context.Context, userID string, in []dto.ImportPosition) ([]dto.Position, error) {
	uid, err := parseID(userID, ErrPositionNotFound)
	if err != nil {
		return nil, err
	}
	out := make([]dto.Position, len(in))
	err = s.inTx(ctx, func(q *sqlc.Queries) error {
		for i := len(in) - 1; i >= 0; i-- {
			p := in[i]
			start, err := parseDate(p.StartDate)
			if err != nil {
				return err
			}
			end, err := parseDate(p.EndDate)
			if err != nil {
				return err
			}
			row, err := q.CreatePosition(ctx, sqlc.CreatePositionParams{
				UserID: uid, Employer: p.Employer, Title: p.Title, StartDate: start, EndDate: end,
			})
			if err != nil {
				return fmt.Errorf("store.ImportPositions position: %w", err)
			}
			out[i] = toPosition(row)
			for _, text := range p.Achievements {
				a, err := q.CreateAchievement(ctx, sqlc.CreateAchievementParams{UserID: uid, PositionID: row.ID, Text: text})
				if err != nil {
					return fmt.Errorf("store.ImportPositions achievement: %w", err)
				}
				out[i].Achievements = append(out[i].Achievements, toAchievement(a))
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

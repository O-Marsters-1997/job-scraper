package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/store/sqlc"
)

func (s *Store) ListHeadingMappings(ctx context.Context, userID, docID, tabID string) ([]dto.HeadingMapping, error) {
	uid, err := parseID(userID, ErrPositionNotFound)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListHeadingMappings(ctx, sqlc.ListHeadingMappingsParams{UserID: uid, DocID: docID, TabID: tabID})
	if err != nil {
		return nil, fmt.Errorf("store.ListHeadingMappings: %w", err)
	}
	out := make([]dto.HeadingMapping, len(rows))
	for i, r := range rows {
		out[i] = dto.HeadingMapping{HeadingText: r.HeadingText}
		if r.PositionID.Valid {
			id := r.PositionID.String()
			out[i].PositionID = &id
		}
	}
	return out, nil
}

func (s *Store) SaveHeadingMappings(ctx context.Context, userID, docID, tabID string, mappings []dto.HeadingMapping) error {
	uid, err := parseID(userID, ErrPositionNotFound)
	if err != nil {
		return err
	}
	return s.inTx(ctx, func(q *sqlc.Queries) error {
		for _, m := range mappings {
			var pid pgtype.UUID
			if m.PositionID != nil {
				if pid, err = parseID(*m.PositionID, ErrPositionNotFound); err != nil {
					return err
				}
			}
			if err := q.UpsertHeadingMapping(ctx, sqlc.UpsertHeadingMappingParams{
				UserID: uid, DocID: docID, TabID: tabID, HeadingText: m.HeadingText, PositionID: pid,
			}); err != nil {
				return fmt.Errorf("store.SaveHeadingMappings: %w", err)
			}
		}
		return nil
	})
}

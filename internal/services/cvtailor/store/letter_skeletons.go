package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/store/sqlc"
)

func toLetterSkeleton(r sqlc.LetterSkeleton) dto.LetterSkeleton {
	return dto.LetterSkeleton{Intro: r.Intro, Conclusion: r.Conclusion, Sections: r.Sections}
}

func (s *Store) GetLetterSkeleton(ctx context.Context, userID string) (dto.LetterSkeleton, error) {
	uid, err := parseID(userID, ErrPositionNotFound)
	if err != nil {
		return dto.LetterSkeleton{}, err
	}
	row, err := s.queries.GetLetterSkeleton(ctx, uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.LetterSkeleton{Sections: []string{}}, nil
	}
	if err != nil {
		return dto.LetterSkeleton{}, fmt.Errorf("store.GetLetterSkeleton: %w", err)
	}
	return toLetterSkeleton(row), nil
}

func (s *Store) SaveLetterSkeleton(ctx context.Context, userID string, in dto.LetterSkeleton) (dto.LetterSkeleton, error) {
	uid, err := parseID(userID, ErrPositionNotFound)
	if err != nil {
		return dto.LetterSkeleton{}, err
	}
	sections := in.Sections
	if sections == nil {
		sections = []string{}
	}
	row, err := s.queries.UpsertLetterSkeleton(ctx, sqlc.UpsertLetterSkeletonParams{
		UserID: uid, Intro: in.Intro, Conclusion: in.Conclusion, Sections: sections,
	})
	if err != nil {
		return dto.LetterSkeleton{}, fmt.Errorf("store.SaveLetterSkeleton: %w", err)
	}
	return toLetterSkeleton(row), nil
}

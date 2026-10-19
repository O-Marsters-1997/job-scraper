package store

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/store/sqlc"
)

func toCoverLetter(r sqlc.CoverLetter) dto.CoverLetter {
	ids := make([]string, len(r.AchievementIds))
	for i, id := range r.AchievementIds {
		ids[i] = id.String()
	}
	return dto.CoverLetter{
		ID: r.ID.String(), JobID: r.JobID.String(), SourceDocID: r.SourceDocID, SourceTabID: r.SourceTabID,
		TabURL:         fmt.Sprintf("https://docs.google.com/document/d/%s/edit?tab=%s", url.PathEscape(r.SourceDocID), url.QueryEscape(r.ScaffoldTabID)),
		AchievementIDs: ids, Cost: float64(r.Cost), CreatedAt: r.CreatedAt.Time,
	}
}

func (s *Store) CreateCoverLetter(ctx context.Context, userID string, in dto.NewCoverLetter) (dto.CoverLetter, error) {
	uid, err := parseID(userID, ErrJobNotFound)
	if err != nil {
		return dto.CoverLetter{}, err
	}
	jid, err := parseID(in.JobID, ErrJobNotFound)
	if err != nil {
		return dto.CoverLetter{}, err
	}
	achievements, err := data.UUIDs(in.AchievementIDs)
	if err != nil {
		return dto.CoverLetter{}, fmt.Errorf("store.CreateCoverLetter: %w", err)
	}
	row, err := s.queries.InsertCoverLetter(ctx, sqlc.InsertCoverLetterParams{
		UserID: uid, JobID: jid, SourceDocID: in.SourceDocID, SourceTabID: in.SourceTabID,
		ScaffoldTabID: in.ScaffoldTabID, AchievementIds: achievements, Model: in.Model, Cost: float32(in.Cost),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.CoverLetter{}, ErrJobNotFound
	}
	if err != nil {
		return dto.CoverLetter{}, fmt.Errorf("store.CreateCoverLetter: %w", err)
	}
	return toCoverLetter(row), nil
}

func (s *Store) ListJobCoverLetters(ctx context.Context, userID, jobID string) ([]dto.CoverLetter, error) {
	uid, err := parseID(userID, ErrJobNotFound)
	if err != nil {
		return nil, err
	}
	jid, err := parseID(jobID, ErrJobNotFound)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListJobCoverLetters(ctx, sqlc.ListJobCoverLettersParams{JobID: jid, UserID: uid})
	if err != nil {
		return nil, fmt.Errorf("store.ListJobCoverLetters: %w", err)
	}
	out := make([]dto.CoverLetter, len(rows))
	for i, r := range rows {
		out[i] = toCoverLetter(r)
	}
	return out, nil
}

func (s *Store) IsLetterTab(ctx context.Context, userID, docID, tabID string) (bool, error) {
	uid, err := parseID(userID, ErrJobNotFound)
	if err != nil {
		return false, err
	}
	ok, err := s.queries.IsLetterTab(ctx, sqlc.IsLetterTabParams{UserID: uid, DocID: docID, TabID: tabID})
	if err != nil {
		return false, fmt.Errorf("store.IsLetterTab: %w", err)
	}
	return ok, nil
}

func (s *Store) JobBrief(ctx context.Context, jobID string) (dto.JobBrief, error) {
	jid, err := parseID(jobID, ErrJobNotFound)
	if err != nil {
		return dto.JobBrief{}, err
	}
	row, err := s.queries.GetJobBrief(ctx, jid)
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.JobBrief{}, ErrJobNotFound
	}
	if err != nil {
		return dto.JobBrief{}, fmt.Errorf("store.JobBrief: %w", err)
	}
	return dto.JobBrief{Title: row.Title, Company: row.CompanyName, Description: row.Description}, nil
}

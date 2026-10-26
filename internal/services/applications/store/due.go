package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func (s *Store) ListDue(ctx context.Context, userID string) (dto.DueList, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.DueList{}, err
	}
	rows, err := s.queries.ListDue(ctx, uid)
	if err != nil {
		return dto.DueList{}, fmt.Errorf("store.ListDue: %w", err)
	}
	out := dto.DueList{Jobs: make([]dto.DueJob, len(rows))}
	for i, r := range rows {
		j := dto.DueJob{
			JobID:         r.JobID.String(),
			Title:         r.JobTitle,
			CompanySlug:   r.CompanySlug,
			Band:          r.Band,
			ApplicationID: r.ApplicationID.String(),
		}
		if r.Score.Valid {
			score := int(r.Score.Int32)
			j.Score = &score
		}
		if r.ChaseBy.Valid {
			day := r.ChaseBy.Time.Format(time.DateOnly)
			j.ChaseBy = &day
		}
		if r.ApplyTaskID.Valid {
			id := r.ApplyTaskID.String()
			j.ApplyTaskID = &id
		}
		if err := json.Unmarshal(r.Tasks, &j.Tasks); err != nil {
			return dto.DueList{}, fmt.Errorf("store.ListDue: decode tasks: %w", err)
		}
		if r.DraftID.Valid {
			j.Draft = &dto.DueDraft{ID: r.DraftID.String(), State: dto.DraftState(r.DraftState), CVTitle: r.DraftCvTitle}
		}
		out.Jobs[i] = j
	}
	return out, nil
}

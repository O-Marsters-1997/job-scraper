package scoring

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
)

const (
	feedbackKindJob        = "job"
	feedbackKindCollection = "collection"
	feedbackKindOverall    = "overall"
	feedbackPageSize       = 20
	exportAllFeedback      = math.MaxInt32
)

// AppendOverallFeedback logs a reason about the scoring as a whole, beside
// userID's current Picks and the live Jev model.
func (s *Service) AppendOverallFeedback(ctx context.Context, userID string, in dto.OverallFeedbackInput) (dto.ScoreFeedback, error) {
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		return dto.ScoreFeedback{}, apperr.Invalid("reason must not be blank")
	}
	cfg, err := s.searchConfigOrZero(ctx, userID)
	if err != nil {
		return dto.ScoreFeedback{}, err
	}
	picks := cfg.Preferences.Picks
	if picks == nil {
		picks = []dto.Pick{}
	}
	return s.store.InsertScoreFeedback(ctx, userID, dto.ScoreFeedback{
		Kind: feedbackKindOverall, Reason: reason, Picks: picks, Model: jev.Model,
	})
}

// ListFeedback returns one page of userID's log, newest first, with the total
// matching q.Kind.
func (s *Service) ListFeedback(ctx context.Context, userID string, q dto.ScoreFeedbackQuery) (dto.ScoreFeedbackPage, error) {
	switch q.Kind {
	case "", feedbackKindJob, feedbackKindCollection, feedbackKindOverall:
	default:
		return dto.ScoreFeedbackPage{}, apperr.Invalid("unknown kind")
	}
	page := 1
	if q.Page != "" {
		n, err := strconv.Atoi(q.Page)
		if err != nil || n < 1 {
			return dto.ScoreFeedbackPage{}, apperr.Invalid("page must be a positive integer")
		}
		page = n
	}
	entries, err := s.store.ListScoreFeedback(ctx, userID, q.Kind, feedbackPageSize, (page-1)*feedbackPageSize)
	if err != nil {
		return dto.ScoreFeedbackPage{}, err
	}
	total, err := s.store.CountScoreFeedback(ctx, userID, q.Kind)
	if err != nil {
		return dto.ScoreFeedbackPage{}, err
	}
	return dto.ScoreFeedbackPage{Entries: entries, Total: total}, nil
}

// DeleteFeedback removes one of userID's entries.
func (s *Service) DeleteFeedback(ctx context.Context, userID, id string) error {
	err := s.store.DeleteScoreFeedback(ctx, userID, id)
	if errors.Is(err, data.ErrNotFound) {
		return apperr.NotFound("feedback entry not found")
	}
	return err
}

// ExportFeedback renders userID's whole log as a Feedback Pack.
func (s *Service) ExportFeedback(ctx context.Context, userID string) (string, error) {
	entries, err := s.store.ListScoreFeedback(ctx, userID, "", exportAllFeedback, 0)
	if err != nil {
		return "", err
	}
	cfg, err := s.searchConfigOrZero(ctx, userID)
	if err != nil {
		return "", err
	}
	b, err := s.loadBank(ctx)
	if err != nil {
		return "", err
	}
	return renderPack(entries, packPicks(cfg.Preferences.Picks, b)), nil
}

// ClearFeedback hard-deletes userID's log and returns how many entries went.
func (s *Service) ClearFeedback(ctx context.Context, userID string) (int64, error) {
	return s.store.ClearScoreFeedback(ctx, userID)
}

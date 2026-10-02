package scoring

import (
	"context"
	"math"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
)

const (
	feedbackKindOverall = "overall"
	feedbackPageSize    = 20
	exportAllFeedback   = math.MaxInt32
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

// ListFeedback returns the first page of userID's log, newest first.
func (s *Service) ListFeedback(ctx context.Context, userID string) ([]dto.ScoreFeedback, error) {
	return s.store.ListScoreFeedback(ctx, userID, feedbackPageSize)
}

// ExportFeedback renders userID's whole log as a Feedback Pack.
func (s *Service) ExportFeedback(ctx context.Context, userID string) (string, error) {
	entries, err := s.store.ListScoreFeedback(ctx, userID, exportAllFeedback)
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

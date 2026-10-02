package scoring

import (
	"context"
	"errors"
	"fmt"
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
	feedbackHigher         = "higher"
	feedbackLower          = "lower"
	maxFeedbackPage        = 100000
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

// AppendJobFeedback logs that jobID's score should move in in.Direction,
// freezing the stored score, the picked Options' cached answers and the job
// as Jev saw it. The browser sends none of that evidence.
func (s *Service) AppendJobFeedback(ctx context.Context, userID string, in dto.JobFeedbackInput) (dto.ScoreFeedback, error) {
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		return dto.ScoreFeedback{}, apperr.Invalid("reason must not be blank")
	}
	if in.Direction != feedbackHigher && in.Direction != feedbackLower {
		return dto.ScoreFeedback{}, apperr.Invalid("direction must be higher or lower")
	}
	job, err := s.store.GetJobForScoring(ctx, in.JobID)
	if notFound(err) {
		return dto.ScoreFeedback{}, apperr.NotFound("job not found")
	}
	if err != nil {
		return dto.ScoreFeedback{}, fmt.Errorf("scoring.AppendJobFeedback: load job: %w", err)
	}
	score, err := s.store.GetJobScoreForFeedback(ctx, userID, in.JobID)
	if notFound(err) {
		return dto.ScoreFeedback{}, apperr.Unprocessable("job has no score yet")
	}
	if err != nil {
		return dto.ScoreFeedback{}, fmt.Errorf("scoring.AppendJobFeedback: load score: %w", err)
	}
	cfg, err := s.searchConfigOrZero(ctx, userID)
	if err != nil {
		return dto.ScoreFeedback{}, err
	}
	b, err := s.loadBank(ctx)
	if err != nil {
		return dto.ScoreFeedback{}, err
	}
	answers, err := s.store.ListAnswers(ctx, in.JobID, job.ContentFingerprint, jev.Model)
	if err != nil {
		return dto.ScoreFeedback{}, fmt.Errorf("scoring.AppendJobFeedback: load answers: %w", err)
	}

	picks := cfg.Preferences.Picks
	if picks == nil {
		picks = []dto.Pick{}
	}
	state := jev.StateFor(job)
	return s.store.InsertScoreFeedback(ctx, userID, dto.ScoreFeedback{
		Kind: feedbackKindJob, Direction: &in.Direction, JobID: &in.JobID, Reason: reason, Picks: picks, Model: jev.Model,
		Snapshot: dto.ScoreFeedbackSnapshot{
			Score: &score.Score, Breakdown: score.Breakdown, ScoreFingerprint: score.Fingerprint,
			ScoreModel: score.Model, ContentFingerprint: job.ContentFingerprint,
			Options: feedbackOptions(picks, b, answers), JevState: &state,
		},
	})
}

func feedbackOptions(picks []dto.Pick, b bank, answers map[string]dto.Answer) []dto.FeedbackOption {
	picks = dedupeBySource(picks)
	out := make([]dto.FeedbackOption, 0, len(picks))
	for _, p := range picks {
		opt, ok := b.byID[p.OptionID]
		if !ok {
			continue
		}
		a, known := answers[QuestionHash(opt.Question)]
		resolved := "unknown"
		switch {
		case opt.RetiredAt != nil:
			resolved = "retired"
		case known:
			resolved = resolveAnswer(a)
		}
		out = append(out, dto.FeedbackOption{
			OptionID: opt.ID, Label: opt.Label, Question: opt.Question, Stance: p.Stance, Resolved: resolved,
			PYes: a.PYes, PNo: a.PNo, PNotStated: a.PNotStated, Confidence: a.Confidence, Known: known,
		})
	}
	return out
}

// ListFeedback returns one page of userID's log, newest first, with the total
// matching q. Outdated entries are listed only when q.Outdated is "true".
func (s *Service) ListFeedback(ctx context.Context, userID string, q dto.ScoreFeedbackQuery) (dto.ScoreFeedbackPage, error) {
	switch q.Kind {
	case "", feedbackKindJob, feedbackKindCollection, feedbackKindOverall:
	default:
		return dto.ScoreFeedbackPage{}, apperr.Invalid("unknown kind")
	}
	page := 1
	if q.Page != "" {
		n, err := strconv.Atoi(q.Page)
		if err != nil || n < 1 || n > maxFeedbackPage {
			return dto.ScoreFeedbackPage{}, apperr.Invalid("page must be between 1 and 100000")
		}
		page = n
	}
	f := dto.ScoreFeedbackFilter{Kind: q.Kind, Model: jev.Model, IncludeOutdated: q.Outdated == "true"}
	entries, err := s.store.ListScoreFeedback(ctx, userID, f, feedbackPageSize, (page-1)*feedbackPageSize)
	if err != nil {
		return dto.ScoreFeedbackPage{}, err
	}
	current, outdated, err := s.store.CountScoreFeedback(ctx, userID, f)
	if err != nil {
		return dto.ScoreFeedbackPage{}, err
	}
	total := current
	if f.IncludeOutdated {
		total += outdated
	}
	return dto.ScoreFeedbackPage{Entries: entries, Total: total, CurrentCount: current, OutdatedCount: outdated}, nil
}

// DeleteFeedback removes one of userID's entries.
func (s *Service) DeleteFeedback(ctx context.Context, userID, id string) error {
	err := s.store.DeleteScoreFeedback(ctx, userID, id)
	if errors.Is(err, data.ErrNotFound) {
		return apperr.NotFound("feedback entry not found")
	}
	return err
}

// ExportFeedback renders userID's log as a Feedback Pack. Outdated entries
// are left out, and counted in the header, unless includeOutdated.
func (s *Service) ExportFeedback(ctx context.Context, userID string, includeOutdated bool) (string, error) {
	f := dto.ScoreFeedbackFilter{Model: jev.Model, IncludeOutdated: includeOutdated}
	entries, err := s.store.ListScoreFeedback(ctx, userID, f, exportAllFeedback, 0)
	if err != nil {
		return "", err
	}
	_, outdated, err := s.store.CountScoreFeedback(ctx, userID, f)
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
	return renderPack(entries, packPicks(cfg.Preferences.Picks, b), outdated, includeOutdated), nil
}

// ClearFeedback hard-deletes userID's log and returns how many entries went.
func (s *Service) ClearFeedback(ctx context.Context, userID string) (int64, error) {
	return s.store.ClearScoreFeedback(ctx, userID)
}

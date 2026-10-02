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
	maxCollectionJobs      = 500
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
		if err != nil || n < 1 || n > maxFeedbackPage {
			return dto.ScoreFeedbackPage{}, apperr.Invalid("page must be between 1 and 100000")
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

// AppendCollectionFeedback logs a reason about the ranking of in.JobIDs as the
// user saw it. Each rank keeps the submitted position and the score the DB
// holds; the browser sends none of that.
func (s *Service) AppendCollectionFeedback(ctx context.Context, userID string, in dto.CollectionFeedbackInput) (dto.ScoreFeedback, error) {
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		return dto.ScoreFeedback{}, apperr.Invalid("reason must not be blank")
	}
	if n := len(in.JobIDs); n < 1 || n > maxCollectionJobs {
		return dto.ScoreFeedback{}, apperr.Invalid(fmt.Sprintf("jobIds must hold between 1 and %d ids", maxCollectionJobs))
	}
	scores, err := s.store.ListJobScoresForCollection(ctx, userID, in.JobIDs)
	if err != nil {
		return dto.ScoreFeedback{}, fmt.Errorf("scoring.AppendCollectionFeedback: load scores: %w", err)
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
		Kind: feedbackKindCollection, Reason: reason, Picks: picks, Model: jev.Model,
		Snapshot: dto.ScoreFeedbackSnapshot{Filters: in.Filters, Ranking: rankJobs(in.JobIDs, scores)},
	})
}

func rankJobs(jobIDs []string, scores []dto.CollectionJobScore) []dto.RankedJob {
	byID := make(map[string]dto.CollectionJobScore, len(scores))
	for _, sc := range scores {
		byID[sc.JobID] = sc
	}
	ranking := make([]dto.RankedJob, len(jobIDs))
	for i, id := range jobIDs {
		sc := byID[id]
		effects := make([]string, 0, len(sc.Breakdown))
		for _, row := range sc.Breakdown {
			effects = append(effects, row.Label+" "+row.Effect)
		}
		ranking[i] = dto.RankedJob{Rank: i + 1, JobID: id, Title: sc.Title, Company: sc.Company, Score: sc.Score, Effects: effects}
	}
	return ranking
}

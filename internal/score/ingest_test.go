package score_test

import (
	"context"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/score"
)

type stubScorer struct {
	result      score.SuitabilityResult
	lastModelID string
}

func (s *stubScorer) Score(_ context.Context, _ dto.Job, _ dto.SearchConfig, modelID string) (score.SuitabilityResult, error) {
	s.lastModelID = modelID
	return s.result, nil
}

func (s *stubScorer) ScoreBatch(_ context.Context, jobs []dto.Job, _ dto.SearchConfig, modelID string) ([]score.SuitabilityResult, error) {
	s.lastModelID = modelID
	results := make([]score.SuitabilityResult, len(jobs))
	for i := range results {
		results[i] = s.result
	}
	return results, nil
}


type stubScoreWriter struct {
	lastJobID     string
	lastScore     int
	lastReason    string
	lastMatch     []string
	lastMiss      []string
	err           error
	skippedJobID  string
	skippedCalled bool
}

func (w *stubScoreWriter) UpsertJobScoreSuitability(_ context.Context, jobID, _ string, sc int, reasoning string, matched, missing []string) error {
	w.lastJobID = jobID
	w.lastScore = sc
	w.lastReason = reasoning
	w.lastMatch = matched
	w.lastMiss = missing
	return w.err
}

func (w *stubScoreWriter) UpsertJobScoreSkipped(_ context.Context, jobID, _ string) error {
	w.skippedJobID = jobID
	w.skippedCalled = true
	return w.err
}

type stubConfigReader struct {
	cfg dto.SearchConfig
	err error
}

func (r *stubConfigReader) GetSearchConfig(_ context.Context, _ string) (dto.SearchConfig, error) {
	return r.cfg, r.err
}

type stubAIPrefsReader struct {
	prefs dto.UserAIPrefs
	err   error
}

func (r *stubAIPrefsReader) GetUserAIPrefs(_ context.Context, _ string) (dto.UserAIPrefs, error) {
	return r.prefs, r.err
}

func TestIngestScorer_RelevanceGate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		relevanceScore int
		cutoff         int
		wantSkipped    bool
		wantClaudeCall bool
	}{
		{name: "below cutoff skips claude", relevanceScore: 30, cutoff: 50, wantSkipped: true, wantClaudeCall: false},
		{name: "at cutoff scores normally", relevanceScore: 50, cutoff: 50, wantSkipped: false, wantClaudeCall: true},
		{name: "cutoff=0 disables gate", relevanceScore: 0, cutoff: 0, wantSkipped: false, wantClaudeCall: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sc := &stubScorer{result: score.SuitabilityResult{Score: 80}}
			writer := &stubScoreWriter{}
			cfgReader := &stubConfigReader{cfg: dto.SearchConfig{
				SuitabilityRubric: "be good",
				RelevanceCutoff:   tc.cutoff,
			}}
			job := dto.Job{ID: "job-1", URL: "https://example.com/job", Title: "Engineer", RelevanceScore: &tc.relevanceScore}

			is := score.NewIngestScorer(sc, writer, cfgReader, nil)
			is.ScoreAndSaveBatch(context.Background(), []dto.Job{job}, "user-1")

			if tc.wantSkipped && !writer.skippedCalled {
				t.Error("UpsertJobScoreSkipped was not called")
			}
			if !tc.wantSkipped && writer.skippedCalled {
				t.Error("UpsertJobScoreSkipped was called unexpectedly")
			}
			if tc.wantClaudeCall && sc.lastModelID == "" {
				t.Error("scorer was not called; expected Claude call")
			}
			if !tc.wantClaudeCall && sc.lastModelID != "" {
				t.Error("scorer was called; expected no Claude call")
			}
		})
	}
}

func TestIngestScorer_ModelIDFromPrefs(t *testing.T) {
	t.Parallel()

	sc := &stubScorer{result: score.SuitabilityResult{Score: 70}}
	writer := &stubScoreWriter{}
	cfgReader := &stubConfigReader{cfg: dto.SearchConfig{SuitabilityRubric: "be good"}}
	aiPrefs := &stubAIPrefsReader{prefs: dto.UserAIPrefs{SuitabilityModel: "claude-sonnet-4-6"}}

	is := score.NewIngestScorer(sc, writer, cfgReader, aiPrefs)
	is.ScoreAndSaveBatch(context.Background(), []dto.Job{{ID: "job-m", URL: "https://example.com/job3", Title: "Engineer"}}, "user-1")

	if sc.lastModelID != "claude-sonnet-4-6" {
		t.Errorf("model passed to scorer = %q; want %q", sc.lastModelID, "claude-sonnet-4-6")
	}
}

func TestIngestScorer_ModelIDDefaultWhenNoPrefs(t *testing.T) {
	t.Parallel()

	sc := &stubScorer{result: score.SuitabilityResult{Score: 70}}
	writer := &stubScoreWriter{}
	cfgReader := &stubConfigReader{cfg: dto.SearchConfig{SuitabilityRubric: "be good"}}
	aiPrefs := &stubAIPrefsReader{err: providers.ErrNotFound}

	is := score.NewIngestScorer(sc, writer, cfgReader, aiPrefs)
	is.ScoreAndSaveBatch(context.Background(), []dto.Job{{ID: "job-d", URL: "https://example.com/job4", Title: "Engineer"}}, "user-1")

	if sc.lastModelID != score.DefaultSuitabilityModel {
		t.Errorf("model passed to scorer = %q; want default haiku", sc.lastModelID)
	}
}

func TestIngestScorer_ScoreAndSaveBatch(t *testing.T) {
	t.Parallel()

	jobs := []dto.Job{
		{ID: "job-1", URL: "https://example.com/1", Title: "Engineer A"},
		{ID: "job-2", URL: "https://example.com/2", Title: "Engineer B"},
	}

	writer := &stubScoreWriter{}
	cfgReader := &stubConfigReader{cfg: dto.SearchConfig{SuitabilityRubric: "be good"}}
	sc := &stubScorer{result: score.SuitabilityResult{
		Score: 80, Matched: []string{"Go"}, Missing: []string{"K8s"}, Rationale: "Good match.",
	}}

	is := score.NewIngestScorer(sc, writer, cfgReader, nil)
	is.ScoreAndSaveBatch(context.Background(), jobs, "user-1")

	// Both jobs should be written; last write is job-2.
	if writer.lastJobID != "job-2" {
		t.Errorf("last written job ID = %q; want %q", writer.lastJobID, "job-2")
	}
	if writer.lastScore != 80 {
		t.Errorf("last written score = %d; want 80", writer.lastScore)
	}
}

func TestIngestScorer_ScoreAndSaveBatch_RelevanceGate(t *testing.T) {
	t.Parallel()

	cutoff := 50
	belowScore := 30
	aboveScore := 70

	jobs := []dto.Job{
		{ID: "job-skip", URL: "https://example.com/skip", Title: "Skip", RelevanceScore: &belowScore},
		{ID: "job-score", URL: "https://example.com/score", Title: "Score", RelevanceScore: &aboveScore},
	}

	writer := &stubScoreWriter{}
	cfgReader := &stubConfigReader{cfg: dto.SearchConfig{
		SuitabilityRubric: "be good",
		RelevanceCutoff:   cutoff,
	}}
	sc := &stubScorer{result: score.SuitabilityResult{Score: 75}}

	is := score.NewIngestScorer(sc, writer, cfgReader, nil)
	is.ScoreAndSaveBatch(context.Background(), jobs, "user-1")

	if !writer.skippedCalled {
		t.Error("UpsertJobScoreSkipped was not called for below-cutoff job")
	}
	if writer.skippedJobID != "job-skip" {
		t.Errorf("skipped job ID = %q; want %q", writer.skippedJobID, "job-skip")
	}
	// job-score should be written via batch
	if writer.lastJobID != "job-score" {
		t.Errorf("last written job ID = %q; want %q", writer.lastJobID, "job-score")
	}
}

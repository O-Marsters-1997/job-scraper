package score_test

import (
	"context"
	"errors"
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

type errScorer struct{}

func (e *errScorer) Score(_ context.Context, _ dto.Job, _ dto.SearchConfig, _ string) (score.SuitabilityResult, error) {
	return score.SuitabilityResult{}, errors.New("api error")
}

type stubScoreWriter struct {
	lastJobID  string
	lastScore  int
	lastReason string
	lastMatch  []string
	lastMiss   []string
	err        error
}

func (w *stubScoreWriter) UpsertJobScoreSuitability(_ context.Context, jobID, _ string, sc int, reasoning string, matched, missing []string) error {
	w.lastJobID = jobID
	w.lastScore = sc
	w.lastReason = reasoning
	w.lastMatch = matched
	w.lastMiss = missing
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

func TestIngestScorer_ScoreAndSave(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		scorer        score.SuitabilityScorer
		cfgErr        error
		wantWritten   bool
		wantScore     int
		wantReasoning string
		wantMatched   []string
		wantMissing   []string
	}{
		{
			name: "writes score on success",
			scorer: &stubScorer{result: score.SuitabilityResult{
				Score:     75,
				Matched:   []string{"Go", "Postgres"},
				Missing:   []string{"Kubernetes"},
				Rationale: "Strong backend match.",
			}},
			wantWritten: true,
			wantScore:   75,
		},
		{
			name:        "config read failure — no write",
			scorer:      &stubScorer{result: score.SuitabilityResult{Score: 50}},
			cfgErr:      errors.New("db down"),
			wantWritten: false,
		},
		{
			name:        "scorer error — no write",
			scorer:      &errScorer{},
			wantWritten: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			writer := &stubScoreWriter{}
			cfgReader := &stubConfigReader{
				cfg: dto.SearchConfig{SuitabilityRubric: "be good"},
				err: tt.cfgErr,
			}

			is := score.NewIngestScorer(tt.scorer, writer, cfgReader, "user-1")
			is.ScoreAndSave(context.Background(), dto.Job{ID: "job-abc", URL: "https://example.com/job", Title: "Engineer"})

			if tt.wantWritten {
				if writer.last.JobID != "job-abc" {
					t.Errorf("written job ID = %q; want %q", writer.last.JobID, "job-abc")
				}
				if writer.last.Score != tt.wantScore {
					t.Errorf("written score = %d; want %d", writer.last.Score, tt.wantScore)
				}
				if writer.last.Reasoning != tt.wantReasoning {
					t.Errorf("reasoning = %q; want %q", writer.last.Reasoning, tt.wantReasoning)
				}
				if len(writer.last.Matched) != len(tt.wantMatched) {
					t.Errorf("matched len = %d; want %d", len(writer.last.Matched), len(tt.wantMatched))
				}
				if len(writer.last.Missing) != len(tt.wantMissing) {
					t.Errorf("missing len = %d; want %d", len(writer.last.Missing), len(tt.wantMissing))
				}
			} else {
				if writer.last.JobID != "" {
					t.Errorf("expected no write; got job ID %q", writer.last.JobID)
				}
			}
		})
	}
}

func TestIngestScorer_ScoreAndSave_ReasoningFields(t *testing.T) {
	t.Parallel()

	matched := []string{"Go", "Postgres"}
	missing := []string{"Kubernetes"}
	rationale := "Strong backend match."

	writer := &stubScoreWriter{}
	cfgReader := &stubConfigReader{cfg: dto.SearchConfig{SuitabilityRubric: "be good"}}

	is := score.NewIngestScorer(
		&stubScorer{result: score.SuitabilityResult{
			Score:     80,
			Matched:   matched,
			Missing:   missing,
			Rationale: rationale,
		}},
		writer, cfgReader, "user-1",
	)
	is.ScoreAndSave(context.Background(), dto.Job{ID: "job-xyz", URL: "https://example.com/job2", Title: "Go Engineer"})

	if writer.lastReason != rationale {
		t.Errorf("reasoning = %q; want %q", writer.lastReason, rationale)
	}
	if len(writer.lastMatch) != len(matched) {
		t.Errorf("matched len = %d; want %d", len(writer.lastMatch), len(matched))
	}
	if len(writer.lastMiss) != len(missing) {
		t.Errorf("missing len = %d; want %d", len(writer.lastMiss), len(missing))
	}
}

func TestIngestScorer_ModelIDFromPrefs(t *testing.T) {
	t.Parallel()

	sc := &stubScorer{result: score.SuitabilityResult{Score: 70}}
	writer := &stubScoreWriter{}
	cfgReader := &stubConfigReader{cfg: dto.SearchConfig{SuitabilityRubric: "be good"}}
	aiPrefs := &stubAIPrefsReader{prefs: dto.UserAIPrefs{SuitabilityModel: "claude-sonnet-4-6"}}

	is := score.NewIngestScorerWithPrefs(sc, writer, cfgReader, aiPrefs, "user-1")
	is.ScoreAndSave(context.Background(), dto.Job{ID: "job-m", URL: "https://example.com/job3", Title: "Engineer"})

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

	is := score.NewIngestScorerWithPrefs(sc, writer, cfgReader, aiPrefs, "user-1")
	is.ScoreAndSave(context.Background(), dto.Job{ID: "job-d", URL: "https://example.com/job4", Title: "Engineer"})

	if sc.lastModelID != "claude-haiku-4-5-20251001" {
		t.Errorf("model passed to scorer = %q; want default haiku", sc.lastModelID)
	}
}

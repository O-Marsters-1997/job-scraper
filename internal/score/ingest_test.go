package score_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/score"
)

type stubScorer struct {
	result score.SuitabilityResult
}

func (s *stubScorer) Score(_ context.Context, _ dto.Job, _ dto.SearchConfig) (score.SuitabilityResult, error) {
	return s.result, nil
}

type errScorer struct{}

func (e *errScorer) Score(_ context.Context, _ dto.Job, _ dto.SearchConfig) (score.SuitabilityResult, error) {
	return score.SuitabilityResult{}, errors.New("api error")
}

type stubScoreWriter struct {
	last score.SuitabilityScore
	err  error
}

func (w *stubScoreWriter) UpsertJobScoreSuitability(_ context.Context, s score.SuitabilityScore) error {
	w.last = s
	return w.err
}

type stubConfigReader struct {
	cfg dto.SearchConfig
	err error
}

func (r *stubConfigReader) GetSearchConfig(_ context.Context, _ string) (dto.SearchConfig, error) {
	return r.cfg, r.err
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
			name: "writes score and reasoning fields on success",
			scorer: &stubScorer{result: score.SuitabilityResult{
				Score:     75,
				Matched:   []string{"Go", "Postgres"},
				Missing:   []string{"Kubernetes"},
				Rationale: "Strong backend match.",
			}},
			wantWritten:   true,
			wantScore:     75,
			wantReasoning: "Strong backend match.",
			wantMatched:   []string{"Go", "Postgres"},
			wantMissing:   []string{"Kubernetes"},
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

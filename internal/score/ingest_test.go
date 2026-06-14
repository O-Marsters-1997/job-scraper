package score_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/score"
)

type stubScorer struct {
	fixedScore int
}

func (s *stubScorer) Score(_ context.Context, _ dto.Job, _ dto.SearchConfig) (int, score.TokenUsage, error) {
	return s.fixedScore, score.TokenUsage{}, nil
}

type errScorer struct{}

func (e *errScorer) Score(_ context.Context, _ dto.Job, _ dto.SearchConfig) (int, score.TokenUsage, error) {
	return 0, score.TokenUsage{}, errors.New("api error")
}

type stubScoreWriter struct {
	lastJobID string
	lastScore int
	err       error
}

func (w *stubScoreWriter) UpsertJobScoreSuitability(_ context.Context, jobID, _ string, sc int) error {
	w.lastJobID = jobID
	w.lastScore = sc
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
		name        string
		scorer      score.SuitabilityScorer
		cfgErr      error
		wantWritten bool
		wantScore   int
	}{
		{
			name:        "writes score on success",
			scorer:      &stubScorer{fixedScore: 75},
			wantWritten: true,
			wantScore:   75,
		},
		{
			name:        "config read failure — no write",
			scorer:      &stubScorer{fixedScore: 50},
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
				if writer.lastJobID != "job-abc" {
					t.Errorf("written job ID = %q; want %q", writer.lastJobID, "job-abc")
				}
				if writer.lastScore != tt.wantScore {
					t.Errorf("written score = %d; want %d", writer.lastScore, tt.wantScore)
				}
			} else {
				if writer.lastJobID != "" {
					t.Errorf("expected no write; got job ID %q", writer.lastJobID)
				}
			}
		})
	}
}

package score

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestSuitabilityScore(t *testing.T) {
	tests := []struct {
		name          string
		overallRaw    float64
		scaleLen      int
		criteria      []dto.ScoringCriterion
		criteriaProbs map[string]float64
		wantScore     int
		wantConf      float64
	}{
		{
			name:       "no criteria uses overall only",
			overallRaw: 3,
			scaleLen:   5,
			wantScore:  75,
			wantConf:   0.75,
		},
		{
			name:       "required criterion at 0.1 sinks the score",
			overallRaw: 4,
			scaleLen:   5,
			criteria:   []dto.ScoringCriterion{{Key: "go", Required: true}},
			criteriaProbs: map[string]float64{
				"go": 0.1,
			},
			wantScore: 10,
			wantConf:  1,
		},
		{
			name:       "non-required criterion has no floor effect",
			overallRaw: 4,
			scaleLen:   5,
			criteria:   []dto.ScoringCriterion{{Key: "nice_to_have", Required: false}},
			criteriaProbs: map[string]float64{
				"nice_to_have": 0.0,
			},
			wantScore: 100,
			wantConf:  1,
		},
		{
			name:       "scale of two levels",
			overallRaw: 1,
			scaleLen:   2,
			wantScore:  100,
			wantConf:   1,
		},
		{
			name:       "scale of five levels midpoint",
			overallRaw: 2,
			scaleLen:   5,
			wantScore:  50,
			wantConf:   0.5,
		},
		{
			name:       "missing required criterion answer treated as neutral",
			overallRaw: 4,
			scaleLen:   5,
			criteria:   []dto.ScoringCriterion{{Key: "missing", Required: true}},
			wantScore:  100,
			wantConf:   1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotScore, gotConf := suitabilityScore(tt.overallRaw, tt.scaleLen, tt.criteria, tt.criteriaProbs)
			if gotScore != tt.wantScore {
				t.Errorf("score = %d, want %d", gotScore, tt.wantScore)
			}
			if gotConf != tt.wantConf {
				t.Errorf("confidence = %v, want %v", gotConf, tt.wantConf)
			}
		})
	}
}

func TestTruncateRunes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxRunes int
		want     string
	}{
		{"under limit unchanged", "hello", 10, "hello"},
		{"exact limit unchanged", "hello", 5, "hello"},
		{"ascii truncation", "hello world", 5, "hello"},
		{"never splits a multi-byte rune", "hé llo", 2, "hé"},
		{"emoji boundary preserved", "a😀b😀c", 3, "a😀b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateRunes(tt.input, tt.maxRunes)
			if got != tt.want {
				t.Errorf("truncateRunes(%q, %d) = %q, want %q", tt.input, tt.maxRunes, got, tt.want)
			}
		})
	}
}

func TestJevScorer_Score_FallsBackToDefaultScaleWhenUnset(t *testing.T) {
	var captured jevRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&captured)
		resp := jevResponse{
			Model:   JevModel + "-20260917",
			Answers: map[string]jevAnswer{"overall": {Type: "score", Score: 2}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	scorer := &JevScorer{apiKey: "sk-or-test", http: server.Client(), baseURL: server.URL}

	result, err := scorer.Score(context.Background(), dto.Job{}, dto.SearchConfig{}, JevModel)
	if err != nil {
		t.Fatalf("Score: %v", err)
	}
	if want := len(defaultScale); len(captured.Questions["overall"].Criteria.([]any)) != want {
		t.Fatalf("overall question criteria len = %d, want %d", len(captured.Questions["overall"].Criteria.([]any)), want)
	}
	if want := 50; result.Score != want {
		t.Fatalf("score = %d, want %d (2/(5-1) * 100)", result.Score, want)
	}
}

func TestJevScorer_Score_BuildsQuestionsAndState(t *testing.T) {
	var captured jevRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-or-test" {
			t.Errorf("Authorization header = %q", r.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		resp := jevResponse{
			Model: JevModel + "-20260917",
			Answers: map[string]jevAnswer{
				"go_backend": {Type: "noul", Noul: 0.9},
				"overall":    {Type: "score", Score: 3},
			},
			Usage: jevUsage{InputTokens: 100, OutputTokens: 10, Cost: 0.0001},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	scorer := &JevScorer{apiKey: "sk-or-test", http: server.Client(), baseURL: server.URL}

	job := dto.Job{Title: "Backend Engineer", CompanySlug: "acme", Location: "Remote", WorkArrangement: "remote", SalaryRaw: "£80k", Description: "<p>Build things</p>"}
	cfg := dto.SearchConfig{ScoringQuestions: dto.ScoringQuestions{
		Profile: "Go engineer",
		Criteria: []dto.ScoringCriterion{
			{Key: "go_backend", Instructions: "Does it use Go?", True: "yes", False: "no", Required: true},
		},
		Scale: []string{"Not relevant", "Weak", "Possible", "Strong", "Apply today"},
	}}

	result, err := scorer.Score(context.Background(), job, cfg, JevModel)
	if err != nil {
		t.Fatalf("Score: %v", err)
	}

	if len(captured.Questions) != 2 {
		t.Fatalf("questions = %d, want 2 (one noul + one overall)", len(captured.Questions))
	}
	if captured.Questions["go_backend"].Type != "noul" {
		t.Errorf("go_backend question type = %q, want noul", captured.Questions["go_backend"].Type)
	}
	if captured.Questions["overall"].Type != "score" {
		t.Errorf("overall question type = %q, want score", captured.Questions["overall"].Type)
	}
	if captured.State.Job.Title != "Backend Engineer" || captured.State.Job.Location != "Remote" || captured.State.Job.SalaryRaw != "£80k" {
		t.Errorf("job state = %+v", captured.State.Job)
	}
	if !strings.Contains(captured.State.Job.Description, "Build things") || strings.Contains(captured.State.Job.Description, "<p>") {
		t.Errorf("description not HTML-stripped: %q", captured.State.Job.Description)
	}
	if captured.State.Candidate.Profile != "Go engineer" {
		t.Errorf("candidate profile = %q", captured.State.Candidate.Profile)
	}

	if result.Model != JevModel+"-20260917" {
		t.Errorf("result.Model = %q", result.Model)
	}
	if result.Criteria["go_backend"] != 0.9 {
		t.Errorf("result.Criteria[go_backend] = %v, want 0.9", result.Criteria["go_backend"])
	}
	if result.Cost != 0.0001 {
		t.Errorf("result.Cost = %v", result.Cost)
	}
}

func TestClassifyJevStatus(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		header         http.Header
		wantKind       FailureKind
		wantRetryAfter time.Duration
	}{
		{"unauthorized", http.StatusUnauthorized, nil, FailureTerminal, 0},
		{"no credit", http.StatusPaymentRequired, nil, FailureTerminal, 0},
		{"rate limited with retry-after", http.StatusTooManyRequests, http.Header{"Retry-After": []string{"120"}}, FailureRateLimited, 120 * time.Second},
		{"rate limited without retry-after", http.StatusTooManyRequests, nil, FailureRetryable, 0},
		{"server error", http.StatusInternalServerError, nil, FailureRetryable, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tt.status, Header: tt.header, Body: http.NoBody}
			err := classifyJevStatus(resp)

			var scorerErr *ScorerError
			if ok := errors.As(err, &scorerErr); ok {
				if scorerErr.Kind != tt.wantKind {
					t.Fatalf("kind = %v, want %v", scorerErr.Kind, tt.wantKind)
				}
				if scorerErr.RetryAfter != tt.wantRetryAfter {
					t.Fatalf("retryAfter = %v, want %v", scorerErr.RetryAfter, tt.wantRetryAfter)
				}
				return
			}
			if tt.wantKind != FailureRetryable {
				t.Fatalf("expected classified error of kind %v, got plain error %v", tt.wantKind, err)
			}
		})
	}
}

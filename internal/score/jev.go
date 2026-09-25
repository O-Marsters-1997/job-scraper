package score

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

const (
	// Provider is the credential provider key JevScorer scores under.
	Provider = "openrouter"

	// JevModel is the model requested in every decisions call. Jev's
	// response carries a dated snapshot of this (e.g. "-20260917"),
	// which is what gets stored in job_scores.score_model.
	JevModel = "typesafe/jev-1.13"

	decisionsURL        = "https://openrouter.ai/api/alpha/decisions"
	maxDescriptionRunes = 4096 * 4
	overallInstructions = "How well does this job fit what the candidate is looking for?"

	// OverallQuestionKey is reserved: scoringconfig rejects a criterion key
	// that collides with it.
	OverallQuestionKey = "overall"
)

type JevScorer struct {
	apiKey  string
	http    *http.Client
	baseURL string
}

func NewJevScorer(apiKey string) *JevScorer {
	return &JevScorer{apiKey: apiKey, http: &http.Client{Timeout: 60 * time.Second}, baseURL: decisionsURL}
}

type jevQuestion struct {
	Type         string `json:"type"`
	Instructions string `json:"instructions"`
	Criteria     any    `json:"criteria"`
}

type jevJobState struct {
	Title           string `json:"title"`
	Company         string `json:"company"`
	Location        string `json:"location"`
	WorkArrangement string `json:"work_arrangement"`
	SalaryRaw       string `json:"salary_raw"`
	Description     string `json:"description"`
}

type jevCandidateState struct {
	Profile string `json:"profile"`
}

type jevState struct {
	Job       jevJobState       `json:"job"`
	Candidate jevCandidateState `json:"candidate"`
}

type jevRequest struct {
	Model     string                 `json:"model"`
	State     jevState               `json:"state"`
	Questions map[string]jevQuestion `json:"questions"`
}

type jevAnswer struct {
	Type  string  `json:"type"`
	Noul  float64 `json:"noul"`
	Score float64 `json:"score"`
}

type jevUsage struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	Cost         float64 `json:"cost"`
}

type jevResponse struct {
	Model   string               `json:"model"`
	Answers map[string]jevAnswer `json:"answers"`
	Usage   jevUsage             `json:"usage"`
}

var defaultScale = []string{"Not relevant", "Weak", "Possible", "Strong", "Apply today"}

func (j *JevScorer) Score(ctx context.Context, job dto.Job, cfg dto.SearchConfig, modelID string) (SuitabilityResult, error) {
	if modelID == "" {
		modelID = JevModel
	}
	questions := cfg.ScoringQuestions
	if len(questions.Scale) < 2 {
		questions.Scale = defaultScale
	}

	desc := truncateRunes(stripHTML(job.Description), maxDescriptionRunes)
	reqBody := jevRequest{
		Model: modelID,
		State: jevState{
			Job: jevJobState{
				Title:           job.Title,
				Company:         job.CompanySlug,
				Location:        job.Location,
				WorkArrangement: job.WorkArrangement,
				SalaryRaw:       job.SalaryRaw,
				Description:     desc,
			},
			Candidate: jevCandidateState{Profile: questions.Profile},
		},
		Questions: buildQuestions(questions),
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return SuitabilityResult{}, fmt.Errorf("marshal jev request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, j.baseURL, bytes.NewReader(body))
	if err != nil {
		return SuitabilityResult{}, fmt.Errorf("build jev request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+j.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := j.http.Do(httpReq)
	if err != nil {
		return SuitabilityResult{}, fmt.Errorf("jev decisions request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return SuitabilityResult{}, classifyJevStatus(resp)
	}

	var decoded jevResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return SuitabilityResult{}, fmt.Errorf("decode jev response: %w", err)
	}

	overallAnswer, ok := decoded.Answers[OverallQuestionKey]
	if !ok {
		return SuitabilityResult{}, fmt.Errorf("jev decisions: response missing answer for %q", OverallQuestionKey)
	}
	criteriaProbs := make(map[string]float64, len(questions.Criteria))
	for _, c := range questions.Criteria {
		answer, ok := decoded.Answers[c.Key]
		if !ok {
			return SuitabilityResult{}, fmt.Errorf("jev decisions: response missing answer for %q", c.Key)
		}
		criteriaProbs[c.Key] = answer.Noul
	}
	suitability, confidence := suitabilityScore(overallAnswer.Score, len(questions.Scale), questions.Criteria, criteriaProbs)

	slog.Info("suitability scored via jev",
		slog.String("url", job.URL),
		slog.Int("score", suitability),
		slog.Float64("confidence", confidence),
		slog.Int("input_tokens", decoded.Usage.InputTokens),
		slog.Int("output_tokens", decoded.Usage.OutputTokens),
		slog.Float64("cost_usd", decoded.Usage.Cost),
	)

	return SuitabilityResult{
		Score:      suitability,
		Criteria:   criteriaProbs,
		Confidence: confidence,
		Model:      decoded.Model,
		Cost:       decoded.Usage.Cost,
	}, nil
}

func buildQuestions(q dto.ScoringQuestions) map[string]jevQuestion {
	questions := make(map[string]jevQuestion, len(q.Criteria)+1)
	for _, c := range q.Criteria {
		questions[c.Key] = jevQuestion{
			Type:         "noul",
			Instructions: c.Instructions,
			Criteria:     map[string]string{"true": c.True, "false": c.False},
		}
	}
	questions[OverallQuestionKey] = jevQuestion{
		Type:         "score",
		Instructions: overallInstructions,
		Criteria:     q.Scale,
	}
	return questions
}

func suitabilityScore(overallRaw float64, scaleLen int, criteria []dto.ScoringCriterion, criteriaProbs map[string]float64) (score int, confidence float64) {
	confidence = overallRaw / float64(scaleLen-1)
	confidence = math.Max(0, math.Min(1, confidence))

	minRequired := 1.0
	for _, c := range criteria {
		if !c.Required {
			continue
		}
		if p, ok := criteriaProbs[c.Key]; ok && p < minRequired {
			minRequired = p
		}
	}

	raw := int(math.Round(100 * confidence * minRequired))
	return max(0, min(100, raw)), confidence
}

func classifyJevStatus(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<10))
	wrapped := fmt.Errorf("jev decisions: status %d: %s", resp.StatusCode, string(body))
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusPaymentRequired:
		return TerminalScoreError(wrapped)
	case http.StatusTooManyRequests:
		if retryAfter, ok := parseRetryAfter(resp); ok {
			return RateLimitedScoreError(wrapped, retryAfter)
		}
	}
	return wrapped
}

// ponytail: seconds form only (RFC 9110 also allows an HTTP-date); add that
// if OpenRouter sends one.
func parseRetryAfter(resp *http.Response) (time.Duration, bool) {
	if resp == nil {
		return 0, false
	}
	secs, err := strconv.Atoi(resp.Header.Get("Retry-After"))
	if err != nil || secs < 0 {
		return 0, false
	}
	return time.Duration(secs) * time.Second, true
}

func truncateRunes(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes])
}

func stripHTML(s string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(s))
	if err != nil {
		return s
	}
	return strings.TrimSpace(doc.Text())
}

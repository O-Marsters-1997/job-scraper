// Package jev calls Jev's decisions API to answer choice questions about a
// job: probabilities and confidence, keyed by question text.
package jev

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

const (
	decisionsURL = "https://openrouter.ai/api/alpha/decisions"

	// Model is the model requested in every decisions call and the effect
	// model prefix new answer effects are stamped with (schema default).
	Model = "typesafe/jev-1.13"

	// Provider is the user_ai_credentials provider key Jev bills against.
	Provider = "openrouter"

	maxDescriptionRunes = 4096 * 4
)

var choiceOptions = []string{"yes", "no", "not_stated"}

// Client answers Jev choice questions over HTTP.
type Client struct {
	http    *http.Client
	baseURL string
}

func NewClient() *Client {
	return &Client{http: &http.Client{Timeout: 60 * time.Second}, baseURL: decisionsURL}
}

type choiceQuestion struct {
	Type         string   `json:"type"`
	Instructions string   `json:"instructions"`
	Options      []string `json:"options"`
}

type choiceState struct {
	Title           string `json:"title"`
	Company         string `json:"company"`
	Location        string `json:"location"`
	WorkArrangement string `json:"work_arrangement"`
	SalaryRaw       string `json:"salary_raw"`
	Description     string `json:"description"`
}

type choiceRequest struct {
	Model     string                    `json:"model"`
	State     choiceState               `json:"state"`
	Questions map[string]choiceQuestion `json:"questions"`
}

type choiceAnswer struct {
	Probabilities map[string]float64 `json:"probabilities"`
	Confidence    float64            `json:"confidence"`
}

type choiceUsage struct {
	Cost float64 `json:"cost"`
}

type choiceResponse struct {
	Model   string                  `json:"model"`
	Answers map[string]choiceAnswer `json:"answers"`
	Usage   choiceUsage             `json:"usage"`
}

// Answer asks Jev each of questions as a choice between yes, no and
// not_stated, against the given job's state.
func (c *Client) Answer(ctx context.Context, apiKey string, job dto.Job, questions []string) (map[string]dto.Answer, dto.Usage, error) {
	desc := truncateRunes(stripHTML(job.Description), maxDescriptionRunes)

	qs := make(map[string]choiceQuestion, len(questions))
	for _, q := range questions {
		qs[q] = choiceQuestion{Type: "choice", Instructions: q, Options: choiceOptions}
	}

	reqBody := choiceRequest{
		Model: Model,
		State: choiceState{
			Title:           job.Title,
			Company:         job.CompanySlug,
			Location:        job.Location,
			WorkArrangement: job.WorkArrangement,
			SalaryRaw:       job.SalaryRaw,
			Description:     desc,
		},
		Questions: qs,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, dto.Usage{}, fmt.Errorf("marshal jev request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, dto.Usage{}, fmt.Errorf("build jev request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, dto.Usage{}, fmt.Errorf("jev decisions request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, dto.Usage{}, classifyStatus(resp)
	}

	var decoded choiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, dto.Usage{}, fmt.Errorf("decode jev response: %w", err)
	}

	answers := make(map[string]dto.Answer, len(questions))
	for _, q := range questions {
		a, ok := decoded.Answers[q]
		if !ok {
			return nil, dto.Usage{}, fmt.Errorf("jev decisions: response missing answer for %q", q)
		}
		answers[q] = dto.Answer{
			PYes:       a.Probabilities["yes"],
			PNo:        a.Probabilities["no"],
			PNotStated: a.Probabilities["not_stated"],
			Confidence: a.Confidence,
		}
	}

	return answers, dto.Usage{Model: decoded.Model, Cost: decoded.Usage.Cost}, nil
}

func classifyStatus(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<10))
	wrapped := fmt.Errorf("jev decisions: status %d: %s", resp.StatusCode, string(body))
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusPaymentRequired:
		return TerminalError(wrapped)
	case http.StatusTooManyRequests:
		if retryAfter, ok := parseRetryAfter(resp); ok {
			return RateLimitedError(wrapped, retryAfter)
		}
	}
	return wrapped
}

// ponytail: seconds form only (RFC 9110 also allows an HTTP-date); add that
// if OpenRouter sends one.
func parseRetryAfter(resp *http.Response) (time.Duration, bool) {
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

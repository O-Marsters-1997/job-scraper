package jev_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
)

const apiKey = "sk-or-test"

type wireQuestion struct {
	Type         string         `json:"type"`
	Instructions string         `json:"instructions"`
	Criteria     map[string]any `json:"criteria"`
}

type wireState struct {
	Title           string `json:"title"`
	Company         string `json:"company"`
	Location        string `json:"location"`
	WorkArrangement string `json:"work_arrangement"`
	SalaryRaw       string `json:"salary_raw"`
	Description     string `json:"description"`
}

type wireRequest struct {
	Model     string                  `json:"model"`
	State     wireState               `json:"state"`
	Questions map[string]wireQuestion `json:"questions"`
}

type wireAnswer struct {
	Probabilities map[string]float64 `json:"probabilities"`
	Confidence    float64            `json:"confidence"`
}

type wireUsage struct {
	Cost float64 `json:"cost"`
}

type wireResponse struct {
	Model   string                `json:"model"`
	Answers map[string]wireAnswer `json:"answers"`
	Usage   wireUsage             `json:"usage"`
}

func newClient(t *testing.T, h http.HandlerFunc) *jev.Client {
	t.Helper()
	server := httptest.NewServer(h)
	t.Cleanup(server.Close)
	return jev.NewClientAt(server.URL, server.Client())
}

func readRequest(t *testing.T, r *http.Request) (req wireRequest, bodyLen int) {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Errorf("read request body: %v", err)
	}
	if err := json.Unmarshal(body, &req); err != nil {
		t.Errorf("decode request: %v", err)
	}
	return req, len(body)
}

func writeResponse(w http.ResponseWriter, resp wireResponse) {
	resp.Model = "typesafe/jev-1.13-20260917"
	_ = json.NewEncoder(w).Encode(resp)
}

func answerAll(t *testing.T, bodyLens *[]int) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		req, n := readRequest(t, r)
		if bodyLens != nil {
			*bodyLens = append(*bodyLens, n)
		}
		resp := wireResponse{Usage: wireUsage{Cost: 0.001}, Answers: make(map[string]wireAnswer, len(req.Questions))}
		for q := range req.Questions {
			resp.Answers[q] = wireAnswer{Probabilities: map[string]float64{"yes": 1}, Confidence: 1}
		}
		writeResponse(w, resp)
	}
}

func answerGo(t *testing.T, captured *wireRequest) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		req, _ := readRequest(t, r)
		if captured != nil {
			*captured = req
		}
		writeResponse(w, wireResponse{Answers: map[string]wireAnswer{"tech:go": {Probabilities: map[string]float64{"yes": 1}}}})
	}
}

func overBudgetQuestions() []string {
	questions := make([]string, 80)
	for i := range questions {
		questions[i] = fmt.Sprintf("tech:%d:%s", i, strings.Repeat("x", 1000))
	}
	return questions
}

func TestAnswer(t *testing.T) {
	ctx := t.Context()

	t.Run("sends choice questions and decodes answers", func(t *testing.T) {
		var captured wireRequest
		client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("Authorization"); got != "Bearer "+apiKey {
				t.Errorf("Authorization header = %q, want Bearer %s", got, apiKey)
			}
			captured, _ = readRequest(t, r)
			writeResponse(w, wireResponse{
				Answers: map[string]wireAnswer{
					"tech:go":     {Probabilities: map[string]float64{"yes": 0.9, "no": 0.05, "not_stated": 0.05}, Confidence: 0.9},
					"tech:docker": {Probabilities: map[string]float64{"yes": 0.1, "no": 0.2, "not_stated": 0.7}, Confidence: 0.4},
				},
				Usage: wireUsage{Cost: 0.0004},
			})
		})

		job := dto.Job{Title: "Backend Engineer", Description: "<p>Go and Kubernetes</p>"}
		answers, usage, err := client.Answer(ctx, apiKey, job, []string{"tech:go", "tech:docker"})
		if err != nil {
			t.Fatalf("Answer() err = %v", err)
		}

		choice := func(q string) wireQuestion {
			return wireQuestion{Type: "choice", Instructions: q, Criteria: map[string]any{"yes": nil, "no": nil, "not_stated": nil}}
		}
		wantQuestions := map[string]wireQuestion{"tech:go": choice("tech:go"), "tech:docker": choice("tech:docker")}
		if diff := cmp.Diff(wantQuestions, captured.Questions); diff != "" {
			t.Errorf("questions sent (-want +got):\n%s", diff)
		}
		wantAnswers := map[string]dto.Answer{
			"tech:go":     {PYes: 0.9, PNo: 0.05, PNotStated: 0.05, Confidence: 0.9},
			"tech:docker": {PYes: 0.1, PNo: 0.2, PNotStated: 0.7, Confidence: 0.4},
		}
		if diff := cmp.Diff(wantAnswers, answers); diff != "" {
			t.Errorf("answers (-want +got):\n%s", diff)
		}
		wantUsage := dto.Usage{Model: "typesafe/jev-1.13-20260917", Cost: 0.0004}
		if diff := cmp.Diff(wantUsage, usage); diff != "" {
			t.Errorf("usage (-want +got):\n%s", diff)
		}
	})

	t.Run("status errors", func(t *testing.T) {
		tests := []struct {
			name           string
			status         int
			retryAfter     string
			wantTerminal   bool
			wantRetryAfter time.Duration
		}{
			{name: "401 is terminal", status: http.StatusUnauthorized, wantTerminal: true},
			{name: "402 is terminal", status: http.StatusPaymentRequired, wantTerminal: true},
			{name: "429 carries retry-after", status: http.StatusTooManyRequests, retryAfter: "30", wantRetryAfter: 30 * time.Second},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
					if tt.retryAfter != "" {
						w.Header().Set("Retry-After", tt.retryAfter)
					}
					w.WriteHeader(tt.status)
				})

				_, _, err := client.Answer(ctx, apiKey, dto.Job{}, []string{"tech:go"})

				answerErr, ok := errors.AsType[*jev.Error](err)
				if !ok {
					t.Fatalf("Answer() err = %v, want *jev.Error", err)
				}
				if answerErr.Terminal != tt.wantTerminal || answerErr.RetryAfter != tt.wantRetryAfter {
					t.Errorf("Answer() Terminal, RetryAfter = %v, %v, want %v, %v",
						answerErr.Terminal, answerErr.RetryAfter, tt.wantTerminal, tt.wantRetryAfter)
				}
			})
		}
	})

	t.Run("a missing answer is an error", func(t *testing.T) {
		client := newClient(t, func(w http.ResponseWriter, _ *http.Request) {
			writeResponse(w, wireResponse{Answers: map[string]wireAnswer{}})
		})

		if _, _, err := client.Answer(ctx, apiKey, dto.Job{}, []string{"tech:go"}); err == nil {
			t.Error("Answer() err = nil, want an error")
		}
	})

	t.Run("batches questions under budget", func(t *testing.T) {
		questions := overBudgetQuestions()
		var bodyLens []int
		client := newClient(t, answerAll(t, &bodyLens))

		answers, usage, err := client.Answer(ctx, apiKey, dto.Job{}, questions)
		if err != nil {
			t.Fatalf("Answer() err = %v", err)
		}

		if len(bodyLens) < 2 {
			t.Fatalf("requests = %d, want at least 2 batches for a question list over budget", len(bodyLens))
		}
		for i, n := range bodyLens {
			if n >= jev.MaxBatchChars {
				t.Errorf("request %d body = %d chars, want under budget %d", i, n, jev.MaxBatchChars)
			}
		}
		if len(answers) != len(questions) {
			t.Errorf("answers = %d, want %d (one per question across batches)", len(answers), len(questions))
		}
		for _, q := range questions {
			if _, ok := answers[q]; !ok {
				t.Errorf("answers missing question %q", q)
			}
		}
		wantCost := 0.001 * float64(len(bodyLens))
		if usage.Cost < wantCost-1e-9 || usage.Cost > wantCost+1e-9 {
			t.Errorf("usage.Cost = %v, want the sum across batches %v", usage.Cost, wantCost)
		}
	})

	t.Run("a small list makes one request", func(t *testing.T) {
		var requests int
		handler := answerGo(t, nil)
		client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
			requests++
			handler(w, r)
		})

		if _, _, err := client.Answer(ctx, apiKey, dto.Job{}, []string{"tech:go"}); err != nil {
			t.Fatalf("Answer() err = %v", err)
		}
		if requests != 1 {
			t.Errorf("requests = %d, want 1", requests)
		}
	})

	t.Run("a failing batch returns no partial answers", func(t *testing.T) {
		var requests int
		handler := answerAll(t, nil)
		client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
			requests++
			if requests == 2 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			handler(w, r)
		})

		answers, usage, err := client.Answer(ctx, apiKey, dto.Job{}, overBudgetQuestions())

		if err == nil {
			t.Error("Answer() err = nil, want an error")
		}
		if answers != nil || usage != (dto.Usage{}) {
			t.Errorf("Answer() = %+v, %+v, want nil answers and zero usage", answers, usage)
		}
		if requests < 2 {
			t.Errorf("requests = %d, want at least 2 (failure on the 2nd batch)", requests)
		}
	})

	t.Run("truncates the description", func(t *testing.T) {
		repeatToRuneCount := func(s string, runes int) string {
			return string([]rune(strings.Repeat(s, runes))[:runes])
		}
		tests := []struct {
			name  string
			input string
		}{
			{"under limit unchanged", "hello"},
			{"exact limit unchanged", repeatToRuneCount("a", jev.MaxDescriptionRunes)},
			{"ascii truncation", repeatToRuneCount("a", jev.MaxDescriptionRunes+10)},
			{"never splits a multi-byte rune", repeatToRuneCount("é", jev.MaxDescriptionRunes+10)},
			{"emoji boundary preserved", repeatToRuneCount("😀", jev.MaxDescriptionRunes+10)},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var captured wireRequest
				client := newClient(t, answerGo(t, &captured))

				if _, _, err := client.Answer(ctx, apiKey, dto.Job{Description: tt.input}, []string{"tech:go"}); err != nil {
					t.Fatalf("Answer() err = %v", err)
				}

				desc := captured.State.Description
				if !utf8.ValidString(desc) {
					t.Fatalf("description is not valid UTF-8: %q", desc)
				}
				wantRunes := min(utf8.RuneCountInString(tt.input), jev.MaxDescriptionRunes)
				if got := utf8.RuneCountInString(desc); got != wantRunes {
					t.Errorf("description rune count = %d, want %d", got, wantRunes)
				}
			})
		}
	})
}

func TestStateFor(t *testing.T) {
	job := dto.Job{
		Title:           "Engineer",
		CompanySlug:     "acme",
		Location:        "London",
		WorkArrangement: "remote",
		SalaryRaw:       "£50k",
		Description:     "<p>Build <b>things</b></p>",
	}
	want := jev.State{
		Title:           "Engineer",
		Company:         "acme",
		Location:        "London",
		WorkArrangement: "remote",
		SalaryRaw:       "£50k",
		Description:     "Build things",
	}
	if diff := cmp.Diff(want, jev.StateFor(job)); diff != "" {
		t.Errorf("StateFor mismatch (-want +got):\n%s", diff)
	}
}

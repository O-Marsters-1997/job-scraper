package jev_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
)

type wireQuestion struct {
	Type         string   `json:"type"`
	Instructions string   `json:"instructions"`
	Options      []string `json:"options"`
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

func TestClient_Answer_SendsChoiceQuestionsAndDecodesAnswers(t *testing.T) {
	var captured wireRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-or-test" {
			t.Errorf("Authorization header = %q, want Bearer sk-or-test", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		resp := wireResponse{Model: "typesafe/jev-1.13-20260917"}
		resp.Answers = map[string]wireAnswer{
			"tech:go":     {Probabilities: map[string]float64{"yes": 0.9, "no": 0.05, "not_stated": 0.05}, Confidence: 0.9},
			"tech:docker": {Probabilities: map[string]float64{"yes": 0.1, "no": 0.2, "not_stated": 0.7}, Confidence: 0.4},
		}
		resp.Usage.Cost = 0.0004
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := jev.NewClientAt(server.URL, server.Client())

	job := dto.Job{Title: "Backend Engineer", Description: "<p>Go and Kubernetes</p>"}
	questions := []string{"tech:go", "tech:docker"}
	answers, usage, err := client.Answer(context.Background(), "sk-or-test", job, questions)
	if err != nil {
		t.Fatalf("Answer: %v", err)
	}

	wantOptions := []string{"yes", "no", "not_stated"}
	for _, q := range questions {
		question, ok := captured.Questions[q]
		if !ok {
			t.Fatalf("questions missing %q: %+v", q, captured.Questions)
		}
		if question.Type != "choice" {
			t.Errorf("question %q type = %q, want choice", q, question.Type)
		}
		if diff := cmp.Diff(wantOptions, question.Options); diff != "" {
			t.Errorf("question %q options mismatch (-want +got):\n%s", q, diff)
		}
		if question.Instructions != q {
			t.Errorf("question %q instructions = %q, want %q", q, question.Instructions, q)
		}
	}

	wantAnswers := map[string]dto.Answer{
		"tech:go":     {PYes: 0.9, PNo: 0.05, PNotStated: 0.05, Confidence: 0.9},
		"tech:docker": {PYes: 0.1, PNo: 0.2, PNotStated: 0.7, Confidence: 0.4},
	}
	if diff := cmp.Diff(wantAnswers, answers); diff != "" {
		t.Errorf("answers mismatch (-want +got):\n%s", diff)
	}

	wantUsage := dto.Usage{Model: "typesafe/jev-1.13-20260917", Cost: 0.0004}
	if usage != wantUsage {
		t.Errorf("usage = %+v, want %+v", usage, wantUsage)
	}
}

func TestClient_Answer_StatusErrors(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		retryAfter     string
		wantKind       jev.FailureKind
		wantRetryAfter time.Duration
	}{
		{name: "401 is terminal", status: http.StatusUnauthorized, wantKind: jev.FailureTerminal},
		{name: "402 is terminal", status: http.StatusPaymentRequired, wantKind: jev.FailureTerminal},
		{name: "429 carries retry-after", status: http.StatusTooManyRequests, retryAfter: "30", wantKind: jev.FailureRateLimited, wantRetryAfter: 30 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.retryAfter != "" {
					w.Header().Set("Retry-After", tt.retryAfter)
				}
				w.WriteHeader(tt.status)
			}))
			defer server.Close()

			client := jev.NewClientAt(server.URL, server.Client())

			_, _, err := client.Answer(context.Background(), "sk-or-test", dto.Job{}, []string{"tech:go"})
			if err == nil {
				t.Fatal("Answer: want error, got nil")
			}

			var answerErr *jev.Error
			if !errors.As(err, &answerErr) {
				t.Fatalf("error = %v, want *jev.Error", err)
			}
			if answerErr.Kind != tt.wantKind {
				t.Errorf("Kind = %v, want %v", answerErr.Kind, tt.wantKind)
			}
			if answerErr.RetryAfter != tt.wantRetryAfter {
				t.Errorf("RetryAfter = %v, want %v", answerErr.RetryAfter, tt.wantRetryAfter)
			}
		})
	}
}

func TestClient_Answer_MissingAnswerIsAnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := wireResponse{Model: "typesafe/jev-1.13-20260917", Answers: map[string]wireAnswer{}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := jev.NewClientAt(server.URL, server.Client())

	_, _, err := client.Answer(context.Background(), "sk-or-test", dto.Job{}, []string{"tech:go"})
	if err == nil {
		t.Fatal("Answer: want error, got nil")
	}
}

func TestClient_Answer_BatchesQuestionsUnderBudget(t *testing.T) {
	questions := make([]string, 80)
	for i := range questions {
		questions[i] = fmt.Sprintf("tech:%d:%s", i, strings.Repeat("x", 1000))
	}

	var requests [][]byte
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		mu.Lock()
		requests = append(requests, body)
		mu.Unlock()

		var req wireRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		resp := wireResponse{Model: "typesafe/jev-1.13-20260917", Usage: wireUsage{Cost: 0.001}}
		resp.Answers = make(map[string]wireAnswer, len(req.Questions))
		for q := range req.Questions {
			resp.Answers[q] = wireAnswer{Probabilities: map[string]float64{"yes": 1}, Confidence: 1}
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := jev.NewClientAt(server.URL, server.Client())

	answers, usage, err := client.Answer(context.Background(), "sk-or-test", dto.Job{}, questions)
	if err != nil {
		t.Fatalf("Answer: %v", err)
	}

	if len(requests) < 2 {
		t.Fatalf("got %d requests, want at least 2 batches for a question list over budget", len(requests))
	}
	for i, body := range requests {
		if len(body) >= jev.MaxBatchChars {
			t.Errorf("request %d body = %d chars, want under budget %d", i, len(body), jev.MaxBatchChars)
		}
	}

	if len(answers) != len(questions) {
		t.Errorf("got %d answers, want %d (one per question across batches)", len(answers), len(questions))
	}
	for _, q := range questions {
		if _, ok := answers[q]; !ok {
			t.Errorf("answers missing question %q", q)
		}
	}

	wantCost := 0.001 * float64(len(requests))
	if usage.Cost < wantCost-1e-9 || usage.Cost > wantCost+1e-9 {
		t.Errorf("usage.Cost = %v, want sum across batches %v", usage.Cost, wantCost)
	}
}

func TestClient_Answer_SmallListMakesOneRequest(t *testing.T) {
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		resp := wireResponse{Model: "typesafe/jev-1.13-20260917"}
		resp.Answers = map[string]wireAnswer{"tech:go": {Probabilities: map[string]float64{"yes": 1}}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := jev.NewClientAt(server.URL, server.Client())

	if _, _, err := client.Answer(context.Background(), "sk-or-test", dto.Job{}, []string{"tech:go"}); err != nil {
		t.Fatalf("Answer: %v", err)
	}
	if requestCount != 1 {
		t.Errorf("requestCount = %d, want 1", requestCount)
	}
}

func TestClient_Answer_BatchErrorReturnsNoPartialAnswers(t *testing.T) {
	questions := make([]string, 80)
	for i := range questions {
		questions[i] = fmt.Sprintf("tech:%d:%s", i, strings.Repeat("x", 1000))
	}

	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var req wireRequest
		_ = json.Unmarshal(body, &req)
		resp := wireResponse{Model: "typesafe/jev-1.13-20260917"}
		resp.Answers = make(map[string]wireAnswer, len(req.Questions))
		for q := range req.Questions {
			resp.Answers[q] = wireAnswer{Probabilities: map[string]float64{"yes": 1}}
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := jev.NewClientAt(server.URL, server.Client())

	answers, usage, err := client.Answer(context.Background(), "sk-or-test", dto.Job{}, questions)
	if err == nil {
		t.Fatal("Answer: want error, got nil")
	}
	if answers != nil {
		t.Errorf("answers = %+v, want nil on batch error", answers)
	}
	if usage != (dto.Usage{}) {
		t.Errorf("usage = %+v, want zero value on batch error", usage)
	}
	if requestCount < 2 {
		t.Fatalf("got %d requests, want at least 2 (failure on the 2nd batch)", requestCount)
	}
}

func TestClient_Answer_TruncatesDescription(t *testing.T) {
	repeatToRuneCount := func(s string, runes int) string {
		var b strings.Builder
		for utf8.RuneCountInString(b.String()) < runes {
			b.WriteString(s)
		}
		r := []rune(b.String())
		return string(r[:runes])
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
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewDecoder(r.Body).Decode(&captured)
				resp := wireResponse{Model: "typesafe/jev-1.13-20260917"}
				resp.Answers = map[string]wireAnswer{"tech:go": {Probabilities: map[string]float64{"yes": 1}}}
				_ = json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			client := jev.NewClientAt(server.URL, server.Client())

			job := dto.Job{Description: tt.input}
			if _, _, err := client.Answer(context.Background(), "sk-or-test", job, []string{"tech:go"}); err != nil {
				t.Fatalf("Answer: %v", err)
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
}

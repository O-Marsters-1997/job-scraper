package jev

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
)

func TestClient_Answer_SendsChoiceQuestionsAndDecodesAnswers(t *testing.T) {
	var captured choiceRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-or-test" {
			t.Errorf("Authorization header = %q, want Bearer sk-or-test", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		resp := choiceResponse{Model: "typesafe/jev-1.13-20260917"}
		resp.Answers = map[string]choiceAnswer{
			"tech:go":     {Probabilities: map[string]float64{"yes": 0.9, "no": 0.05, "not_stated": 0.05}, Confidence: 0.9},
			"tech:docker": {Probabilities: map[string]float64{"yes": 0.1, "no": 0.2, "not_stated": 0.7}, Confidence: 0.4},
		}
		resp.Usage.Cost = 0.0004
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{http: server.Client(), baseURL: server.URL}

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
		wantKind       FailureKind
		wantRetryAfter time.Duration
	}{
		{name: "401 is terminal", status: http.StatusUnauthorized, wantKind: FailureTerminal},
		{name: "402 is terminal", status: http.StatusPaymentRequired, wantKind: FailureTerminal},
		{name: "429 carries retry-after", status: http.StatusTooManyRequests, retryAfter: "30", wantKind: FailureRateLimited, wantRetryAfter: 30 * time.Second},
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

			client := &Client{http: server.Client(), baseURL: server.URL}

			_, _, err := client.Answer(context.Background(), "sk-or-test", dto.Job{}, []string{"tech:go"})
			if err == nil {
				t.Fatal("Answer: want error, got nil")
			}

			var answerErr *Error
			if !errors.As(err, &answerErr) {
				t.Fatalf("error = %v, want *Error", err)
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
		resp := choiceResponse{Model: "typesafe/jev-1.13-20260917", Answers: map[string]choiceAnswer{}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{http: server.Client(), baseURL: server.URL}

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

		var req choiceRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		resp := choiceResponse{Model: "typesafe/jev-1.13-20260917", Usage: choiceUsage{Cost: 0.001}}
		resp.Answers = make(map[string]choiceAnswer, len(req.Questions))
		for q := range req.Questions {
			resp.Answers[q] = choiceAnswer{Probabilities: map[string]float64{"yes": 1}, Confidence: 1}
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{http: server.Client(), baseURL: server.URL}

	answers, usage, err := client.Answer(context.Background(), "sk-or-test", dto.Job{}, questions)
	if err != nil {
		t.Fatalf("Answer: %v", err)
	}

	if len(requests) < 2 {
		t.Fatalf("got %d requests, want at least 2 batches for a question list over budget", len(requests))
	}
	for i, body := range requests {
		if len(body) >= maxBatchChars {
			t.Errorf("request %d body = %d chars, want under budget %d", i, len(body), maxBatchChars)
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
		resp := choiceResponse{Model: "typesafe/jev-1.13-20260917"}
		resp.Answers = map[string]choiceAnswer{"tech:go": {Probabilities: map[string]float64{"yes": 1}}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{http: server.Client(), baseURL: server.URL}

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
		var req choiceRequest
		_ = json.Unmarshal(body, &req)
		resp := choiceResponse{Model: "typesafe/jev-1.13-20260917"}
		resp.Answers = make(map[string]choiceAnswer, len(req.Questions))
		for q := range req.Questions {
			resp.Answers[q] = choiceAnswer{Probabilities: map[string]float64{"yes": 1}}
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{http: server.Client(), baseURL: server.URL}

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
			if !utf8.ValidString(got) {
				t.Fatalf("truncateRunes produced invalid UTF-8: %q", got)
			}
		})
	}
}

func TestClient_Answer_TruncatesDescriptionOnRuneBoundary(t *testing.T) {
	longDesc := ""
	for utf8.RuneCountInString(longDesc) < maxDescriptionRunes+10 {
		longDesc += "😀"
	}

	var captured choiceRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&captured)
		resp := choiceResponse{Model: "typesafe/jev-1.13-20260917"}
		resp.Answers = map[string]choiceAnswer{"tech:go": {Probabilities: map[string]float64{"yes": 1}}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{http: server.Client(), baseURL: server.URL}

	job := dto.Job{Description: longDesc}
	if _, _, err := client.Answer(context.Background(), "sk-or-test", job, []string{"tech:go"}); err != nil {
		t.Fatalf("Answer: %v", err)
	}

	desc := captured.State.Description
	if !utf8.ValidString(desc) {
		t.Fatalf("truncated description is not valid UTF-8: %q", desc)
	}
	if got := utf8.RuneCountInString(desc); got != maxDescriptionRunes {
		t.Errorf("truncated description rune count = %d, want %d", got, maxDescriptionRunes)
	}
}

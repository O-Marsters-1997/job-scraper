package openrouter_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/openrouter"
)

func TestChat(t *testing.T) {
	t.Run("returns first choice content and cost", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("Authorization"); got != "Bearer k" {
				t.Errorf("Authorization = %q, want Bearer k", got)
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hi"}}],"usage":{"cost":0.5}}`))
		}))
		defer srv.Close()

		got, err := openrouter.Chat(t.Context(), srv.Client(), srv.URL, "k", openrouter.Request{})
		if err != nil || got.Content != "hi" || got.Cost != 0.5 {
			t.Errorf("Chat() = %+v, %v, want hi, 0.5", got, err)
		}
	})

	t.Run("non-200 is a StatusError", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusPaymentRequired)
			_, _ = w.Write([]byte("no credit"))
		}))
		defer srv.Close()

		_, err := openrouter.Chat(t.Context(), srv.Client(), srv.URL, "k", openrouter.Request{})
		var se *openrouter.StatusError
		if !errors.As(err, &se) || se.Code != http.StatusPaymentRequired || se.Body != "no credit" {
			t.Errorf("Chat() error = %v, want StatusError 402 no credit", err)
		}
	})

	t.Run("no choices is an error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"choices":[]}`))
		}))
		defer srv.Close()

		if _, err := openrouter.Chat(t.Context(), srv.Client(), srv.URL, "k", openrouter.Request{}); err == nil {
			t.Error("Chat() error = nil, want error")
		}
	})
}

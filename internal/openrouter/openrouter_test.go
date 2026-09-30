package openrouter_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/ollymarsters/job-scraper/internal/openrouter"
)

func chat(t *testing.T, h http.HandlerFunc) (openrouter.Reply, error) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return openrouter.Chat(t.Context(), srv.Client(), srv.URL, "k", openrouter.Request{})
}

func TestChat(t *testing.T) {
	t.Run("returns first choice content and cost", func(t *testing.T) {
		got, err := chat(t, func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("Authorization"); got != "Bearer k" {
				t.Errorf("Authorization = %q, want Bearer k", got)
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hi"}}],"usage":{"cost":0.5}}`))
		})
		if err != nil {
			t.Fatalf("Chat() err = %v", err)
		}
		if diff := cmp.Diff(openrouter.Reply{Content: "hi", Cost: 0.5}, got); diff != "" {
			t.Errorf("Chat() (-want +got):\n%s", diff)
		}
	})

	t.Run("non-200 is a StatusError", func(t *testing.T) {
		_, err := chat(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusPaymentRequired)
			_, _ = w.Write([]byte("no credit"))
		})
		var se *openrouter.StatusError
		if !errors.As(err, &se) {
			t.Fatalf("Chat() err = %v, want *StatusError", err)
		}
		if diff := cmp.Diff(&openrouter.StatusError{Code: http.StatusPaymentRequired, Body: "no credit"}, se, cmpopts.IgnoreFields(openrouter.StatusError{}, "Header")); diff != "" {
			t.Errorf("StatusError (-want +got):\n%s", diff)
		}
	})

	t.Run("no choices is an error", func(t *testing.T) {
		_, err := chat(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"choices":[]}`))
		})
		if err == nil {
			t.Error("Chat() err = nil, want error")
		}
	})
}

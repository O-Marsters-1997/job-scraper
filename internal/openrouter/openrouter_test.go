package openrouter_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

func stream(t *testing.T, body string, onDelta func(string)) (openrouter.Reply, error) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req["stream"] != true {
			t.Errorf("request = %v (%v), want stream:true", req, err)
		}
		if _, ok := req["response_format"]; ok {
			t.Errorf("request carries response_format %v, want none for a plain-text stream", req["response_format"])
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return openrouter.ChatStream(t.Context(), srv.Client(), srv.URL, "k", openrouter.Request{Model: "m"}, onDelta)
}

func TestChatStream(t *testing.T) {
	t.Run("passes deltas through and reads the cost from the last chunk", func(t *testing.T) {
		var deltas []string
		got, err := stream(t, ": OPENROUTER PROCESSING\n\n"+
			"data: {\"choices\":[{\"delta\":{\"content\":\"Cut \"}}]}\n\n"+
			"data: {\"choices\":[{\"delta\":{\"content\":\"latency\"}}]}\n\n"+
			"data: {\"choices\":[],\"usage\":{\"cost\":0.002}}\n\n"+
			"data: [DONE]\n\n", func(s string) { deltas = append(deltas, s) })

		if err != nil {
			t.Fatalf("ChatStream() err = %v", err)
		}
		if diff := cmp.Diff([]string{"Cut ", "latency"}, deltas); diff != "" {
			t.Errorf("deltas (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(openrouter.Reply{Content: "Cut latency", Cost: 0.002}, got); diff != "" {
			t.Errorf("ChatStream() (-want +got):\n%s", diff)
		}
	})

	t.Run("a mid-stream error chunk fails the call", func(t *testing.T) {
		_, err := stream(t, "data: {\"choices\":[{\"delta\":{\"content\":\"Cut\"}}]}\n\n"+
			"data: {\"error\":{\"code\":502,\"message\":\"upstream hung up\"},\"choices\":[{\"delta\":{\"content\":\"\"},\"finish_reason\":\"error\"}]}\n\n", func(string) {})

		if err == nil || !strings.Contains(err.Error(), "upstream hung up") {
			t.Errorf("ChatStream() err = %v, want the upstream message", err)
		}
	})

	t.Run("a stream that ends without DONE is an error", func(t *testing.T) {
		_, err := stream(t, "data: {\"choices\":[{\"delta\":{\"content\":\"Cut\"}}]}\n\n", func(string) {})

		if err == nil {
			t.Error("ChatStream() err = nil, want an error for a truncated stream")
		}
	})

	t.Run("non-200 is a StatusError before any delta", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		t.Cleanup(srv.Close)

		_, err := openrouter.ChatStream(t.Context(), srv.Client(), srv.URL, "k", openrouter.Request{}, func(string) { t.Error("delta on a failed request") })

		var se *openrouter.StatusError
		if !errors.As(err, &se) || se.Code != http.StatusUnauthorized {
			t.Errorf("ChatStream() err = %v, want a 401 StatusError", err)
		}
	})
}

func TestGetKey(t *testing.T) {
	t.Run("parses a limited key and sends the bearer token", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("Authorization"); got != "Bearer k" {
				t.Errorf("Authorization = %q, want Bearer k", got)
			}
			_, _ = w.Write([]byte(`{"data":{"limit":10,"limit_remaining":4,"limit_reset":"monthly","usage":6,"usage_monthly":6}}`))
		}))
		t.Cleanup(srv.Close)

		got, err := openrouter.GetKey(t.Context(), srv.Client(), srv.URL, "k")
		if err != nil {
			t.Fatalf("GetKey() err = %v", err)
		}
		limit, remaining, reset := 10.0, 4.0, "monthly"
		want := openrouter.KeyInfo{Limit: &limit, LimitRemaining: &remaining, LimitReset: &reset, Usage: 6, UsageMonthly: 6}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("GetKey() (-want +got):\n%s", diff)
		}
	})

	t.Run("null limit leaves Limit nil", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"data":{"limit":null,"limit_remaining":null,"usage":2.5,"usage_monthly":1.5}}`))
		}))
		t.Cleanup(srv.Close)

		got, err := openrouter.GetKey(t.Context(), srv.Client(), srv.URL, "k")
		if err != nil {
			t.Fatalf("GetKey() err = %v", err)
		}
		if got.Limit != nil || got.UsageMonthly != 1.5 {
			t.Errorf("GetKey() = %+v, want nil Limit and UsageMonthly 1.5", got)
		}
	})

	t.Run("non-200 is a StatusError", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		t.Cleanup(srv.Close)

		_, err := openrouter.GetKey(t.Context(), srv.Client(), srv.URL, "k")
		var se *openrouter.StatusError
		if !errors.As(err, &se) || se.Code != http.StatusUnauthorized {
			t.Fatalf("GetKey() err = %v, want 401 StatusError", err)
		}
	})
}

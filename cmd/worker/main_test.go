package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/worker/scraper"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

type detailSourceStub struct{}

func (detailSourceStub) GetDetails(_ context.Context, url string) (dto.Job, error) {
	return dto.Job{Title: "Job", URL: url}, nil
}

func TestSourceDetailHandlerExportsToAPI(t *testing.T) {
	var delivered dto.Job
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ingest" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&delivered); err != nil {
			t.Error(err)
		}
		_, _ = w.Write([]byte(`{"status":"new"}`))
	}))
	defer server.Close()

	jobURL := "https://example.com/job"
	processor := &taskProcessor{detailers: map[string]sources.DetailFetcher{"wis": detailSourceStub{}}, exporter: scraper.NewAPIExporter(server.URL, "token")}
	task := queue.Task{Version: 1, Source: "wis", Kind: queue.DetailTask, URL: jobURL}
	if err := processor.process(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	if delivered.URL != jobURL || delivered.Title != "Job" {
		t.Fatalf("delivered = %+v", delivered)
	}
}

func TestFullFeedCardExportsWithoutDetailFetcher(t *testing.T) {
	var delivered dto.Job
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&delivered); err != nil {
			t.Error(err)
		}
		_, _ = w.Write([]byte(`{"status":"new"}`))
	}))
	defer server.Close()

	processor := &taskProcessor{exporter: scraper.NewAPIExporter(server.URL, "token")}
	card := dto.Job{Source: "remoteok", URL: "https://remoteok.com/jobs/1", Title: "Full feed job", Description: "Already complete"}
	if err := processor.process(context.Background(), queue.Task{Source: "remoteok", Kind: queue.DetailTask, Card: card}); err != nil {
		t.Fatal(err)
	}
	if delivered.Description != card.Description {
		t.Fatalf("delivered = %+v", delivered)
	}
}

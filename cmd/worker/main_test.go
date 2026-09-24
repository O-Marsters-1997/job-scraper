package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/scraper"
	"github.com/ollymarsters/job-scraper/internal/sources"
)

type detailSourceStub struct{}

func (detailSourceStub) CanHandle(string) bool { return true }
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

	job := dto.QueuedJob{URL: "https://example.com/job", Card: dto.Job{Source: "wis"}}
	payload, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	item := queue.SourceItem{ID: job.URL, Source: "wis", Kind: queue.SourceDetail, Payload: payload}
	if err := deliverSourceDetail(context.Background(), item, []sources.DetailFetcher{detailSourceStub{}}, scraper.NewAPIExporter(server.URL, "token")); err != nil {
		t.Fatal(err)
	}
	if delivered.URL != job.URL || delivered.Title != "Job" {
		t.Fatalf("delivered = %+v", delivered)
	}
}

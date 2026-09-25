package registry_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/registry"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGreenhouseFetchPage(t *testing.T) {
	entry, err := registry.Open(dto.SourceTarget{Source: "greenhouse", Value: "acme", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if entry.Name != "greenhouse" || entry.Role != "ats" || entry.RequestGap <= 0 || entry.Details != nil {
		t.Fatalf("entry = %+v", entry)
	}
	src := entry.Source
	requests := 0
	src.(interface{ Client() *http.Client }).Client().Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		if r.URL.String() != "https://boards-api.greenhouse.io/v1/boards/acme/jobs?content=true" {
			t.Fatalf("URL = %s", r.URL)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"jobs":[{"id":1,"title":"Engineer","absolute_url":"https://boards.greenhouse.io/acme/jobs/1"}]}`)), Header: make(http.Header)}, nil
	})
	page, err := src.FetchPage(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if requests != 1 || len(page.Jobs) != 1 || page.NextCursor != "" || page.Jobs[0].CompanySlug != "acme" {
		t.Fatalf("page = %+v, requests = %d", page, requests)
	}
	if _, err := src.FetchPage(context.Background(), "again"); err == nil {
		t.Fatal("nonempty cursor accepted")
	}
	if requests != 1 {
		t.Fatalf("unexpected second request: %d", requests)
	}
}

func TestOpenRejectsInvalidConfiguration(t *testing.T) {
	for _, target := range []dto.SourceTarget{
		{Source: "unknown", Value: "acme", Enabled: true},
		{Source: "greenhouse", Value: "", Enabled: true},
		{Source: "greenhouse", Value: "a/b", Enabled: true},
		{Source: "greenhouse", Value: "..", Enabled: true},
		{Source: "greenhouse", Value: "acme", Enabled: false},
		{Source: "greenhouse", Value: "acme", Enabled: true, Filters: map[string]string{"x": "y"}},
	} {
		if _, err := registry.Open(target); err == nil {
			t.Errorf("accepted %+v", target)
		}
	}
}

package jobsearch_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
)

func newIngestRouter(t *testing.T) http.Handler {
	t.Helper()
	r := chi.NewRouter()
	jobsearch.Build(jobsearchtest.NewDeps(jobsearchtest.NewFakeStore())).PublicRoutes(r)
	return r
}

func ingest(t *testing.T, h http.Handler, body string) jobsearch.IngestResult {
	t.Helper()
	return handlerstest.Do[jobsearch.IngestResult](t, h, http.StatusOK, "POST /ingest", body)
}

func TestRoutesRequireAuth(t *testing.T) {
	r := chi.NewRouter()
	jobsearch.Build(jobsearchtest.NewDeps(jobsearchtest.NewFakeStore())).Routes(r)
	handlerstest.RequiresAuth(t, r,
		"GET /jobs", "GET /jobs/all", "GET /jobs/{id}",
		"GET /sources", "GET /sources/resolve",
		"GET /source-targets", "POST /source-targets", "PATCH /source-targets/{id}",
		"POST /source-targets/{id}/scrape", "DELETE /source-targets/{id}",
		"GET /companies", "GET /companies/{id}", "GET /companies/new", "GET /companies/tracked", "POST /companies",
		"PUT /companies/{id}/tracking", "PUT /companies/{id}/review", "PUT /companies/{id}/exclusion", "POST /companies/{id}/exclusion/undo", "DELETE /companies/{id}/tracking",
		"GET /companies/{id}/boards", "POST /companies/{id}/boards",
	)
}

func TestIngestHandler(t *testing.T) {
	handler := newIngestRouter(t)

	t.Run("rejects a malformed body", func(t *testing.T) {
		handlerstest.RejectsMalformedBody(t, handler, "POST /ingest")
	})

	t.Run("repeated delivery keeps one canonical job", func(t *testing.T) {
		body := `{"title":"Engineer","url":"https://example.com/job2"}`
		first := ingest(t, handler, body)
		if first.Status != "new" {
			t.Fatalf("first delivery status = %q, want new", first.Status)
		}
		second := ingest(t, handler, body)
		if second.Status != "unchanged" || second.JobID != first.JobID {
			t.Errorf("second delivery = %+v, want unchanged with job_id %q", second, first.JobID)
		}
	})

	t.Run("rejects invalid jobs", func(t *testing.T) {
		tests := []struct {
			name string
			body string
		}{
			{"non-http url scheme", `{"Title":"Engineer","URL":"file:///tmp/job"}`},
			{"malformed board id", `{"Title":"Engineer","URL":"https://example.com/jobs/bad-board","BoardID":"not-a-uuid"}`},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := ingest(t, handler, tt.body); got.Status != "rejected" {
					t.Errorf("result = %+v, want rejected", got)
				}
			})
		}
	})

	t.Run("rejects a conflicting board identity", func(t *testing.T) {
		first := `{"Title":"Engineer","URL":"https://example.com/conflict","BoardID":"11111111-1111-1111-1111-111111111111","ProviderPostingID":"posting-1"}`
		conflict := `{"Title":"Engineer","URL":"https://example.com/conflict","BoardID":"22222222-2222-2222-2222-222222222222","ProviderPostingID":"posting-2"}`
		if got := ingest(t, handler, first); got.Status != "new" {
			t.Fatalf("first board status = %q, want new", got.Status)
		}
		if got := ingest(t, handler, conflict); got.Status != "rejected" {
			t.Errorf("conflicting board result = %+v, want rejected", got)
		}
	})
}

func TestIngestBatchHandler(t *testing.T) {
	body := `{"jobs":[{"title":"Engineer","url":"https://example.com/1"},{"title":"","url":"https://example.com/2"}]}`
	view := handlerstest.Do[struct {
		Results []jobsearch.IngestResult `json:"results"`
	}](t, newIngestRouter(t), http.StatusOK, "POST /ingest/batch", body)

	if len(view.Results) != 2 {
		t.Fatalf("results = %+v, want 2", view.Results)
	}
	if view.Results[0].Status != "new" || view.Results[1].Status != "rejected" {
		t.Errorf("statuses = %q, %q, want new, rejected", view.Results[0].Status, view.Results[1].Status)
	}
}

func TestGetCompanyHandlerUnknownIDIs404(t *testing.T) {
	r := chi.NewRouter()
	jobsearch.Build(jobsearchtest.NewDeps(jobsearchtest.NewFakeStore())).Routes(r)
	if w := handlerstest.Serve(t, r, "GET /companies/missing", ""); w.Code != http.StatusNotFound {
		t.Errorf("GET /companies/missing = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestListCompaniesHandlerReadsQueryParams(t *testing.T) {
	st := jobsearchtest.NewFakeStore()
	for _, name := range []string{"Alpha", "Beta"} {
		if _, err := st.UpsertCompany(t.Context(), dto.CompanyUpsert{Slug: strings.ToLower(name), Name: name}); err != nil {
			t.Fatal(err)
		}
	}
	r := chi.NewRouter()
	jobsearch.Build(jobsearchtest.NewDeps(st)).Routes(r)

	page := handlerstest.Do[dto.CompanyPage](t, r, http.StatusOK, "GET /companies?limit=1&q=a", "")
	if len(page.Items) != 1 || page.Items[0].Name != "Alpha" || page.NextCursor == "" {
		t.Errorf("GET /companies?limit=1&q=a = %+v, want Alpha and a next cursor", page)
	}
}

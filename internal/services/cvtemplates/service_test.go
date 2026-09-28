package cvtemplates_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates/cvtemplatestest"
	"github.com/ollymarsters/job-scraper/internal/services/google"
	"github.com/ollymarsters/job-scraper/internal/services/identity/identitytest"
)

var modTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

type failingListTabs struct {
	*identitytest.DocsClient
	docID string
	err   error
}

func (f failingListTabs) ListTabs(ctx context.Context, userID, docID string) ([]google.Tab, error) {
	if docID == f.docID {
		return nil, f.err
	}
	return f.DocsClient.ListTabs(ctx, userID, docID)
}

func TestServiceList_ReturnsAllVisibleTabsAcrossDocs(t *testing.T) {
	gc := identitytest.NewDocsClient().
		WithDoc("docA", []google.Tab{{ID: "t1", Title: "CV 1"}, {ID: "t2", Title: "CV 2"}}, google.FileMeta{Title: "Doc A", ModifiedAt: modTime}).
		WithDoc("docB", []google.Tab{{ID: "t3", Title: "CV 3"}, {ID: "t4", Title: "CV 4"}}, google.FileMeta{Title: "Doc B", ModifiedAt: modTime})
	st := cvtemplatestest.NewFakeStore()
	st.SeedTrackedDoc("u1", "docA")
	st.SeedTrackedDoc("u1", "docB")
	svc := cvtemplates.NewService(gc, st)

	cvs, err := svc.List(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(cvs) != 4 {
		t.Fatalf("List(...) returned %d CVs, want 4", len(cvs))
	}
}

func TestServiceList_DocURLNormalisesTabPrefix(t *testing.T) {
	cases := []struct {
		name  string
		tabID string
		want  string
	}{
		{name: "adds t. prefix", tabID: "0", want: "https://docs.google.com/document/d/docA/edit?tab=t.0"},
		{name: "does not double-prefix", tabID: "t.0", want: "https://docs.google.com/document/d/docA/edit?tab=t.0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gc := identitytest.NewDocsClient().WithDoc("docA", []google.Tab{{ID: tc.tabID, Title: "CV 1"}}, google.FileMeta{Title: "Doc A", ModifiedAt: modTime})
			st := cvtemplatestest.NewFakeStore()
			st.SeedTrackedDoc("u1", "docA")
			svc := cvtemplates.NewService(gc, st)

			cvs, err := svc.List(context.Background(), "u1")
			if err != nil {
				t.Fatal(err)
			}
			if len(cvs) != 1 || cvs[0].DocURL != tc.want {
				t.Fatalf("List(...) = %+v, want a single CV with DocURL %q", cvs, tc.want)
			}
		})
	}
}

func TestServiceList_SkipsInaccessibleDoc(t *testing.T) {
	gc := failingListTabs{
		DocsClient: identitytest.NewDocsClient().WithDoc("docB", []google.Tab{{ID: "t1", Title: "CV 1"}}, google.FileMeta{Title: "Doc B", ModifiedAt: modTime}),
		docID:      "docA",
		err:        errors.New("permission denied"),
	}
	st := cvtemplatestest.NewFakeStore()
	st.SeedTrackedDoc("u1", "docA")
	st.SeedTrackedDoc("u1", "docB")
	svc := cvtemplates.NewService(gc, st)

	cvs, err := svc.List(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(cvs) != 1 || cvs[0].DocID != "docB" {
		t.Fatalf("List(...) = %+v, want a single CV from docB", cvs)
	}
}

func TestServiceList_ReconcileDoesNotUnhideAHiddenTab(t *testing.T) {
	gc := identitytest.NewDocsClient().WithDoc("docA", []google.Tab{{ID: "t1", Title: "CV 1"}}, google.FileMeta{Title: "Doc A", ModifiedAt: modTime})
	st := cvtemplatestest.NewFakeStore()
	tdID := st.SeedTrackedDoc("u1", "docA")
	st.SeedTab(tdID, "t1", "CV 1", false)
	svc := cvtemplates.NewService(gc, st)

	cvs, err := svc.List(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(cvs) != 1 || cvs[0].Visible {
		t.Fatalf("List(...) = %+v, want a single hidden CV", cvs)
	}
	tabs, err := st.ListTabs(context.Background(), tdID)
	if err != nil {
		t.Fatal(err)
	}
	for _, tab := range tabs {
		if tab.TabID == "t1" && tab.Visible {
			t.Error("reconcile must not un-hide a previously hidden tab")
		}
	}
}

func TestServiceList_NotConnectedReturnsError(t *testing.T) {
	gc := identitytest.Disconnected(apperr.Unauthorized("google account not connected"))
	svc := cvtemplates.NewService(gc, cvtemplatestest.NewFakeStore())

	if _, err := svc.List(context.Background(), "u1"); err == nil {
		t.Fatal("expected an error, got nil")
	}
}

type failingExport struct {
	*identitytest.DocsClient
	err error
}

func (f failingExport) ExportPDF(context.Context, string, string, string) (io.ReadCloser, error) {
	return nil, f.err
}

func TestServiceExportPDF_UpstreamFailureMapsTo502(t *testing.T) {
	gc := failingExport{DocsClient: identitytest.NewDocsClient(), err: errors.New("google is down")}
	svc := cvtemplates.NewService(gc, cvtemplatestest.NewFakeStore())

	_, err := svc.ExportPDF(context.Background(), "u1", "docA", "t1")

	status, ok := apperr.StatusFor(err)
	if !ok || status != http.StatusBadGateway {
		t.Fatalf("status = %v, ok = %v, want %d", status, ok, http.StatusBadGateway)
	}
}

func TestServiceExportPDF_PassesTheStreamThroughOnSuccess(t *testing.T) {
	gc := identitytest.NewDocsClient()
	svc := cvtemplates.NewService(gc, cvtemplatestest.NewFakeStore())

	body, err := svc.ExportPDF(context.Background(), "u1", "docA", "t1")
	if err != nil {
		t.Fatal(err)
	}
	if body == nil {
		t.Fatal("want a non-nil body")
	}
}

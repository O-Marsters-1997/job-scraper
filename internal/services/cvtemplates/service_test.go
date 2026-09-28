package cvtemplates_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates/cvtemplatestest"
	"github.com/ollymarsters/job-scraper/internal/services/google"
	"github.com/ollymarsters/job-scraper/internal/services/identity/identitytest"
)

var modTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func seedDoc(t *testing.T, st cvtemplates.Store, userID, docID string) string {
	t.Helper()
	ctx := context.Background()
	if err := st.AddTrackedDoc(ctx, dto.AddTrackedDocInput{UserID: userID, DocID: docID}); err != nil {
		t.Fatal(err)
	}
	docs, err := st.ListTrackedDocs(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range docs {
		if d.DocID == docID {
			return d.ID
		}
	}
	t.Fatalf("doc %q not tracked", docID)
	return ""
}

func tabVisible(t *testing.T, st cvtemplates.Store, trackedDocID, tabID string) bool {
	t.Helper()
	tabs, err := st.ListTabs(context.Background(), trackedDocID)
	if err != nil {
		t.Fatal(err)
	}
	for _, tab := range tabs {
		if tab.TabID == tabID {
			return tab.Visible
		}
	}
	t.Fatalf("tab %q not found", tabID)
	return false
}

func seedTab(t *testing.T, st cvtemplates.Store, userID, docID, tabID string, visible bool) string {
	t.Helper()
	ctx := context.Background()
	id := seedDoc(t, st, userID, docID)
	if err := st.EnsureTabs(ctx, id, []string{tabID}, []string{tabID}); err != nil {
		t.Fatal(err)
	}
	if !visible {
		if err := st.HideTab(ctx, userID, docID, tabID); err != nil {
			t.Fatal(err)
		}
	}
	return id
}

type failingFileMeta struct {
	*identitytest.DocsClient
	err error
}

func (f failingFileMeta) FileMeta(context.Context, string, string) (google.FileMeta, error) {
	return google.FileMeta{}, f.err
}

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

func TestList(t *testing.T) {
	t.Run("returns all visible tabs across docs", func(t *testing.T) {
		gc := identitytest.NewDocsClient().
			WithDoc("docA", []google.Tab{{ID: "t1", Title: "CV 1"}, {ID: "t2", Title: "CV 2"}}, google.FileMeta{Title: "Doc A", ModifiedAt: modTime}).
			WithDoc("docB", []google.Tab{{ID: "t3", Title: "CV 3"}, {ID: "t4", Title: "CV 4"}}, google.FileMeta{Title: "Doc B", ModifiedAt: modTime})
		st := cvtemplatestest.NewFakeStore()
		seedDoc(t, st, "u1", "docA")
		seedDoc(t, st, "u1", "docB")
		svc := cvtemplates.NewService(gc, st)

		cvs, err := svc.List(context.Background(), "u1")
		if err != nil {
			t.Fatal(err)
		}
		if len(cvs) != 4 {
			t.Fatalf("List(...) returned %d CVs, want 4", len(cvs))
		}
	})

	t.Run("doc URL normalises the tab prefix", func(t *testing.T) {
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
				seedDoc(t, st, "u1", "docA")
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
	})

	t.Run("skips an inaccessible doc", func(t *testing.T) {
		gc := failingListTabs{
			DocsClient: identitytest.NewDocsClient().WithDoc("docB", []google.Tab{{ID: "t1", Title: "CV 1"}}, google.FileMeta{Title: "Doc B", ModifiedAt: modTime}),
			docID:      "docA",
			err:        errors.New("permission denied"),
		}
		st := cvtemplatestest.NewFakeStore()
		seedDoc(t, st, "u1", "docA")
		seedDoc(t, st, "u1", "docB")
		svc := cvtemplates.NewService(gc, st)

		cvs, err := svc.List(context.Background(), "u1")
		if err != nil {
			t.Fatal(err)
		}
		if len(cvs) != 1 || cvs[0].DocID != "docB" {
			t.Fatalf("List(...) = %+v, want a single CV from docB", cvs)
		}
	})

	t.Run("reconcile does not unhide a hidden tab", func(t *testing.T) {
		gc := identitytest.NewDocsClient().WithDoc("docA", []google.Tab{{ID: "t1", Title: "CV 1"}}, google.FileMeta{Title: "Doc A", ModifiedAt: modTime})
		st := cvtemplatestest.NewFakeStore()
		tdID := seedDoc(t, st, "u1", "docA")
		if err := st.EnsureTabs(context.Background(), tdID, []string{"t1"}, []string{"CV 1"}); err != nil {
			t.Fatal(err)
		}
		if err := st.HideTab(context.Background(), "u1", "docA", "t1"); err != nil {
			t.Fatal(err)
		}
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
	})

	t.Run("not connected returns an error", func(t *testing.T) {
		gc := identitytest.Disconnected(apperr.Unauthorized("google account not connected"))
		svc := cvtemplates.NewService(gc, cvtemplatestest.NewFakeStore())

		if _, err := svc.List(context.Background(), "u1"); err == nil {
			t.Fatal("expected an error, got nil")
		}
	})
}

type failingExport struct {
	*identitytest.DocsClient
	err error
}

func (f failingExport) ExportPDF(context.Context, string, string, string) (io.ReadCloser, error) {
	return nil, f.err
}

func TestExportPDF(t *testing.T) {
	t.Run("upstream failure maps to 502", func(t *testing.T) {
		gc := failingExport{DocsClient: identitytest.NewDocsClient(), err: errors.New("google is down")}
		svc := cvtemplates.NewService(gc, cvtemplatestest.NewFakeStore())

		_, err := svc.ExportPDF(context.Background(), "u1", "docA", "t1")

		status, ok := apperr.StatusFor(err)
		if !ok || status != http.StatusBadGateway {
			t.Fatalf("status = %v, ok = %v, want %d", status, ok, http.StatusBadGateway)
		}
	})

	t.Run("passes the stream through on success", func(t *testing.T) {
		gc := identitytest.NewDocsClient()
		svc := cvtemplates.NewService(gc, cvtemplatestest.NewFakeStore())

		body, err := svc.ExportPDF(context.Background(), "u1", "docA", "t1")
		if err != nil {
			t.Fatal(err)
		}
		if body == nil {
			t.Fatal("want a non-nil body")
		}
	})
}

func TestAddDoc(t *testing.T) {
	t.Run("tracks a valid doc", func(t *testing.T) {
		gc := identitytest.NewDocsClient().WithDoc("abc1234567890", nil, google.FileMeta{})
		st := cvtemplatestest.NewFakeStore()
		svc := cvtemplates.NewService(gc, st)

		if _, err := svc.AddDoc(context.Background(), "u1", dto.TrackedDocInput{URL: "https://docs.google.com/document/d/abc1234567890/edit"}); err != nil {
			t.Fatal(err)
		}
		if err := st.RemoveTrackedDoc(context.Background(), "u1", "abc1234567890"); err != nil {
			t.Errorf("doc should have been tracked: %v", err)
		}
	})

	t.Run("garbage URL", func(t *testing.T) {
		svc := cvtemplates.NewService(identitytest.NewDocsClient(), cvtemplatestest.NewFakeStore())

		_, err := svc.AddDoc(context.Background(), "u1", dto.TrackedDocInput{URL: "not-a-url"})
		if !errors.Is(err, cvtemplates.ErrInvalidDoc) {
			t.Fatalf("err = %v, want ErrInvalidDoc", err)
		}
	})

	t.Run("inaccessible doc", func(t *testing.T) {
		gc := failingFileMeta{DocsClient: identitytest.NewDocsClient(), err: errors.New("permission denied")}
		svc := cvtemplates.NewService(gc, cvtemplatestest.NewFakeStore())

		_, err := svc.AddDoc(context.Background(), "u1", dto.TrackedDocInput{URL: "https://docs.google.com/document/d/inaccessible123/edit"})
		if !errors.Is(err, cvtemplates.ErrInaccessibleDoc) {
			t.Fatalf("err = %v, want ErrInaccessibleDoc", err)
		}
	})
}

func TestRemoveDoc(t *testing.T) {
	t.Run("removes a tracked doc", func(t *testing.T) {
		st := cvtemplatestest.NewFakeStore()
		seedDoc(t, st, "u1", "docA")
		svc := cvtemplates.NewService(identitytest.NewDocsClient(), st)

		if err := svc.RemoveDoc(context.Background(), "u1", "docA"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("missing doc returns not found", func(t *testing.T) {
		svc := cvtemplates.NewService(identitytest.NewDocsClient(), cvtemplatestest.NewFakeStore())

		err := svc.RemoveDoc(context.Background(), "u1", "docA")
		status, ok := apperr.StatusFor(err)
		if !ok || status != apperr.KindNotFound.Status() {
			t.Errorf("expected a not-found error, got %v", err)
		}
	})
}

func TestHideTab(t *testing.T) {
	t.Run("hides a visible tab", func(t *testing.T) {
		st := cvtemplatestest.NewFakeStore()
		id := seedTab(t, st, "u1", "docA", "t1", true)
		svc := cvtemplates.NewService(identitytest.NewDocsClient(), st)

		if _, err := svc.HideTab(context.Background(), "u1", dto.TabVisibilityInput{DocID: "docA", TabID: "t1"}); err != nil {
			t.Fatal(err)
		}
		if tabVisible(t, st, id, "t1") {
			t.Error("tab should be hidden")
		}
	})

	t.Run("missing tab returns not found", func(t *testing.T) {
		svc := cvtemplates.NewService(identitytest.NewDocsClient(), cvtemplatestest.NewFakeStore())

		_, err := svc.HideTab(context.Background(), "u1", dto.TabVisibilityInput{DocID: "docA", TabID: "t-missing"})

		status, ok := apperr.StatusFor(err)
		if !ok || status != apperr.KindNotFound.Status() {
			t.Fatalf("expected a not-found error, got %v", err)
		}
	})
}

func TestShowTab(t *testing.T) {
	t.Run("shows a hidden tab", func(t *testing.T) {
		st := cvtemplatestest.NewFakeStore()
		id := seedTab(t, st, "u1", "docA", "t1", false)
		svc := cvtemplates.NewService(identitytest.NewDocsClient(), st)

		if _, err := svc.ShowTab(context.Background(), "u1", dto.TabVisibilityInput{DocID: "docA", TabID: "t1"}); err != nil {
			t.Fatal(err)
		}
		if !tabVisible(t, st, id, "t1") {
			t.Error("tab should be visible")
		}
	})

	t.Run("missing tab returns not found", func(t *testing.T) {
		svc := cvtemplates.NewService(identitytest.NewDocsClient(), cvtemplatestest.NewFakeStore())

		_, err := svc.ShowTab(context.Background(), "u1", dto.TabVisibilityInput{DocID: "docA", TabID: "t-missing"})

		status, ok := apperr.StatusFor(err)
		if !ok || status != apperr.KindNotFound.Status() {
			t.Fatalf("expected a not-found error, got %v", err)
		}
	})
}

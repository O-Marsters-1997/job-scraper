package cvtemplates_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers/handlerstest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates/cvtemplatestest"
	"github.com/ollymarsters/job-scraper/internal/services/google"
	"github.com/ollymarsters/job-scraper/internal/services/identity/identitytest"
)

const userID = handlerstest.UserID

var modTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func newService(gc cvtemplates.DocsClient) (*cvtemplates.Service, *cvtemplatestest.FakeStore) {
	st := cvtemplatestest.NewFakeStore()
	return cvtemplates.NewService(gc, st), st
}

func docsWith(docIDs ...string) *identitytest.DocsClient {
	gc := identitytest.NewDocsClient()
	for _, id := range docIDs {
		gc.WithDoc(id, []google.Tab{{ID: "t1", Title: "CV 1"}}, google.FileMeta{Title: "Doc " + id, ModifiedAt: modTime})
	}
	return gc
}

func tabVisible(t *testing.T, st cvtemplates.Store, trackedDocID, tabID string) bool {
	t.Helper()
	tabs, err := st.ListTabs(t.Context(), trackedDocID)
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

func seedTab(t *testing.T, st cvtemplates.Store, docID, tabID string, visible bool) string {
	t.Helper()
	id := cvtemplatestest.Track(t, st, userID, docID)
	if err := st.EnsureTabs(t.Context(), id, []string{tabID}, []string{tabID}); err != nil {
		t.Fatal(err)
	}
	if !visible {
		if err := st.SetTabVisible(t.Context(), userID, docID, tabID, false); err != nil {
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

type failingExport struct {
	*identitytest.DocsClient
	err error
}

func (f failingExport) ExportPDF(context.Context, string, string, string) (io.ReadCloser, error) {
	return nil, f.err
}

func TestList(t *testing.T) {
	t.Run("returns a CV per tab across docs", func(t *testing.T) {
		svc, st := newService(docsWith("docA", "docB"))
		cvtemplatestest.Track(t, st, userID, "docA")
		cvtemplatestest.Track(t, st, userID, "docB")

		got, err := svc.List(t.Context(), userID)
		if err != nil {
			t.Fatalf("List err = %v", err)
		}

		want := []cvtemplates.CV{
			{DocID: "docA", TabID: "t1", Title: "CV 1", SourceDoc: "Doc docA", ModifiedAt: modTime, DocURL: "https://docs.google.com/document/d/docA/edit?tab=t.t1", Visible: true},
			{DocID: "docB", TabID: "t1", Title: "CV 1", SourceDoc: "Doc docB", ModifiedAt: modTime, DocURL: "https://docs.google.com/document/d/docB/edit?tab=t.t1", Visible: true},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("List mismatch (-want +got):\n%s", diff)
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
				gc := identitytest.NewDocsClient().WithDoc("docA", []google.Tab{{ID: tc.tabID, Title: "CV 1"}}, google.FileMeta{})
				svc, st := newService(gc)
				cvtemplatestest.Track(t, st, userID, "docA")

				cvs, err := svc.List(t.Context(), userID)
				if err != nil {
					t.Fatalf("List err = %v", err)
				}
				if len(cvs) != 1 || cvs[0].DocURL != tc.want {
					t.Fatalf("List = %+v, want a single CV with DocURL %q", cvs, tc.want)
				}
			})
		}
	})

	t.Run("skips an inaccessible doc", func(t *testing.T) {
		gc := failingListTabs{DocsClient: docsWith("docB"), docID: "docA", err: errors.New("permission denied")}
		svc, st := newService(gc)
		cvtemplatestest.Track(t, st, userID, "docA")
		cvtemplatestest.Track(t, st, userID, "docB")

		cvs, err := svc.List(t.Context(), userID)
		if err != nil {
			t.Fatalf("List err = %v", err)
		}
		if len(cvs) != 1 || cvs[0].DocID != "docB" {
			t.Fatalf("List = %+v, want a single CV from docB", cvs)
		}
	})

	t.Run("hidden tab stays hidden after reconcile", func(t *testing.T) {
		svc, st := newService(docsWith("docA"))
		tdID := seedTab(t, st, "docA", "t1", false)

		cvs, err := svc.List(t.Context(), userID)
		if err != nil {
			t.Fatalf("List err = %v", err)
		}
		if len(cvs) != 1 || cvs[0].Visible {
			t.Fatalf("List = %+v, want a single hidden CV", cvs)
		}
		if tabVisible(t, st, tdID, "t1") {
			t.Error("tab t1 visible after List, want still hidden")
		}
	})

	t.Run("not connected returns unauthorized", func(t *testing.T) {
		svc, _ := newService(identitytest.Disconnected(apperr.Unauthorized("google account not connected")))

		_, err := svc.List(t.Context(), userID)
		if !apperr.IsKind(err, apperr.KindUnauthorized) {
			t.Fatalf("List err = %v, want an unauthorized error", err)
		}
	})
}

func TestExportPDF(t *testing.T) {
	t.Run("upstream failure maps to upstream error", func(t *testing.T) {
		svc, _ := newService(failingExport{DocsClient: identitytest.NewDocsClient(), err: errors.New("google is down")})

		_, err := svc.ExportPDF(t.Context(), userID, "docA", "t1")
		if !apperr.IsKind(err, apperr.KindUpstream) {
			t.Fatalf("ExportPDF err = %v, want an upstream error", err)
		}
	})

	t.Run("passes the stream through on success", func(t *testing.T) {
		svc, _ := newService(identitytest.NewDocsClient())

		body, err := svc.ExportPDF(t.Context(), userID, "docA", "t1")
		if err != nil {
			t.Fatalf("ExportPDF err = %v", err)
		}
		defer func() { _ = body.Close() }()
		got, err := io.ReadAll(body)
		if err != nil {
			t.Fatalf("ReadAll err = %v", err)
		}
		if string(got) != "pdf-bytes" {
			t.Errorf("ExportPDF body = %q, want %q", got, "pdf-bytes")
		}
	})
}

func TestAddDoc(t *testing.T) {
	const docURL = "https://docs.google.com/document/d/abc1234567890/edit"

	t.Run("tracks a valid doc", func(t *testing.T) {
		svc, st := newService(docsWith("abc1234567890"))

		if _, err := svc.AddDoc(t.Context(), userID, dto.TrackedDocInput{URL: docURL}); err != nil {
			t.Fatalf("AddDoc err = %v", err)
		}
		docs, err := st.ListTrackedDocs(t.Context(), userID)
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 1 || docs[0].DocID != "abc1234567890" {
			t.Errorf("ListTrackedDocs = %+v, want the added doc", docs)
		}
	})

	t.Run("garbage URL", func(t *testing.T) {
		svc, _ := newService(identitytest.NewDocsClient())

		_, err := svc.AddDoc(t.Context(), userID, dto.TrackedDocInput{URL: "not-a-url"})
		if !errors.Is(err, cvtemplates.ErrInvalidDoc) || !apperr.IsKind(err, apperr.KindInvalid) {
			t.Fatalf("AddDoc err = %v, want ErrInvalidDoc", err)
		}
	})

	t.Run("inaccessible doc", func(t *testing.T) {
		svc, _ := newService(failingFileMeta{DocsClient: identitytest.NewDocsClient(), err: errors.New("permission denied")})

		_, err := svc.AddDoc(t.Context(), userID, dto.TrackedDocInput{URL: docURL})
		if !errors.Is(err, cvtemplates.ErrInaccessibleDoc) || !apperr.IsKind(err, apperr.KindNotFound) {
			t.Fatalf("AddDoc err = %v, want ErrInaccessibleDoc", err)
		}
	})
}

func TestHideTab(t *testing.T) {
	in := dto.TabVisibilityInput{DocID: "docA", TabID: "t1"}

	t.Run("hides a visible tab", func(t *testing.T) {
		svc, st := newService(identitytest.NewDocsClient())
		id := seedTab(t, st, "docA", "t1", true)

		if _, err := svc.HideTab(t.Context(), userID, in); err != nil {
			t.Fatalf("HideTab err = %v", err)
		}
		if tabVisible(t, st, id, "t1") {
			t.Error("tab t1 visible after HideTab, want hidden")
		}
	})

	t.Run("missing tab returns not found", func(t *testing.T) {
		svc, _ := newService(identitytest.NewDocsClient())

		_, err := svc.HideTab(t.Context(), userID, in)
		if !apperr.IsKind(err, apperr.KindNotFound) {
			t.Fatalf("HideTab err = %v, want a not-found error", err)
		}
	})
}

func TestShowTab(t *testing.T) {
	in := dto.TabVisibilityInput{DocID: "docA", TabID: "t1"}

	t.Run("shows a hidden tab", func(t *testing.T) {
		svc, st := newService(identitytest.NewDocsClient())
		id := seedTab(t, st, "docA", "t1", false)

		if _, err := svc.ShowTab(t.Context(), userID, in); err != nil {
			t.Fatalf("ShowTab err = %v", err)
		}
		if !tabVisible(t, st, id, "t1") {
			t.Error("tab t1 hidden after ShowTab, want visible")
		}
	})

	t.Run("missing tab returns not found", func(t *testing.T) {
		svc, _ := newService(identitytest.NewDocsClient())

		_, err := svc.ShowTab(t.Context(), userID, in)
		if !apperr.IsKind(err, apperr.KindNotFound) {
			t.Fatalf("ShowTab err = %v, want a not-found error", err)
		}
	})
}

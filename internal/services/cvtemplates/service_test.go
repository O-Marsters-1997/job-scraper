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
	"github.com/ollymarsters/job-scraper/internal/services/google"
)

type fakeDocsClient struct {
	connectErr error
	tabs       map[string][]google.Tab
	meta       map[string]google.FileMeta
	tabErr     map[string]error
	metaErr    map[string]error
	exportErr  error
}

func (f *fakeDocsClient) HTTPClientForUser(_ context.Context, _ string) (*http.Client, error) {
	if f.connectErr != nil {
		return nil, f.connectErr
	}
	return &http.Client{}, nil
}

func (f *fakeDocsClient) ListTabs(_ context.Context, _, docID string) ([]google.Tab, error) {
	if err, ok := f.tabErr[docID]; ok {
		return nil, err
	}
	return f.tabs[docID], nil
}

func (f *fakeDocsClient) FileMeta(_ context.Context, _, docID string) (google.FileMeta, error) {
	if err, ok := f.metaErr[docID]; ok {
		return google.FileMeta{}, err
	}
	return f.meta[docID], nil
}

func (f *fakeDocsClient) ExportPDF(context.Context, string, string, string) (io.ReadCloser, error) {
	if f.exportErr != nil {
		return nil, f.exportErr
	}
	return io.NopCloser(nil), nil
}

type fakeStore struct {
	docs []dto.TrackedDoc
	tabs map[string][]dto.Tab
}

func (f *fakeStore) ListTrackedDocs(_ context.Context, _ string) ([]dto.TrackedDoc, error) {
	return f.docs, nil
}

func (f *fakeStore) EnsureTabs(_ context.Context, trackedDocID string, tabIDs, titles []string) error {
	if f.tabs == nil {
		f.tabs = map[string][]dto.Tab{}
	}
	existing := map[string]dto.Tab{}
	for _, t := range f.tabs[trackedDocID] {
		existing[t.TabID] = t
	}
	for i, id := range tabIDs {
		if t, ok := existing[id]; ok {
			t.Title = titles[i]
			existing[id] = t
			continue
		}
		existing[id] = dto.Tab{TrackedDocID: trackedDocID, TabID: id, Title: titles[i], Visible: true}
	}
	tabs := make([]dto.Tab, 0, len(existing))
	for _, t := range existing {
		tabs = append(tabs, t)
	}
	f.tabs[trackedDocID] = tabs
	return nil
}

func (f *fakeStore) ListTabs(_ context.Context, trackedDocID string) ([]dto.Tab, error) {
	return f.tabs[trackedDocID], nil
}

func TestService_List(t *testing.T) {
	modTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name  string
		setup func(gc *fakeDocsClient, st *fakeStore)
		wantN int
		check func(t *testing.T, cvs []cvtemplates.CV, st *fakeStore)
	}{
		{
			name: "two docs two tabs each",
			setup: func(gc *fakeDocsClient, st *fakeStore) {
				st.docs = []dto.TrackedDoc{{ID: "1", UserID: "u1", DocID: "docA"}, {ID: "2", UserID: "u1", DocID: "docB"}}
				gc.tabs = map[string][]google.Tab{
					"docA": {{ID: "t1", Title: "CV 1"}, {ID: "t2", Title: "CV 2"}},
					"docB": {{ID: "t3", Title: "CV 3"}, {ID: "t4", Title: "CV 4"}},
				}
				gc.meta = map[string]google.FileMeta{
					"docA": {Title: "Doc A", ModifiedAt: modTime},
					"docB": {Title: "Doc B", ModifiedAt: modTime},
				}
			},
			wantN: 4,
			check: func(t *testing.T, cvs []cvtemplates.CV, _ *fakeStore) {
				t.Helper()
				for _, cv := range cvs {
					want := "https://docs.google.com/document/d/" + cv.DocID + "/edit?tab=t." + cv.TabID
					if cv.DocURL != want {
						t.Errorf("DocURL = %q, want %q", cv.DocURL, want)
					}
				}
			},
		},
		{
			name: "DocURL does not double-prefix t. in tabId",
			setup: func(gc *fakeDocsClient, st *fakeStore) {
				st.docs = []dto.TrackedDoc{{ID: "1", UserID: "u1", DocID: "docA"}}
				gc.tabs = map[string][]google.Tab{"docA": {{ID: "t.0", Title: "CV 1"}}}
				gc.meta = map[string]google.FileMeta{"docA": {Title: "Doc A", ModifiedAt: modTime}}
			},
			wantN: 1,
			check: func(t *testing.T, cvs []cvtemplates.CV, _ *fakeStore) {
				t.Helper()
				want := "https://docs.google.com/document/d/docA/edit?tab=t.0"
				if cvs[0].DocURL != want {
					t.Errorf("DocURL = %q, want %q", cvs[0].DocURL, want)
				}
			},
		},
		{
			name: "skips inaccessible doc",
			setup: func(gc *fakeDocsClient, st *fakeStore) {
				st.docs = []dto.TrackedDoc{{ID: "1", UserID: "u1", DocID: "docA"}, {ID: "2", UserID: "u1", DocID: "docB"}}
				gc.tabs = map[string][]google.Tab{"docB": {{ID: "t1", Title: "CV 1"}}}
				gc.meta = map[string]google.FileMeta{"docB": {Title: "Doc B", ModifiedAt: modTime}}
				gc.tabErr = map[string]error{"docA": errors.New("permission denied")}
			},
			wantN: 1,
			check: func(t *testing.T, cvs []cvtemplates.CV, _ *fakeStore) {
				t.Helper()
				if cvs[0].DocID != "docB" {
					t.Errorf("expected CV from docB, got %q", cvs[0].DocID)
				}
			},
		},
		{
			name: "reconcile does not un-hide a hidden tab",
			setup: func(gc *fakeDocsClient, st *fakeStore) {
				st.docs = []dto.TrackedDoc{{ID: "1", UserID: "u1", DocID: "docA"}}
				gc.tabs = map[string][]google.Tab{"docA": {{ID: "t1", Title: "CV 1"}}}
				gc.meta = map[string]google.FileMeta{"docA": {Title: "Doc A", ModifiedAt: modTime}}
				st.tabs = map[string][]dto.Tab{"1": {{TrackedDocID: "1", TabID: "t1", Title: "CV 1", Visible: false}}}
			},
			wantN: 1,
			check: func(t *testing.T, cvs []cvtemplates.CV, st *fakeStore) {
				t.Helper()
				if cvs[0].Visible {
					t.Error("returned CV should have Visible=false for a hidden tab")
				}
				for _, tab := range st.tabs["1"] {
					if tab.TabID == "t1" && tab.Visible {
						t.Error("reconcile must not un-hide a previously hidden tab")
					}
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gc := &fakeDocsClient{tabs: map[string][]google.Tab{}, meta: map[string]google.FileMeta{}, tabErr: map[string]error{}, metaErr: map[string]error{}}
			st := &fakeStore{}
			if tc.setup != nil {
				tc.setup(gc, st)
			}

			svc := cvtemplates.NewService(gc, st)
			cvs, err := svc.List(context.Background(), "u1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(cvs) != tc.wantN {
				t.Fatalf("expected %d CVs, got %d", tc.wantN, len(cvs))
			}
			if tc.check != nil {
				tc.check(t, cvs, st)
			}
		})
	}
}

func TestService_List_NotConnectedReturnsError(t *testing.T) {
	gc := &fakeDocsClient{connectErr: apperr.Unauthorized("google account not connected")}
	svc := cvtemplates.NewService(gc, &fakeStore{})

	_, err := svc.List(context.Background(), "u1")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestService_ExportPDF(t *testing.T) {
	t.Run("returns an upstream error when the export fails", func(t *testing.T) {
		gc := &fakeDocsClient{exportErr: errors.New("google is down")}
		svc := cvtemplates.NewService(gc, &fakeStore{})

		_, err := svc.ExportPDF(context.Background(), "u1", "docA", "t1")

		status, ok := apperr.StatusFor(err)
		if !ok || status != 502 {
			t.Fatalf("status = %v, ok = %v, want 502", status, ok)
		}
	})

	t.Run("passes the stream through on success", func(t *testing.T) {
		gc := &fakeDocsClient{}
		svc := cvtemplates.NewService(gc, &fakeStore{})

		body, err := svc.ExportPDF(context.Background(), "u1", "docA", "t1")
		if err != nil {
			t.Fatal(err)
		}
		if body == nil {
			t.Fatal("want a non-nil body")
		}
	})
}

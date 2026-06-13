package cvtemplates_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/cvtemplates"
	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/google"
)

type mockGoogleClient struct {
	tabs    map[string][]google.Tab
	meta    map[string]google.FileMeta
	tabErr  map[string]error
	metaErr map[string]error
}

func (m *mockGoogleClient) ListTabs(_ context.Context, _, docID string) ([]google.Tab, error) {
	if err, ok := m.tabErr[docID]; ok {
		return nil, err
	}
	return m.tabs[docID], nil
}

func (m *mockGoogleClient) FileMeta(_ context.Context, _, docID string) (google.FileMeta, error) {
	if err, ok := m.metaErr[docID]; ok {
		return google.FileMeta{}, err
	}
	return m.meta[docID], nil
}

type mockTokenChecker struct {
	err error
}

func (m *mockTokenChecker) GetGoogleToken(_ context.Context, _ string) (dto.GoogleToken, error) {
	if m.err != nil {
		return dto.GoogleToken{}, m.err
	}
	return dto.GoogleToken{}, nil
}

type mockStore struct {
	*mockTokenChecker
	*providers.MockTrackedDocProvider
	*providers.MockTabProvider
}

func newStore(tc *mockTokenChecker) *mockStore {
	return &mockStore{
		mockTokenChecker:       tc,
		MockTrackedDocProvider: &providers.MockTrackedDocProvider{},
		MockTabProvider:        &providers.MockTabProvider{},
	}
}

func TestService_List(t *testing.T) {
	modTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name    string
		setup   func(gc *mockGoogleClient, st *mockStore)
		wantN   int
		wantErr string
		check   func(t *testing.T, cvs []cvtemplates.CV, st *mockStore)
	}{
		{
			name: "two docs two tabs each",
			setup: func(gc *mockGoogleClient, st *mockStore) {
				st.Seed(dto.TrackedDoc{ID: "1", UserID: "u1", DocID: "docA"})
				st.Seed(dto.TrackedDoc{ID: "2", UserID: "u1", DocID: "docB"})
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
			check: func(t *testing.T, cvs []cvtemplates.CV, _ *mockStore) {
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
			setup: func(gc *mockGoogleClient, st *mockStore) {
				st.Seed(dto.TrackedDoc{ID: "1", UserID: "u1", DocID: "docA"})
				gc.tabs = map[string][]google.Tab{
					"docA": {{ID: "t.0", Title: "CV 1"}},
				}
				gc.meta = map[string]google.FileMeta{
					"docA": {Title: "Doc A", ModifiedAt: modTime},
				}
			},
			wantN: 1,
			check: func(t *testing.T, cvs []cvtemplates.CV, _ *mockStore) {
				want := "https://docs.google.com/document/d/docA/edit?tab=t.0"
				if cvs[0].DocURL != want {
					t.Errorf("DocURL = %q, want %q", cvs[0].DocURL, want)
				}
			},
		},
		{
			name: "not connected",
			setup: func(_ *mockGoogleClient, st *mockStore) {
				st.mockTokenChecker.err = providers.ErrGoogleTokenNotFound
			},
			wantErr: "not connected",
		},
		{
			name: "skips inaccessible doc",
			setup: func(gc *mockGoogleClient, st *mockStore) {
				st.Seed(dto.TrackedDoc{ID: "1", UserID: "u1", DocID: "docA"})
				st.Seed(dto.TrackedDoc{ID: "2", UserID: "u1", DocID: "docB"})
				gc.tabs = map[string][]google.Tab{
					"docB": {{ID: "t1", Title: "CV 1"}},
				}
				gc.meta = map[string]google.FileMeta{
					"docB": {Title: "Doc B", ModifiedAt: modTime},
				}
				gc.tabErr = map[string]error{"docA": errors.New("permission denied")}
			},
			wantN: 1,
			check: func(t *testing.T, cvs []cvtemplates.CV, _ *mockStore) {
				if cvs[0].DocID != "docB" {
					t.Errorf("expected CV from docB, got %q", cvs[0].DocID)
				}
			},
		},
		{
			name: "hidden tab appears with Visible=false",
			setup: func(gc *mockGoogleClient, st *mockStore) {
				st.Seed(dto.TrackedDoc{ID: "1", UserID: "u1", DocID: "docA"})
				st.Seed(dto.TrackedDoc{ID: "2", UserID: "u1", DocID: "docB"})
				gc.tabs = map[string][]google.Tab{
					"docA": {{ID: "t1", Title: "CV 1"}, {ID: "t2", Title: "CV 2"}},
					"docB": {{ID: "t3", Title: "CV 3"}, {ID: "t4", Title: "CV 4"}},
				}
				gc.meta = map[string]google.FileMeta{
					"docA": {Title: "Doc A", ModifiedAt: modTime},
					"docB": {Title: "Doc B", ModifiedAt: modTime},
				}
				st.MockTabProvider.SeedTab("u1", "docA", "1", "t1", false)
			},
			wantN: 4,
			check: func(t *testing.T, cvs []cvtemplates.CV, _ *mockStore) {
				for _, cv := range cvs {
					if cv.DocID == "docA" && cv.TabID == "t1" && cv.Visible {
						t.Error("seeded-hidden tab t1 should have Visible=false")
					}
				}
			},
		},
		{
			name: "reconcile inserts missing tab rows",
			setup: func(gc *mockGoogleClient, st *mockStore) {
				st.Seed(dto.TrackedDoc{ID: "1", UserID: "u1", DocID: "docA"})
				gc.tabs = map[string][]google.Tab{
					"docA": {{ID: "t1", Title: "CV 1"}, {ID: "t2", Title: "CV 2"}},
				}
				gc.meta = map[string]google.FileMeta{
					"docA": {Title: "Doc A", ModifiedAt: modTime},
				}
			},
			wantN: 2,
			check: func(t *testing.T, _ []cvtemplates.CV, st *mockStore) {
				rows, err := st.MockTabProvider.ListTabs(context.Background(), "1")
				if err != nil {
					t.Fatalf("ListTabs: %v", err)
				}
				if len(rows) != 2 {
					t.Fatalf("expected 2 persisted tab rows after reconcile, got %d", len(rows))
				}
				for _, r := range rows {
					if !r.Visible {
						t.Errorf("tab %q should be visible after reconcile", r.TabID)
					}
				}
			},
		},
		{
			name: "reconcile does not un-hide a hidden tab",
			setup: func(gc *mockGoogleClient, st *mockStore) {
				st.Seed(dto.TrackedDoc{ID: "1", UserID: "u1", DocID: "docA"})
				gc.tabs = map[string][]google.Tab{
					"docA": {{ID: "t1", Title: "CV 1"}},
				}
				gc.meta = map[string]google.FileMeta{
					"docA": {Title: "Doc A", ModifiedAt: modTime},
				}
				st.MockTabProvider.SeedTab("u1", "docA", "1", "t1", false)
			},
			wantN: 1,
			check: func(t *testing.T, cvs []cvtemplates.CV, st *mockStore) {
				rows, _ := st.MockTabProvider.ListTabs(context.Background(), "1")
				for _, r := range rows {
					if r.TabID == "t1" && r.Visible {
						t.Error("reconcile must not un-hide a previously hidden tab")
					}
				}
				if cvs[0].Visible {
					t.Error("returned CV should have Visible=false for a hidden tab")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gc := &mockGoogleClient{
				tabs:    map[string][]google.Tab{},
				meta:    map[string]google.FileMeta{},
				tabErr:  map[string]error{},
				metaErr: map[string]error{},
			}
			st := newStore(&mockTokenChecker{})
			if tc.setup != nil {
				tc.setup(gc, st)
			}

			svc := cvtemplates.NewService(gc, st)
			cvs, err := svc.List(context.Background(), "u1")

			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
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

func TestService_HideTab(t *testing.T) {
	cases := []struct {
		name    string
		setup   func(st *mockStore)
		wantErr error
	}{
		{
			name: "hides a tab",
			setup: func(st *mockStore) {
				st.MockTabProvider.SeedTab("u1", "docA", "1", "t1", true)
			},
		},
		{
			name:    "returns sentinel for missing tab",
			wantErr: providers.ErrTabNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gc := &mockGoogleClient{
				tabs: map[string][]google.Tab{}, meta: map[string]google.FileMeta{},
				tabErr: map[string]error{}, metaErr: map[string]error{},
			}
			st := newStore(&mockTokenChecker{})
			if tc.setup != nil {
				tc.setup(st)
			}

			svc := cvtemplates.NewService(gc, st)
			err := svc.HideTab(context.Background(), "u1", "docA", "t1")

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("expected %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			rows, _ := st.MockTabProvider.ListTabs(context.Background(), "1")
			for _, r := range rows {
				if r.TabID == "t1" && r.Visible {
					t.Error("tab t1 should be hidden after HideTab")
				}
			}
		})
	}
}

func TestService_AddDoc(t *testing.T) {
	cases := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "valid URL", url: "https://docs.google.com/document/d/abc1234567890/edit"},
		{name: "garbage URL", url: "not-a-url", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gc := &mockGoogleClient{
				tabs: map[string][]google.Tab{},
				meta: map[string]google.FileMeta{
					"abc1234567890": {Title: "My CV", ModifiedAt: time.Now()},
				},
				tabErr:  map[string]error{},
				metaErr: map[string]error{},
			}
			st := newStore(&mockTokenChecker{})

			svc := cvtemplates.NewService(gc, st)
			err := svc.AddDoc(context.Background(), "u1", tc.url)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				docs, _ := st.ListTrackedDocs(context.Background(), "u1")
				if len(docs) != 0 {
					t.Errorf("AddTrackedDoc should not have been called, got %+v", docs)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			docs, _ := st.ListTrackedDocs(context.Background(), "u1")
			if len(docs) != 1 || docs[0].DocID != "abc1234567890" {
				t.Errorf("expected tracked doc with docID=abc1234567890, got %+v", docs)
			}
		})
	}
}

func TestService_ShowTab(t *testing.T) {
	cases := []struct {
		name    string
		setup   func(st *mockStore)
		wantErr error
	}{
		{
			name: "restores a hidden tab",
			setup: func(st *mockStore) {
				st.MockTabProvider.SeedTab("u1", "docA", "1", "t1", false)
			},
		},
		{
			name:    "returns sentinel for missing tab",
			wantErr: providers.ErrTabNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gc := &mockGoogleClient{
				tabs: map[string][]google.Tab{}, meta: map[string]google.FileMeta{},
				tabErr: map[string]error{}, metaErr: map[string]error{},
			}
			st := newStore(&mockTokenChecker{})
			if tc.setup != nil {
				tc.setup(st)
			}

			svc := cvtemplates.NewService(gc, st)
			err := svc.ShowTab(context.Background(), "u1", "docA", "t1")

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("expected %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			rows, _ := st.MockTabProvider.ListTabs(context.Background(), "1")
			for _, r := range rows {
				if r.TabID == "t1" && !r.Visible {
					t.Error("tab t1 should be visible after ShowTab")
				}
			}
		})
	}
}

func TestService_RemoveDoc(t *testing.T) {
	cases := []struct {
		name    string
		wantErr error
	}{
		{name: "removes existing doc"},
		{name: "returns sentinel for already-removed doc", wantErr: providers.ErrTrackedDocNotFound},
	}

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gc := &mockGoogleClient{
				tabs: map[string][]google.Tab{}, meta: map[string]google.FileMeta{},
				tabErr: map[string]error{}, metaErr: map[string]error{},
			}
			st := newStore(&mockTokenChecker{})
			st.Seed(dto.TrackedDoc{ID: "1", UserID: "u1", DocID: "docA"})

			svc := cvtemplates.NewService(gc, st)

			// First removal always succeeds; second (i==1) tests the sentinel.
			if i == 0 {
				if err := svc.RemoveDoc(context.Background(), "u1", "docA"); err != nil {
					t.Fatalf("unexpected error on first remove: %v", err)
				}
				return
			}
			// Remove once to clear the doc, then remove again to hit the sentinel.
			_ = svc.RemoveDoc(context.Background(), "u1", "docA")
			err := svc.RemoveDoc(context.Background(), "u1", "docA")
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("expected %v, got %v", tc.wantErr, err)
			}
		})
	}
}

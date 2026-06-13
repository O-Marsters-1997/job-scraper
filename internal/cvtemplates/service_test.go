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

// mockGoogleClient implements the unexported googleClient interface for tests.
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

// mockTokenChecker implements tokenChecker.
type mockTokenChecker struct {
	err error
}

func (m *mockTokenChecker) GetGoogleToken(_ context.Context, _ string) (dto.GoogleToken, error) {
	if m.err != nil {
		return dto.GoogleToken{}, m.err
	}
	return dto.GoogleToken{}, nil
}

func TestService_List(t *testing.T) {
	modTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("list two docs two tabs each", func(t *testing.T) {
		tdp := &providers.MockTrackedDocProvider{}
		tdp.Seed(dto.TrackedDoc{ID: "1", UserID: "u1", DocID: "docA"})
		tdp.Seed(dto.TrackedDoc{ID: "2", UserID: "u1", DocID: "docB"})

		gc := &mockGoogleClient{
			tabs: map[string][]google.Tab{
				"docA": {{ID: "t1", Title: "CV 1"}, {ID: "t2", Title: "CV 2"}},
				"docB": {{ID: "t3", Title: "CV 3"}, {ID: "t4", Title: "CV 4"}},
			},
			meta: map[string]google.FileMeta{
				"docA": {Title: "Doc A", ModifiedAt: modTime},
				"docB": {Title: "Doc B", ModifiedAt: modTime},
			},
			tabErr:  map[string]error{},
			metaErr: map[string]error{},
		}
		tc := &mockTokenChecker{}

		svc := cvtemplates.NewService(gc, tc, tdp)
		cvs, err := svc.List(context.Background(), "u1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cvs) != 4 {
			t.Fatalf("expected 4 CVs, got %d", len(cvs))
		}
		// Check DocURLs are correctly formed.
		for _, cv := range cvs {
			expected := "https://docs.google.com/document/d/" + cv.DocID + "/edit?tab=t." + cv.TabID
			if cv.DocURL != expected {
				t.Errorf("DocURL = %q, want %q", cv.DocURL, expected)
			}
		}
	})

	t.Run("list not connected", func(t *testing.T) {
		tdp := &providers.MockTrackedDocProvider{}
		gc := &mockGoogleClient{
			tabs: map[string][]google.Tab{}, meta: map[string]google.FileMeta{},
			tabErr: map[string]error{}, metaErr: map[string]error{},
		}
		tc := &mockTokenChecker{err: providers.ErrGoogleTokenNotFound}

		svc := cvtemplates.NewService(gc, tc, tdp)
		_, err := svc.List(context.Background(), "u1")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "not connected") {
			t.Errorf("expected 'not connected' in error, got: %v", err)
		}
	})

	t.Run("list skips inaccessible doc", func(t *testing.T) {
		tdp := &providers.MockTrackedDocProvider{}
		tdp.Seed(dto.TrackedDoc{ID: "1", UserID: "u1", DocID: "docA"})
		tdp.Seed(dto.TrackedDoc{ID: "2", UserID: "u1", DocID: "docB"})

		gc := &mockGoogleClient{
			tabs: map[string][]google.Tab{
				"docB": {{ID: "t1", Title: "CV 1"}},
			},
			meta: map[string]google.FileMeta{
				"docB": {Title: "Doc B", ModifiedAt: modTime},
			},
			tabErr:  map[string]error{"docA": errors.New("permission denied")},
			metaErr: map[string]error{},
		}
		tc := &mockTokenChecker{}

		svc := cvtemplates.NewService(gc, tc, tdp)
		cvs, err := svc.List(context.Background(), "u1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cvs) != 1 {
			t.Fatalf("expected 1 CV (from docB only), got %d", len(cvs))
		}
		if cvs[0].DocID != "docB" {
			t.Errorf("expected CV from docB, got %q", cvs[0].DocID)
		}
	})
}

func TestService_AddDoc(t *testing.T) {
	t.Run("AddDoc valid URL", func(t *testing.T) {
		tdp := &providers.MockTrackedDocProvider{}
		gc := &mockGoogleClient{
			tabs: map[string][]google.Tab{},
			meta: map[string]google.FileMeta{
				"abc1234567890": {Title: "My CV", ModifiedAt: time.Now()},
			},
			tabErr:  map[string]error{},
			metaErr: map[string]error{},
		}
		tc := &mockTokenChecker{}

		svc := cvtemplates.NewService(gc, tc, tdp)
		url := "https://docs.google.com/document/d/abc1234567890/edit"
		if err := svc.AddDoc(context.Background(), "u1", url); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		docs, _ := tdp.ListTrackedDocs(context.Background(), "u1")
		if len(docs) != 1 || docs[0].DocID != "abc1234567890" {
			t.Errorf("expected tracked doc with docID=abc1234567890, got %+v", docs)
		}
	})

	t.Run("AddDoc garbage URL", func(t *testing.T) {
		tdp := &providers.MockTrackedDocProvider{}
		gc := &mockGoogleClient{
			tabs: map[string][]google.Tab{}, meta: map[string]google.FileMeta{},
			tabErr: map[string]error{}, metaErr: map[string]error{},
		}
		tc := &mockTokenChecker{}

		svc := cvtemplates.NewService(gc, tc, tdp)
		err := svc.AddDoc(context.Background(), "u1", "not-a-url")
		if err == nil {
			t.Fatal("expected error for invalid URL, got nil")
		}
		docs, _ := tdp.ListTrackedDocs(context.Background(), "u1")
		if len(docs) != 0 {
			t.Errorf("AddTrackedDoc should not have been called, got %+v", docs)
		}
	})
}

func TestService_RemoveDoc(t *testing.T) {
	t.Run("RemoveDoc", func(t *testing.T) {
		tdp := &providers.MockTrackedDocProvider{}
		tdp.Seed(dto.TrackedDoc{ID: "1", UserID: "u1", DocID: "docA"})
		gc := &mockGoogleClient{
			tabs: map[string][]google.Tab{}, meta: map[string]google.FileMeta{},
			tabErr: map[string]error{}, metaErr: map[string]error{},
		}
		tc := &mockTokenChecker{}

		svc := cvtemplates.NewService(gc, tc, tdp)
		if err := svc.RemoveDoc(context.Background(), "u1", "docA"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Second removal should return ErrTrackedDocNotFound.
		err := svc.RemoveDoc(context.Background(), "u1", "docA")
		if !errors.Is(err, providers.ErrTrackedDocNotFound) {
			t.Errorf("expected ErrTrackedDocNotFound, got %v", err)
		}
	})
}

package trackeddocs_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/google"
	"github.com/ollymarsters/job-scraper/internal/services/trackeddocs"
)

type fakeDocsClient struct {
	metaErr map[string]error
}

func (f *fakeDocsClient) FileMeta(_ context.Context, _, docID string) (google.FileMeta, error) {
	if err, ok := f.metaErr[docID]; ok {
		return google.FileMeta{}, err
	}
	return google.FileMeta{Title: "doc", ModifiedAt: time.Now()}, nil
}

type fakeStore struct {
	docs       map[string]bool
	tabVisible map[string]bool
	knownTabs  map[string]bool
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		docs:       map[string]bool{},
		tabVisible: map[string]bool{},
		knownTabs:  map[string]bool{},
	}
}

func (f *fakeStore) AddTrackedDoc(_ context.Context, in dto.AddTrackedDocInput) error {
	f.docs[in.UserID+"/"+in.DocID] = true
	return nil
}

func (f *fakeStore) RemoveTrackedDoc(_ context.Context, userID, docID string) error {
	key := userID + "/" + docID
	if !f.docs[key] {
		return apperr.NotFound("tracked doc not found")
	}
	delete(f.docs, key)
	return nil
}

func (f *fakeStore) seedTab(userID, docID, tabID string, visible bool) {
	key := userID + "/" + docID + "/" + tabID
	f.knownTabs[key] = true
	f.tabVisible[key] = visible
}

func (f *fakeStore) HideTab(_ context.Context, userID, docID, tabID string) error {
	key := userID + "/" + docID + "/" + tabID
	if !f.knownTabs[key] {
		return apperr.NotFound("tab not found")
	}
	f.tabVisible[key] = false
	return nil
}

func (f *fakeStore) ShowTab(_ context.Context, userID, docID, tabID string) error {
	key := userID + "/" + docID + "/" + tabID
	if !f.knownTabs[key] {
		return apperr.NotFound("tab not found")
	}
	f.tabVisible[key] = true
	return nil
}

func TestAddDoc(t *testing.T) {
	cases := []struct {
		name    string
		url     string
		metaErr map[string]error
		check   func(t *testing.T, err error, st *fakeStore)
	}{
		{
			name: "tracks a valid doc",
			url:  "https://docs.google.com/document/d/abc1234567890/edit",
			check: func(t *testing.T, err error, st *fakeStore) {
				t.Helper()
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !st.docs["u1/abc1234567890"] {
					t.Errorf("expected doc abc1234567890 tracked for u1, got %+v", st.docs)
				}
			},
		},
		{
			name:  "garbage URL",
			url:   "not-a-url",
			check: wantAddDocErr(trackeddocs.ErrInvalidDoc),
		},
		{
			name:    "inaccessible doc",
			url:     "https://docs.google.com/document/d/inaccessible123/edit",
			metaErr: map[string]error{"inaccessible123": errors.New("permission denied")},
			check:   wantAddDocErr(trackeddocs.ErrInaccessibleDoc),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gc := &fakeDocsClient{metaErr: tc.metaErr}
			st := newFakeStore()
			svc := trackeddocs.New(gc, st)

			_, err := svc.AddDoc(context.Background(), "u1", dto.TrackedDocInput{URL: tc.url})

			tc.check(t, err, st)
		})
	}
}

func wantAddDocErr(wantErr error) func(t *testing.T, err error, st *fakeStore) {
	return func(t *testing.T, err error, st *fakeStore) {
		t.Helper()
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected %v, got %v", wantErr, err)
		}
		if len(st.docs) != 0 {
			t.Errorf("AddTrackedDoc should not have been called, got %+v", st.docs)
		}
	}
}

func TestRemoveDoc(t *testing.T) {
	gc := &fakeDocsClient{}
	st := newFakeStore()
	st.docs["u1/docA"] = true
	svc := trackeddocs.New(gc, st)

	if err := svc.RemoveDoc(context.Background(), "u1", "docA"); err != nil {
		t.Fatalf("unexpected error on first remove: %v", err)
	}

	err := svc.RemoveDoc(context.Background(), "u1", "docA")
	status, ok := apperr.StatusFor(err)
	if !ok || status != apperr.KindNotFound.Status() {
		t.Errorf("expected a not-found error, got %v", err)
	}
}

func TestHideShowTab(t *testing.T) {
	gc := &fakeDocsClient{}
	st := newFakeStore()
	st.seedTab("u1", "docA", "t1", true)
	svc := trackeddocs.New(gc, st)

	if _, err := svc.HideTab(context.Background(), "u1", dto.TabVisibilityInput{DocID: "docA", TabID: "t1"}); err != nil {
		t.Fatal(err)
	}
	if st.tabVisible["u1/docA/t1"] {
		t.Error("tab should be hidden")
	}

	if _, err := svc.ShowTab(context.Background(), "u1", dto.TabVisibilityInput{DocID: "docA", TabID: "t1"}); err != nil {
		t.Fatal(err)
	}
	if !st.tabVisible["u1/docA/t1"] {
		t.Error("tab should be visible")
	}
}

func TestHideTab_MissingReturnsNotFound(t *testing.T) {
	gc := &fakeDocsClient{}
	svc := trackeddocs.New(gc, newFakeStore())

	_, err := svc.HideTab(context.Background(), "u1", dto.TabVisibilityInput{DocID: "docA", TabID: "t-missing"})

	status, ok := apperr.StatusFor(err)
	if !ok || status != apperr.KindNotFound.Status() {
		t.Fatalf("expected a not-found error, got %v", err)
	}
}

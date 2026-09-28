// Package cvtemplatestest is a map-backed fake of cvtemplates.Store, proven
// against the real store by RunStoreContract (ADR 0012).
package cvtemplatestest

import (
	"context"
	"fmt"
	"sync"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates"
)

type FakeStore struct {
	mu   sync.Mutex
	docs map[string]dto.TrackedDoc
	tabs map[string][]dto.Tab
	seq  int
}

func NewFakeStore() *FakeStore {
	return &FakeStore{docs: map[string]dto.TrackedDoc{}, tabs: map[string][]dto.Tab{}}
}

// SeedTrackedDoc registers a tracked doc directly, standing in for
// trackeddocs.Store.AddTrackedDoc, which cvtemplates.Store doesn't expose.
func (f *FakeStore) SeedTrackedDoc(userID, docID string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	id := fmt.Sprintf("doc-%d", f.seq)
	f.docs[id] = dto.TrackedDoc{ID: id, UserID: userID, DocID: docID}
	return id
}

// SeedTab registers a tab directly at the given visibility, standing in for
// trackeddocs.Store.HideTab/ShowTab, which cvtemplates.Store doesn't expose.
func (f *FakeStore) SeedTab(trackedDocID, tabID, title string, visible bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tabs[trackedDocID] = append(f.tabs[trackedDocID], dto.Tab{
		TrackedDocID: trackedDocID, TabID: tabID, Title: title, Visible: visible,
	})
}

func (f *FakeStore) ListTrackedDocs(_ context.Context, userID string) ([]dto.TrackedDoc, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []dto.TrackedDoc
	for _, d := range f.docs {
		if d.UserID == userID {
			out = append(out, d)
		}
	}
	return out, nil
}

func (f *FakeStore) EnsureTabs(_ context.Context, trackedDocID string, tabIDs, titles []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
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

func (f *FakeStore) ListTabs(_ context.Context, trackedDocID string) ([]dto.Tab, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.tabs[trackedDocID], nil
}

var _ cvtemplates.Store = (*FakeStore)(nil)

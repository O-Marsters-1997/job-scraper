// Package cvtemplatestest is a map-backed fake of cvtemplates.Store, proven
// against the real store by RunStoreContract (ADR 0012).
package cvtemplatestest

import (
	"context"
	"fmt"
	"sync"

	"github.com/ollymarsters/job-scraper/internal/apperr"
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

func (f *FakeStore) docFor(userID, docID string) (dto.TrackedDoc, bool) {
	for _, d := range f.docs {
		if d.UserID == userID && d.DocID == docID {
			return d, true
		}
	}
	return dto.TrackedDoc{}, false
}

func (f *FakeStore) AddTrackedDoc(_ context.Context, in dto.AddTrackedDocInput) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.docFor(in.UserID, in.DocID); ok {
		return nil
	}
	f.seq++
	id := fmt.Sprintf("doc-%d", f.seq)
	f.docs[id] = dto.TrackedDoc{ID: id, UserID: in.UserID, DocID: in.DocID, Kind: in.Kind}
	return nil
}

func (f *FakeStore) RemoveTrackedDoc(_ context.Context, userID, docID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.docFor(userID, docID)
	if !ok {
		return apperr.NotFound("tracked doc not found")
	}
	delete(f.docs, d.ID)
	delete(f.tabs, d.ID)
	return nil
}

func (f *FakeStore) ListTrackedDocs(_ context.Context, userID string, kind dto.TrackedDocKind) ([]dto.TrackedDoc, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []dto.TrackedDoc
	for _, d := range f.docs {
		if d.UserID == userID && d.Kind == kind {
			out = append(out, d)
		}
	}
	return out, nil
}

func (f *FakeStore) EnsureTabs(_ context.Context, trackedDocID string, tabIDs, titles []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	tabs := f.tabs[trackedDocID]
	for i, id := range tabIDs {
		found := false
		for j := range tabs {
			if tabs[j].TabID == id {
				tabs[j].Title = titles[i]
				found = true
			}
		}
		if !found {
			tabs = append(tabs, dto.Tab{TrackedDocID: trackedDocID, TabID: id, Title: titles[i], Visible: true})
		}
	}
	f.tabs[trackedDocID] = tabs
	return nil
}

func (f *FakeStore) ListTabs(_ context.Context, trackedDocID string) ([]dto.Tab, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]dto.Tab(nil), f.tabs[trackedDocID]...), nil
}

func (f *FakeStore) SetTabVisible(_ context.Context, userID, docID, tabID string, visible bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.docFor(userID, docID)
	if !ok {
		return apperr.NotFound("tab not found")
	}
	tabs := f.tabs[d.ID]
	for i := range tabs {
		if tabs[i].TabID == tabID {
			tabs[i].Visible = visible
			tabs[i].IsDefault = tabs[i].IsDefault && visible
			return nil
		}
	}
	return apperr.NotFound("tab not found")
}

func (f *FakeStore) SetTabDefault(_ context.Context, userID, docID, tabID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.docFor(userID, docID)
	if !ok {
		return apperr.NotFound("tab not found")
	}
	target := -1
	for i, tab := range f.tabs[d.ID] {
		if tab.TabID == tabID && tab.Visible {
			target = i
		}
	}
	if target < 0 {
		return apperr.NotFound("tab not found")
	}
	for id, tabs := range f.tabs {
		if f.docs[id].UserID != userID {
			continue
		}
		for i := range tabs {
			tabs[i].IsDefault = id == d.ID && i == target
		}
	}
	return nil
}

var _ cvtemplates.Store = (*FakeStore)(nil)

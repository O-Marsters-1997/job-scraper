// Package trackeddocstest is a map-backed fake of trackeddocs.Store, proven
// against the real store by RunStoreContract (ADR 0012).
package trackeddocstest

import (
	"context"
	"sync"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/trackeddocs"
)

type FakeStore struct {
	mu   sync.Mutex
	docs map[string]bool
	tabs map[string]bool
}

func NewFakeStore() *FakeStore {
	return &FakeStore{docs: map[string]bool{}, tabs: map[string]bool{}}
}

func docKey(userID, docID string) string        { return userID + "/" + docID }
func tabKey(userID, docID, tabID string) string { return userID + "/" + docID + "/" + tabID }

func (f *FakeStore) AddTrackedDoc(_ context.Context, in dto.AddTrackedDocInput) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.docs[docKey(in.UserID, in.DocID)] = true
	return nil
}

func (f *FakeStore) RemoveTrackedDoc(_ context.Context, userID, docID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := docKey(userID, docID)
	if !f.docs[key] {
		return apperr.NotFound("tracked doc not found")
	}
	delete(f.docs, key)
	return nil
}

// SeedTab registers a tab at the given visibility, standing in for
// cvtemplates.Store.EnsureTabs, which trackeddocs.Store doesn't expose.
func (f *FakeStore) SeedTab(userID, docID, tabID string, visible bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tabs[tabKey(userID, docID, tabID)] = visible
}

// Visible reports a seeded tab's current visibility, for a test to assert
// against after HideTab/ShowTab.
func (f *FakeStore) Visible(userID, docID, tabID string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.tabs[tabKey(userID, docID, tabID)]
}

func (f *FakeStore) HideTab(_ context.Context, userID, docID, tabID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := tabKey(userID, docID, tabID)
	if _, ok := f.tabs[key]; !ok {
		return apperr.NotFound("tab not found")
	}
	f.tabs[key] = false
	return nil
}

func (f *FakeStore) ShowTab(_ context.Context, userID, docID, tabID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := tabKey(userID, docID, tabID)
	if _, ok := f.tabs[key]; !ok {
		return apperr.NotFound("tab not found")
	}
	f.tabs[key] = true
	return nil
}

var _ trackeddocs.Store = (*FakeStore)(nil)

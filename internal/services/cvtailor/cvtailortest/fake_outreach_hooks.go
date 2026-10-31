package cvtailortest

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/store"
)

type outreachHook struct {
	text string
	cost float64
}

func (f *FakeStore) SaveOutreachHook(_ context.Context, userID, contactID, text, _ string, cost float64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.subjects[contactID]; !ok || s.userID != userID {
		return store.ErrContactNotFound
	}
	f.hooks[contactID] = outreachHook{text: text, cost: f.hooks[contactID].cost + cost}
	return nil
}

func (f *FakeStore) OutreachHook(_ context.Context, userID, contactID string) (string, float64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.subjects[contactID]; !ok || s.userID != userID {
		return "", 0, store.ErrContactNotFound
	}
	h := f.hooks[contactID]
	return h.text, h.cost, nil
}

package providers

import (
	"context"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type mockTabRow struct {
	userID       string
	docID        string
	trackedDocID string
	tabID        string
	title        string
	visible      bool
}

type MockTabProvider struct {
	tabs []mockTabRow
}

func (m *MockTabProvider) EnsureTabs(_ context.Context, trackedDocID string, tabIDs []string, titles []string) error {
	for i, id := range tabIDs {
		title := ""
		if i < len(titles) {
			title = titles[i]
		}
		for j := range m.tabs {
			if m.tabs[j].trackedDocID == trackedDocID && m.tabs[j].tabID == id {
				m.tabs[j].title = title
				goto next
			}
		}
		m.tabs = append(m.tabs, mockTabRow{
			trackedDocID: trackedDocID,
			tabID:        id,
			title:        title,
			visible:      true,
		})
	next:
	}
	return nil
}

func (m *MockTabProvider) ListTabs(_ context.Context, trackedDocID string) ([]dto.Tab, error) {
	var out []dto.Tab
	for _, t := range m.tabs {
		if t.trackedDocID == trackedDocID {
			out = append(out, dto.Tab{
				TrackedDocID: t.trackedDocID,
				TabID:        t.tabID,
				Title:        t.title,
				Visible:      t.visible,
				CreatedAt:    time.Time{},
			})
		}
	}
	return out, nil
}

func (m *MockTabProvider) HideTab(_ context.Context, userID, docID, tabID string) error {
	for i := range m.tabs {
		t := &m.tabs[i]
		if t.userID == userID && t.docID == docID && t.tabID == tabID {
			t.visible = false
			return nil
		}
	}
	return ErrTabNotFound
}

func (m *MockTabProvider) ShowTab(_ context.Context, userID, docID, tabID string) error {
	for i := range m.tabs {
		t := &m.tabs[i]
		if t.userID == userID && t.docID == docID && t.tabID == tabID {
			t.visible = true
			return nil
		}
	}
	return ErrTabNotFound
}

func (m *MockTabProvider) SeedTab(userID, docID, trackedDocID, tabID string, visible bool) {
	m.tabs = append(m.tabs, mockTabRow{
		userID:       userID,
		docID:        docID,
		trackedDocID: trackedDocID,
		tabID:        tabID,
		visible:      visible,
	})
}

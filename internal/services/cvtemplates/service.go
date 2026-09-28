// Package cvtemplates is the cvtemplates context: tracked CV docs, their
// tabs, and rendering them as a PDF (ADR 0011).
package cvtemplates

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/google"
)

// DocsClient is the narrow Google Docs/Drive surface cvtemplates needs;
// identity's *google.Client satisfies it (ADR 0011).
type DocsClient interface {
	HTTPClientForUser(ctx context.Context, userID string) (*http.Client, error)
	ListTabs(ctx context.Context, userID, docID string) ([]google.Tab, error)
	FileMeta(ctx context.Context, userID, docID string) (google.FileMeta, error)
	ExportPDF(ctx context.Context, userID, docID, tabID string) (io.ReadCloser, error)
}

type Store interface {
	ListTrackedDocs(ctx context.Context, userID string) ([]dto.TrackedDoc, error)
	EnsureTabs(ctx context.Context, trackedDocID string, tabIDs, titles []string) error
	ListTabs(ctx context.Context, trackedDocID string) ([]dto.Tab, error)
}

type CV struct {
	DocID      string
	TabID      string
	Title      string
	SourceDoc  string
	ModifiedAt time.Time
	DocURL     string
	Visible    bool
}

type Service struct {
	gc    DocsClient
	store Store
}

func NewService(gc DocsClient, store Store) *Service {
	return &Service{gc: gc, store: store}
}

// List returns userID's CVs, one per visible or hidden tab across their
// tracked docs. A doc Google can no longer access is skipped, not an error.
func (s *Service) List(ctx context.Context, userID string) ([]CV, error) {
	if _, err := s.gc.HTTPClientForUser(ctx, userID); err != nil {
		return nil, fmt.Errorf("google account not connected: %w", err)
	}

	tracked, err := s.store.ListTrackedDocs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("cvtemplates.List list docs: %w", err)
	}

	var cvs []CV
	for _, doc := range tracked {
		tabs, err := s.gc.ListTabs(ctx, userID, doc.DocID)
		if err != nil {
			slog.WarnContext(ctx, "cvtemplates.List: skipping inaccessible doc",
				slog.String(logger.KeyDocID, doc.DocID),
				slog.Any(logger.KeyErr, err),
			)
			continue
		}
		meta, err := s.gc.FileMeta(ctx, userID, doc.DocID)
		if err != nil {
			slog.WarnContext(ctx, "cvtemplates.List: skipping doc (FileMeta failed)",
				slog.String(logger.KeyDocID, doc.DocID),
				slog.Any(logger.KeyErr, err),
			)
			continue
		}
		tabIDs := make([]string, len(tabs))
		tabTitles := make([]string, len(tabs))
		for i, tab := range tabs {
			tabIDs[i] = tab.ID
			tabTitles[i] = tab.Title
		}
		if err := s.store.EnsureTabs(ctx, doc.ID, tabIDs, tabTitles); err != nil {
			return nil, fmt.Errorf("cvtemplates.List ensure tabs: %w", err)
		}
		persisted, err := s.store.ListTabs(ctx, doc.ID)
		if err != nil {
			return nil, fmt.Errorf("cvtemplates.List list tabs: %w", err)
		}
		visibilityOf := make(map[string]bool, len(persisted))
		for _, p := range persisted {
			visibilityOf[p.TabID] = p.Visible
		}
		for _, tab := range tabs {
			cvs = append(cvs, CV{
				DocID:      doc.DocID,
				TabID:      tab.ID,
				Title:      tab.Title,
				SourceDoc:  meta.Title,
				ModifiedAt: meta.ModifiedAt,
				DocURL:     docTabURL(doc.DocID, tab.ID),
				Visible:    visibilityOf[tab.ID],
			})
		}
	}

	sort.Slice(cvs, func(i, j int) bool {
		if cvs[i].DocID != cvs[j].DocID {
			return cvs[i].DocID < cvs[j].DocID
		}
		return cvs[i].TabID < cvs[j].TabID
	})
	return cvs, nil
}

func (s *Service) ExportPDF(ctx context.Context, userID, docID, tabID string) (io.ReadCloser, error) {
	body, err := s.gc.ExportPDF(ctx, userID, docID, tabID)
	if err != nil {
		return nil, apperr.Upstream("failed to export PDF")
	}
	return body, nil
}

// The Docs API returns tabId values that already include the "t." prefix
// (e.g. "t.0"), so we normalise defensively rather than assuming the format.
func docTabURL(docID, tabID string) string {
	tab := tabID
	if !strings.HasPrefix(tab, "t.") {
		tab = "t." + tab
	}
	return fmt.Sprintf("https://docs.google.com/document/d/%s/edit?tab=%s", docID, tab)
}

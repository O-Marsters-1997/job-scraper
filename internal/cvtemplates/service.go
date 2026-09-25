package cvtemplates

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/docref"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/google"
)

// The Docs API returns tabId values that already include the "t." prefix (e.g. "t.0"),
// so we normalise defensively rather than assuming the format.
func docTabURL(docID, tabID string) string {
	tab := tabID
	if !strings.HasPrefix(tab, "t.") {
		tab = "t." + tab
	}
	return fmt.Sprintf("https://docs.google.com/document/d/%s/edit?tab=%s", docID, tab)
}

type googleClient interface {
	ListTabs(ctx context.Context, userID, docID string) ([]google.Tab, error)
	FileMeta(ctx context.Context, userID, docID string) (google.FileMeta, error)
}

type store interface {
	GetGoogleToken(ctx context.Context, userID string) (dto.GoogleToken, error)
	providers.TrackedDocProvider
	providers.TabProvider
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
	gc    googleClient
	store store
}

var (
	ErrInvalidDoc      = errors.New("invalid Google Docs URL or ID")
	ErrInaccessibleDoc = errors.New("cannot access document")
)

func NewService(gc googleClient, s store) *Service {
	return &Service{gc: gc, store: s}
}

// Docs that are inaccessible (deleted, permissions revoked) are logged and skipped.
func (s *Service) List(ctx context.Context, userID string) ([]CV, error) {
	if _, err := s.store.GetGoogleToken(ctx, userID); err != nil {
		if errors.Is(err, providers.ErrGoogleTokenNotFound) {
			return nil, fmt.Errorf("google account not connected: %w", err)
		}
		return nil, fmt.Errorf("cvtemplates.List check token: %w", err)
	}

	tracked, err := s.store.ListTrackedDocs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("cvtemplates.List list docs: %w", err)
	}

	var cvs []CV
	for _, doc := range tracked {
		tabs, err := s.gc.ListTabs(ctx, userID, doc.DocID)
		if err != nil {
			slog.Warn("cvtemplates.List: skipping inaccessible doc",
				slog.String("doc_id", doc.DocID),
				slog.Any("err", err),
			)
			continue
		}
		meta, err := s.gc.FileMeta(ctx, userID, doc.DocID)
		if err != nil {
			slog.Warn("cvtemplates.List: skipping doc (FileMeta failed)",
				slog.String("doc_id", doc.DocID),
				slog.Any("err", err),
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

func (s *Service) AddDoc(ctx context.Context, userID, urlOrID string) error {
	docID, err := docref.ParseDocID(urlOrID)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidDoc, err)
	}
	if _, err := s.gc.FileMeta(ctx, userID, docID); err != nil {
		return fmt.Errorf("%w: %w", ErrInaccessibleDoc, err)
	}
	return s.store.AddTrackedDoc(ctx, dto.AddTrackedDocInput{UserID: userID, DocID: docID})
}

func (s *Service) RemoveDoc(ctx context.Context, userID, docID string) error {
	return s.store.RemoveTrackedDoc(ctx, userID, docID)
}

func (s *Service) HideTab(ctx context.Context, userID, docID, tabID string) error {
	return s.store.HideTab(ctx, userID, docID, tabID)
}

func (s *Service) ShowTab(ctx context.Context, userID, docID, tabID string) error {
	return s.store.ShowTab(ctx, userID, docID, tabID)
}

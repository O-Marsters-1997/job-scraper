package cvtemplates

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/docref"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/google"
)

// googleClient is the subset of google.Client used by this service.
type googleClient interface {
	ListTabs(ctx context.Context, userID, docID string) ([]google.Tab, error)
	FileMeta(ctx context.Context, userID, docID string) (google.FileMeta, error)
}

// tokenChecker checks whether a user has a connected Google account.
type tokenChecker interface {
	GetGoogleToken(ctx context.Context, userID string) (dto.GoogleToken, error)
}

// CV represents a single CV tab derived from a tracked Google Doc.
type CV struct {
	DocID      string    `json:"doc_id"`
	TabID      string    `json:"tab_id"`
	Title      string    `json:"title"`
	SourceDoc  string    `json:"source_doc"`
	ModifiedAt time.Time `json:"modified_at"`
	DocURL     string    `json:"doc_url"`
}

// Service orchestrates CV template listing and tracked-doc management.
type Service struct {
	gc  googleClient
	tc  tokenChecker
	tdp providers.TrackedDocProvider
}

// NewService constructs a Service.
func NewService(gc googleClient, tc tokenChecker, tdp providers.TrackedDocProvider) *Service {
	return &Service{gc: gc, tc: tc, tdp: tdp}
}

// List returns all CV tabs across all tracked docs for the user.
// Docs that are inaccessible (deleted, permissions revoked) are logged and skipped.
func (s *Service) List(ctx context.Context, userID string) ([]CV, error) {
	if _, err := s.tc.GetGoogleToken(ctx, userID); err != nil {
		if errors.Is(err, providers.ErrGoogleTokenNotFound) {
			return nil, fmt.Errorf("google account not connected")
		}
		return nil, fmt.Errorf("cvtemplates.List check token: %w", err)
	}

	tracked, err := s.tdp.ListTrackedDocs(ctx, userID)
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
		for _, tab := range tabs {
			cvs = append(cvs, CV{
				DocID:      doc.DocID,
				TabID:      tab.ID,
				Title:      tab.Title,
				SourceDoc:  meta.Title,
				ModifiedAt: meta.ModifiedAt,
				DocURL:     fmt.Sprintf("https://docs.google.com/document/d/%s/edit?tab=t.%s", doc.DocID, tab.ID),
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

// AddDoc validates and tracks a new Google Doc by URL or raw doc ID.
func (s *Service) AddDoc(ctx context.Context, userID, urlOrID string) error {
	docID, err := docref.ParseDocID(urlOrID)
	if err != nil {
		return err
	}
	if _, err := s.gc.FileMeta(ctx, userID, docID); err != nil {
		return fmt.Errorf("cannot access document: %w", err)
	}
	return s.tdp.AddTrackedDoc(ctx, dto.AddTrackedDocInput{UserID: userID, DocID: docID})
}

// RemoveDoc removes a tracked doc for the user.
func (s *Service) RemoveDoc(ctx context.Context, userID, docID string) error {
	return s.tdp.RemoveTrackedDoc(ctx, userID, docID)
}

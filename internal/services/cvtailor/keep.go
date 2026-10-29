package cvtailor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/pdftext"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/tabcopy"
	"github.com/ollymarsters/job-scraper/internal/services/google"
)

const (
	noteNoDocsScope = "Kept as a separate Doc: allow FastTrack to edit your Google Docs to keep drafts as Tabs."
	noteMismatch    = "Kept as a separate Doc: the Tab did not render the same as the draft."
)

func keepName(claim dto.DraftClaim) string {
	return claim.CompanyName + " — " + claim.JobTitle
}

func (m *Module) placeKeep(ctx context.Context, claim dto.DraftClaim) (dto.KeepPlacement, error) {
	if claim.KeptTabID != "" {
		m.dropKeptTab(ctx, claim, claim.KeptTabID)
	}
	tabID, note, err := m.keepAsTab(ctx, claim)
	if err != nil {
		return dto.KeepPlacement{}, err
	}
	if tabID != "" {
		return dto.KeepPlacement{DocID: claim.DocID, TabID: tabID}, nil
	}
	if err := m.drive.RenameFile(ctx, claim.UserID, claim.DraftDocID, keepName(claim)); err != nil {
		return dto.KeepPlacement{}, fmt.Errorf("rename kept doc: %w", err)
	}
	return dto.KeepPlacement{DocID: claim.DraftDocID, Note: note}, nil
}

func (m *Module) keepAsTab(ctx context.Context, claim dto.DraftClaim) (tabID, note string, err error) {
	tabID, err = m.drive.AddChildTab(ctx, claim.UserID, claim.DocID, claim.TabID, keepName(claim))
	if errors.Is(err, google.ErrNoWriteAccess) {
		return "", noteNoDocsScope, nil
	}
	if err != nil {
		return "", "", fmt.Errorf("add kept tab: %w", err)
	}
	if err := m.store.SetKeptTab(ctx, claim, tabID); err != nil {
		m.dropKeptTab(ctx, claim, tabID)
		return "", "", err
	}
	note, err = m.fillKeptTab(ctx, claim, tabID)
	if err != nil || note != "" {
		m.dropKeptTab(ctx, claim, tabID)
		return "", note, err
	}
	return tabID, "", nil
}

func (m *Module) fillKeptTab(ctx context.Context, claim dto.DraftClaim, tabID string) (note string, err error) {
	src, err := m.docs.GetDocument(ctx, claim.UserID, claim.DraftDocID, claim.TabID)
	if err != nil {
		return "", fmt.Errorf("load draft doc: %w", err)
	}
	reqs, err := tabcopy.Requests(src, tabID)
	if errors.Is(err, tabcopy.ErrUnsupported) {
		return "Kept as a separate Doc: this CV can't be copied into a Tab (" + err.Error() + ").", nil
	}
	if err != nil {
		return "", fmt.Errorf("build kept tab: %w", err)
	}
	if err := m.drive.BatchUpdate(ctx, claim.UserID, claim.DocID, reqs); err != nil {
		return "", fmt.Errorf("fill kept tab: %w", err)
	}
	want, wantPages, err := m.svc.measure(ctx, claim.UserID, claim.DraftDocID, claim.TabID)
	if err != nil {
		return "", fmt.Errorf("measure draft: %w", err)
	}
	got, gotPages, err := m.svc.measure(ctx, claim.UserID, claim.DocID, tabID)
	if err != nil {
		return "", fmt.Errorf("measure kept tab: %w", err)
	}
	if gotPages != wantPages || !sameTextWhenReadable(want, got) {
		return noteMismatch, nil
	}
	return "", nil
}

func (m *Module) deleteTempDoc(ctx context.Context, claim dto.DraftClaim) {
	if err := m.drive.DeleteFile(ctx, claim.UserID, claim.DraftDocID); err != nil {
		m.log.ErrorContext(ctx, "delete draft copy failed", slog.String("doc_id", claim.DraftDocID), slog.Any(logger.KeyErr, err))
	}
}

func sameTextWhenReadable(a, b []byte) bool {
	la, errA := pdftext.Lines(a)
	lb, errB := pdftext.Lines(b)
	return errA != nil || errB != nil || slices.Equal(la, lb)
}

func (m *Module) dropKeptTab(ctx context.Context, claim dto.DraftClaim, tabID string) {
	if err := deleteTab(ctx, m.drive, claim.UserID, claim.DocID, tabID); err != nil {
		m.log.ErrorContext(ctx, "delete kept tab failed", slog.String("draft_id", claim.ID), slog.String("tab_id", tabID), slog.Any(logger.KeyErr, err))
		return
	}
	if err := m.store.SetKeptTab(ctx, claim, ""); err != nil {
		m.log.ErrorContext(ctx, "clear kept tab failed", slog.String("draft_id", claim.ID), slog.Any(logger.KeyErr, err))
	}
}

func deleteTab(ctx context.Context, drive Drive, userID, docID, tabID string) error {
	req, err := json.Marshal(map[string]any{"deleteTab": map[string]string{"tabId": tabID}})
	if err != nil {
		return fmt.Errorf("marshal deleteTab: %w", err)
	}
	return drive.BatchUpdate(ctx, userID, docID, []json.RawMessage{req})
}

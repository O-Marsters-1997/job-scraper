package cvtailor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"time"
	"unicode/utf8"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/docedit"
	"github.com/ollymarsters/job-scraper/internal/services/google"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
)

const (
	statusReady    = "ready"
	statusKeeping  = "keeping"
	cleanupTimeout = 30 * time.Second

	shortenBullets = 3
)

var errInvalidEdit = errors.New("model returned an invalid edit")

type Editor interface {
	Edit(ctx context.Context, apiKey string, in cvedit.Input) (cvedit.Result, error)
}

// Credentials reads a user's stored provider API key.
type Credentials interface {
	Get(ctx context.Context, userID, provider string) (string, error)
}

// Drive is the Google write surface a Draft is built on.
type Drive interface {
	ListTabs(ctx context.Context, userID, docID string) ([]google.Tab, error)
	CopyFile(ctx context.Context, userID, fileID, name string) (string, error)
	BatchUpdate(ctx context.Context, userID, docID string, requests []json.RawMessage) error
	DeleteFile(ctx context.Context, userID, fileID string) error
	RenameFile(ctx context.Context, userID, fileID, name string) error
	ExportPDF(ctx context.Context, userID, docID, tabID string) (io.ReadCloser, error)
}

// Docs is everything the module needs from the Google client.
type Docs interface {
	DocFetcher
	Drive
}

func (m *Module) RunTick(ctx context.Context) error {
	for {
		claim, err := m.store.ClaimDraft(ctx)
		if errors.Is(err, data.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := m.process(ctx, claim); err != nil {
			slog.ErrorContext(ctx, "draft generation failed", slog.String("draft_id", claim.ID), slog.Any(logger.KeyErr, err))
		}
	}
}

func (m *Module) process(ctx context.Context, claim dto.DraftClaim) error {
	if claim.Keeping {
		return m.keep(ctx, claim)
	}
	if claim.Attempts > dto.MaxDraftAttempts {
		return m.store.FailDraft(ctx, claim, dto.DraftFailure{Reason: "gave up after repeated crashes", Terminal: true})
	}
	if claim.DraftDocID != "" {
		if err := m.drive.DeleteFile(ctx, claim.UserID, claim.DraftDocID); err != nil {
			return m.fail(ctx, claim, fmt.Errorf("delete earlier copy: %w", err), false)
		}
		if err := m.store.SetDraftDoc(ctx, claim, ""); err != nil {
			return err
		}
	}

	docID, res, err := m.generate(ctx, claim)
	if err != nil {
		cleared := docID == "" || m.deleteCopy(claim.UserID, docID)
		return m.fail(ctx, claim, err, cleared)
	}
	res.DraftDocID = docID
	return m.store.CompleteDraft(ctx, claim, res)
}

func (m *Module) keep(ctx context.Context, claim dto.DraftClaim) error {
	name := claim.CompanyName + " \u2014 " + claim.JobTitle
	if err := m.drive.RenameFile(ctx, claim.UserID, claim.DraftDocID, name); err != nil {
		cause := fmt.Errorf("rename kept doc: %w", err)
		_, terminal := apperr.StatusFor(cause)
		if ferr := m.store.FailKeep(ctx, claim, dto.DraftFailure{Reason: cause.Error(), Terminal: terminal}); ferr != nil {
			return errors.Join(cause, ferr)
		}
		return cause
	}
	return m.store.CompleteKeep(ctx, claim)
}

func (m *Module) deleteCopy(userID, docID string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
	defer cancel()
	if err := m.drive.DeleteFile(ctx, userID, docID); err != nil {
		slog.ErrorContext(ctx, "delete draft copy failed", slog.String("doc_id", docID), slog.Any(logger.KeyErr, err))
		return false
	}
	return true
}

func (m *Module) fail(ctx context.Context, claim dto.DraftClaim, cause error, clearDoc bool) error {
	_, terminal := apperr.StatusFor(cause)
	failure := dto.DraftFailure{Reason: cause.Error(), Terminal: terminal, ClearDoc: clearDoc}
	if err := m.store.FailDraft(ctx, claim, failure); err != nil {
		return errors.Join(cause, err)
	}
	return cause
}

func (m *Module) generate(ctx context.Context, claim dto.DraftClaim) (string, dto.DraftResult, error) {
	key, err := m.creds.Get(ctx, claim.UserID, jev.Provider)
	if errors.Is(err, data.ErrNotFound) {
		return "", dto.DraftResult{}, apperr.Unprocessable("connect an OpenRouter key in Settings, AI to tailor a CV")
	}
	if err != nil {
		return "", dto.DraftResult{}, fmt.Errorf("load credential: %w", err)
	}

	pl, err := m.plan(ctx, claim, claim.DocID)
	if err != nil {
		return "", dto.DraftResult{}, err
	}
	res, err := m.editClean(ctx, key, pl, pl.input(claim.JobDescription))
	if err != nil {
		return "", dto.DraftResult{}, err
	}
	cost := res.Cost

	docID, err := m.copyTab(ctx, claim)
	if err != nil {
		return docID, dto.DraftResult{}, err
	}
	if err := m.applyEdits(ctx, claim, docID, pl, res.Edits); err != nil {
		return docID, dto.DraftResult{}, err
	}

	_, basePages, err := m.measure(ctx, claim.UserID, claim.DocID, claim.TabID)
	if err != nil {
		return docID, dto.DraftResult{}, fmt.Errorf("count base pages: %w", err)
	}
	draftPDF, draftPages, err := m.measure(ctx, claim.UserID, docID, claim.TabID)
	if err != nil {
		return docID, dto.DraftResult{}, fmt.Errorf("count draft pages: %w", err)
	}
	if draftPages > basePages {
		slog.InfoContext(ctx, "draft runs over the base CV, shortening",
			slog.Int("base_pages", basePages), slog.Int("draft_pages", draftPages))
		short, err := m.shorten(ctx, claim, key, docID, pl, res)
		if err != nil {
			return docID, dto.DraftResult{}, err
		}
		res, cost = short, cost+short.Cost
		if draftPDF, draftPages, err = m.measure(ctx, claim.UserID, docID, claim.TabID); err != nil {
			return docID, dto.DraftResult{}, fmt.Errorf("count draft pages: %w", err)
		}
	}

	finalDraft := pl.draft(res.Edits, basePages, draftPages)
	finalDraft.Parse = parseInput(ctx, draftPDF, pl.structure.Headings)

	editSet, err := json.Marshal(res.Edits)
	if err != nil {
		return docID, dto.DraftResult{}, fmt.Errorf("marshal edit set: %w", err)
	}
	return docID, dto.DraftResult{
		EditSet: editSet, RawOutput: res.Raw, Model: cvedit.Model, PromptVersion: cvedit.PromptVersion,
		JobFingerprint: claim.JobFingerprint, Cost: cost,
		Findings: toDraftFindings(checks.Run(finalDraft)),
	}, nil
}

func (m *Module) editClean(ctx context.Context, key string, pl plan, in cvedit.Input) (cvedit.Result, error) {
	res, err := m.editValid(ctx, key, pl, in)
	if err != nil {
		return cvedit.Result{}, err
	}
	var reverted []string
	res.Edits, reverted = pl.revertBlocked(res.Edits)
	if len(reverted) > 0 {
		slog.InfoContext(ctx, "reverted blocked draft slots", slog.Any("checks", reverted))
	}
	return res, nil
}

func (m *Module) shorten(ctx context.Context, claim dto.DraftClaim, key, docID string, pl plan, prior cvedit.Result) (cvedit.Result, error) {
	in := pl.input(claim.JobDescription)
	in.PriorEdits, in.ShortenBullets = &prior.Edits, longestBullets(prior.Edits, shortenBullets)
	res, err := m.editClean(ctx, key, pl, in)
	if err != nil {
		return cvedit.Result{}, err
	}
	copyPlan, err := m.plan(ctx, claim, docID)
	if err != nil {
		return cvedit.Result{}, err
	}
	if err := m.applyEdits(ctx, claim, docID, copyPlan, res.Edits); err != nil {
		return cvedit.Result{}, err
	}
	return res, nil
}

func (m *Module) editValid(ctx context.Context, key string, pl plan, in cvedit.Input) (cvedit.Result, error) {
	res, err := m.editor.Edit(ctx, key, in)
	if err != nil {
		return res, fmt.Errorf("edit: %w", err)
	}
	return res, pl.validate(res.Edits)
}

func longestBullets(edits cvedit.EditSet, n int) []string {
	var texts []string
	for _, pe := range edits.Positions {
		for _, b := range pe.Bullets {
			texts = append(texts, b.Text)
		}
	}
	slices.SortStableFunc(texts, func(a, b string) int { return utf8.RuneCountInString(b) - utf8.RuneCountInString(a) })
	return texts[:min(n, len(texts))]
}

func toDraftFindings(findings []checks.Finding) []dto.DraftFinding {
	out := make([]dto.DraftFinding, len(findings))
	for i, f := range findings {
		out[i] = dto.DraftFinding{Check: f.Check, Severity: string(f.Severity), SlotID: f.SlotID, Message: f.Message, Score: f.Score}
	}
	return out
}

func (m *Module) copyTab(ctx context.Context, claim dto.DraftClaim) (string, error) {
	tabs, err := m.drive.ListTabs(ctx, claim.UserID, claim.DocID)
	if err != nil {
		return "", fmt.Errorf("list tabs: %w", err)
	}
	if !slices.ContainsFunc(tabs, func(t google.Tab) bool { return t.ID == claim.TabID }) {
		return "", apperr.Unprocessable("the base CV tab must be a top-level tab")
	}
	docID, err := m.drive.CopyFile(ctx, claim.UserID, claim.DocID, "CV draft "+claim.ID[:8])
	if err != nil {
		return "", fmt.Errorf("copy doc: %w", err)
	}
	if err := m.store.SetDraftDoc(ctx, claim, docID); err != nil {
		return docID, err
	}

	var trim []json.RawMessage
	for _, t := range tabs {
		if t.ID == claim.TabID {
			continue
		}
		req, err := json.Marshal(map[string]any{"deleteTab": map[string]string{"tabId": t.ID}})
		if err != nil {
			return docID, fmt.Errorf("marshal deleteTab: %w", err)
		}
		trim = append(trim, req)
	}
	if len(trim) > 0 {
		if err := m.drive.BatchUpdate(ctx, claim.UserID, docID, trim); err != nil {
			return docID, fmt.Errorf("remove other tabs: %w", err)
		}
	}
	return docID, nil
}

func (m *Module) applyEdits(ctx context.Context, claim dto.DraftClaim, docID string, pl plan, edits cvedit.EditSet) error {
	requests, err := docedit.Requests(pl.structure, pl.slotIDs(), edits)
	if err != nil {
		return fmt.Errorf("build doc edits: %w", err)
	}
	if len(requests) == 0 {
		return nil
	}
	raw := make([]json.RawMessage, len(requests))
	for i, r := range requests {
		b, err := json.Marshal(r)
		if err != nil {
			return fmt.Errorf("marshal doc edit: %w", err)
		}
		raw[i] = b
	}
	if err := m.drive.BatchUpdate(ctx, claim.UserID, docID, raw); err != nil {
		return fmt.Errorf("apply doc edits: %w", err)
	}
	return nil
}

package cvtailor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/openrouter"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/docedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/letterwrite"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/store"
	"github.com/ollymarsters/job-scraper/internal/services/google"
)

const (
	statusReady    = "ready"
	statusKeeping  = "keeping"
	statusFailed   = "failed"
	cleanupTimeout = 30 * time.Second
)

type Editor interface {
	Explain(ctx context.Context, apiKey string, in cvedit.ExplainInput) (cvedit.ExplainResult, error)
	Insights(ctx context.Context, apiKey string, in cvedit.InsightsInput) (cvedit.InsightsResult, error)
}

type Scaffolder interface {
	Scaffold(ctx context.Context, apiKey string, in letterwrite.Input) (letterwrite.Result, error)
}

var _ Scaffolder = (*letterwrite.Client)(nil)

// Credentials reads a user's stored provider API key.
type Credentials interface {
	Get(ctx context.Context, userID, provider string) (string, error)
}

// Drive is the Google write surface a Draft is built on.
type Drive interface {
	ListTabs(ctx context.Context, userID, docID string) ([]google.Tab, error)
	AddTab(ctx context.Context, userID, docID, title string) (string, error)
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
	if claim.Attempts > dto.MaxDraftAttempts {
		return m.store.FailKeep(ctx, claim, dto.DraftFailure{Reason: "gave up after repeated crashes", Terminal: true})
	}
	name := claim.CompanyName + " — " + claim.JobTitle
	cause := m.drive.RenameFile(ctx, claim.UserID, claim.DraftDocID, name)
	if cause != nil {
		cause = fmt.Errorf("rename kept doc: %w", cause)
	} else if outcomes, err := m.svc.keepOutcomes(ctx, claim); err != nil {
		cause = fmt.Errorf("compute outcomes: %w", err)
	} else if cause = m.store.CompleteKeep(ctx, claim, outcomes); cause == nil {
		return nil
	} else if !errors.Is(cause, store.ErrKeptDraftExists) {
		return cause
	}
	_, terminal := apperr.StatusFor(cause)
	failure := dto.DraftFailure{Reason: cause.Error(), Terminal: terminal || errors.Is(cause, store.ErrKeptDraftExists)}
	if err := m.store.FailKeep(ctx, claim, failure); err != nil {
		return errors.Join(cause, err)
	}
	return cause
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
	stop := stepTimer(ctx, claim.ID, "plan")
	pl, err := m.svc.plan(ctx, claim, claim.DocID)
	stop()
	if err != nil {
		return "", dto.DraftResult{}, err
	}
	edits := pl.keepAll()
	var lean chan leanResult
	if claim.Mode == dto.DraftModeGenerate {
		lean = make(chan leanResult, 1)
		go func() {
			defer stepTimer(ctx, claim.ID, "replace_low_lean")()
			e, err := m.svc.replaceLowLean(ctx, claim, pl)
			lean <- leanResult{edits: e, err: err}
		}()
	}
	docID, err := m.copyTab(ctx, claim)
	if lean != nil {
		res := <-lean
		if err == nil {
			edits, err = res.edits, res.err
		}
	}
	if err != nil {
		return docID, dto.DraftResult{}, err
	}
	stop = stepTimer(ctx, claim.ID, "apply_edits")
	err = m.svc.applyEdits(ctx, claim.UserID, docID, pl, edits)
	stop()
	if err != nil {
		return docID, dto.DraftResult{}, err
	}
	editSet, err := json.Marshal(edits)
	if err != nil {
		return docID, dto.DraftResult{}, fmt.Errorf("marshal edit set: %w", err)
	}
	baseContent, err := json.Marshal(pl.baseContent())
	if err != nil {
		return docID, dto.DraftResult{}, fmt.Errorf("marshal base content: %w", err)
	}
	return docID, dto.DraftResult{
		EditSet: editSet, BaseContent: baseContent, JobFingerprint: claim.JobFingerprint, Findings: []dto.DraftFinding{},
	}, nil
}

type leanResult struct {
	edits cvedit.EditSet
	err   error
}

func stepTimer(ctx context.Context, draftID, step string) func() {
	start := time.Now()
	return func() {
		slog.InfoContext(ctx, "draft step", slog.String("draft_id", draftID), slog.String("step", step), slog.Int64("duration_ms", time.Since(start).Milliseconds()))
	}
}

func (s *Service) replaceLowLean(ctx context.Context, claim dto.DraftClaim, pl plan) (cvedit.EditSet, error) {
	bank, err := s.store.ListPositions(ctx, claim.UserID)
	if err != nil {
		return cvedit.EditSet{}, err
	}
	answers, err := s.asker.Ask(ctx, claim.UserID, claim.JobID, bankQuestions(bank))
	if err != nil {
		return cvedit.EditSet{}, err
	}
	lines, err := sourceLines(pl.raw, pl)
	if err != nil {
		return cvedit.EditSet{}, err
	}
	reps := leanRule{}.Replacements(RuleInput{Positions: bank, Content: pl.baseContent().content(), Answers: answers, Lines: lines})
	return pl.withReplacements(bank, reps), nil
}

func suggestError(err error) error {
	var se *openrouter.StatusError
	if errors.As(err, &se) {
		switch se.Code {
		case http.StatusUnauthorized, http.StatusPaymentRequired, http.StatusForbidden:
			return apperr.Unprocessable("OpenRouter refused the request; check your key and credit in Settings, AI")
		}
	}
	return apperr.Upstream("the suggestion model failed")
}

func toDraftFindings(findings []checks.Finding) []dto.DraftFinding {
	out := make([]dto.DraftFinding, len(findings))
	for i, f := range findings {
		out[i] = dto.DraftFinding{Check: f.Check, Severity: string(f.Severity), SlotID: f.SlotID, Message: f.Message, Score: f.Score}
	}
	return out
}

func (m *Module) copyTab(ctx context.Context, claim dto.DraftClaim) (string, error) {
	stop := stepTimer(ctx, claim.ID, "list_tabs")
	tabs, err := m.drive.ListTabs(ctx, claim.UserID, claim.DocID)
	stop()
	if err != nil {
		return "", fmt.Errorf("list tabs: %w", err)
	}
	if !slices.ContainsFunc(tabs, func(t google.Tab) bool { return t.ID == claim.TabID }) {
		return "", apperr.Unprocessable("the base CV tab must be a top-level tab")
	}
	stop = stepTimer(ctx, claim.ID, "copy_file")
	docID, err := m.drive.CopyFile(ctx, claim.UserID, claim.DocID, "CV draft "+claim.ID[:8])
	stop()
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
		defer stepTimer(ctx, claim.ID, "delete_tabs")()
		if err := m.drive.BatchUpdate(ctx, claim.UserID, docID, trim); err != nil {
			return docID, fmt.Errorf("remove other tabs: %w", err)
		}
	}
	return docID, nil
}

func (s *Service) applyEdits(ctx context.Context, userID, docID string, pl plan, edits cvedit.EditSet) error {
	raw, err := s.editRequests(pl, edits)
	if err != nil || len(raw) == 0 {
		return err
	}
	if err := s.drive.BatchUpdate(ctx, userID, docID, raw); err != nil {
		return fmt.Errorf("apply doc edits: %w", err)
	}
	return nil
}

func (s *Service) editRequests(pl plan, edits cvedit.EditSet) ([]json.RawMessage, error) {
	requests, err := docedit.Requests(pl.structure, pl.slotIDs(), edits)
	if err != nil {
		return nil, fmt.Errorf("build doc edits: %w", err)
	}
	raw := make([]json.RawMessage, len(requests))
	for i, r := range requests {
		b, err := json.Marshal(r)
		if err != nil {
			return nil, fmt.Errorf("marshal doc edit: %w", err)
		}
		raw[i] = b
	}
	return raw, nil
}

package cvtailor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/docparse"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/docedit"
	"github.com/ollymarsters/job-scraper/internal/services/google"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
)

const (
	statusReady    = "ready"
	cleanupTimeout = 30 * time.Second
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
}

// Docs is everything the module needs from the Google client.
type Docs interface {
	DocFetcher
	Drive
}

// Generator turns claimed pending Drafts into Google Docs.
type Generator struct {
	store    Store
	docs     DocFetcher
	drive    Drive
	editor   Editor
	creds    Credentials
	interval time.Duration
}

type GeneratorDeps struct {
	Store    Store
	Docs     DocFetcher
	Drive    Drive
	Editor   Editor
	Creds    Credentials
	Interval time.Duration
}

func NewGenerator(deps GeneratorDeps) *Generator {
	interval := deps.Interval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &Generator{
		store: deps.Store, docs: deps.Docs, drive: deps.Drive, editor: deps.Editor, creds: deps.Creds, interval: interval,
	}
}

// Run drains the Draft queue every interval until ctx is cancelled.
func (g *Generator) Run(ctx context.Context) error {
	ticker := time.NewTicker(g.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := g.RunTick(ctx); err != nil && ctx.Err() == nil {
				slog.ErrorContext(ctx, "draft tick failed", slog.Any(logger.KeyErr, err))
			}
		}
	}
}

// RunTick generates every Draft that is due.
func (g *Generator) RunTick(ctx context.Context) error {
	for {
		claim, err := g.store.ClaimDraft(ctx)
		if errors.Is(err, data.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := g.process(ctx, claim); err != nil {
			slog.ErrorContext(ctx, "draft generation failed", slog.String("draft_id", claim.ID), slog.Any(logger.KeyErr, err))
		}
	}
}

func (g *Generator) process(ctx context.Context, claim dto.DraftClaim) error {
	if claim.Attempts > dto.MaxDraftAttempts {
		return g.store.FailDraft(ctx, claim, dto.DraftFailure{Reason: "gave up after repeated crashes", Terminal: true})
	}
	if claim.DraftDocID != "" {
		if err := g.drive.DeleteFile(ctx, claim.UserID, claim.DraftDocID); err != nil {
			return g.fail(ctx, claim, fmt.Errorf("delete earlier copy: %w", err), false)
		}
		if err := g.store.SetDraftDoc(ctx, claim, ""); err != nil {
			return err
		}
	}

	docID, res, err := g.generate(ctx, claim)
	if err != nil {
		cleared := docID == "" || g.deleteCopy(claim.UserID, docID)
		return g.fail(ctx, claim, err, cleared)
	}
	res.DraftDocID = docID
	return g.store.CompleteDraft(ctx, claim, res)
}

func (g *Generator) deleteCopy(userID, docID string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
	defer cancel()
	if err := g.drive.DeleteFile(ctx, userID, docID); err != nil {
		slog.ErrorContext(ctx, "delete draft copy failed", slog.String("doc_id", docID), slog.Any(logger.KeyErr, err))
		return false
	}
	return true
}

func (g *Generator) fail(ctx context.Context, claim dto.DraftClaim, cause error, clearDoc bool) error {
	_, terminal := apperr.StatusFor(cause)
	failure := dto.DraftFailure{Reason: cause.Error(), Terminal: terminal, ClearDoc: clearDoc}
	if err := g.store.FailDraft(ctx, claim, failure); err != nil {
		return errors.Join(cause, err)
	}
	return cause
}

// generate builds the Draft Doc and returns its id, even alongside an error
// so the caller can delete a copy that was made but not finished.
func (g *Generator) generate(ctx context.Context, claim dto.DraftClaim) (string, dto.DraftResult, error) {
	key, err := g.creds.Get(ctx, claim.UserID, jev.Provider)
	if errors.Is(err, data.ErrNotFound) {
		return "", dto.DraftResult{}, apperr.Unprocessable("connect an OpenRouter key in Settings, AI to tailor a CV")
	}
	if err != nil {
		return "", dto.DraftResult{}, fmt.Errorf("load credential: %w", err)
	}

	pl, err := g.plan(ctx, claim)
	if err != nil {
		return "", dto.DraftResult{}, err
	}
	res, err := g.editor.Edit(ctx, key, pl.input(claim.JobDescription))
	if err != nil {
		return "", dto.DraftResult{}, fmt.Errorf("edit: %w", err)
	}
	if err := pl.validate(res.Edits); err != nil {
		return "", dto.DraftResult{}, err
	}
	requests, err := docedit.Requests(pl.structure, pl.slotIDs(), res.Edits)
	if err != nil {
		return "", dto.DraftResult{}, fmt.Errorf("build doc edits: %w", err)
	}
	editSet, err := json.Marshal(res.Edits)
	if err != nil {
		return "", dto.DraftResult{}, fmt.Errorf("marshal edit set: %w", err)
	}
	result := dto.DraftResult{
		EditSet: editSet, RawOutput: res.Raw, Model: cvedit.Model, PromptVersion: cvedit.PromptVersion,
		JobFingerprint: claim.JobFingerprint, Cost: res.Cost,
	}

	docID, err := g.copyTab(ctx, claim)
	if err != nil {
		return docID, dto.DraftResult{}, err
	}
	if err := g.applyEdits(ctx, claim, docID, requests); err != nil {
		return docID, dto.DraftResult{}, err
	}
	return docID, result, nil
}

func (g *Generator) copyTab(ctx context.Context, claim dto.DraftClaim) (string, error) {
	tabs, err := g.drive.ListTabs(ctx, claim.UserID, claim.DocID)
	if err != nil {
		return "", fmt.Errorf("list tabs: %w", err)
	}
	if !slices.ContainsFunc(tabs, func(t google.Tab) bool { return t.ID == claim.TabID }) {
		return "", apperr.Unprocessable("the base CV tab must be a top-level tab")
	}
	docID, err := g.drive.CopyFile(ctx, claim.UserID, claim.DocID, "CV draft "+claim.ID[:8])
	if err != nil {
		return "", fmt.Errorf("copy doc: %w", err)
	}
	if err := g.store.SetDraftDoc(ctx, claim, docID); err != nil {
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
		if err := g.drive.BatchUpdate(ctx, claim.UserID, docID, trim); err != nil {
			return docID, fmt.Errorf("remove other tabs: %w", err)
		}
	}
	return docID, nil
}

func (g *Generator) applyEdits(ctx context.Context, claim dto.DraftClaim, docID string, requests []docedit.Request) error {
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
	if err := g.drive.BatchUpdate(ctx, claim.UserID, docID, raw); err != nil {
		return fmt.Errorf("apply doc edits: %w", err)
	}
	return nil
}

type planned struct {
	cvedit.Position
	slotIDs []string
	cited   map[string]bool
}

type plan struct {
	structure docparse.DocStructure
	positions []planned
}

func (g *Generator) plan(ctx context.Context, claim dto.DraftClaim) (plan, error) {
	raw, err := g.docs.GetDocument(ctx, claim.UserID, claim.DocID, claim.TabID)
	if err != nil {
		return plan{}, fmt.Errorf("load CV tab: %w", err)
	}
	ds, err := docparse.Parse(raw)
	if err != nil {
		return plan{}, apperr.Unprocessable("could not read the CV tab")
	}
	bank, err := g.store.ListPositions(ctx, claim.UserID)
	if err != nil {
		return plan{}, err
	}
	mappings, err := g.store.ListHeadingMappings(ctx, claim.UserID, claim.DocID, claim.TabID)
	if err != nil {
		return plan{}, err
	}
	positionOfHeading := make(map[string]string, len(mappings))
	for _, m := range mappings {
		if m.PositionID != nil {
			positionOfHeading[m.HeadingText] = *m.PositionID
		}
	}

	slotsOf := map[string][]docparse.Slot{}
	for _, slot := range ds.Slots {
		if slot.HeadingIndex < 0 {
			continue
		}
		if pid, ok := positionOfHeading[ds.Headings[slot.HeadingIndex].Text]; ok {
			slotsOf[pid] = append(slotsOf[pid], slot)
		}
	}

	confirmed := make(map[string]bool, len(claim.AchievementIDs))
	for _, id := range claim.AchievementIDs {
		confirmed[id] = true
	}
	pl := plan{structure: ds}
	found := 0
	for _, p := range bank {
		pp := planned{
			Position: cvedit.Position{ID: p.ID, Employer: p.Employer, Title: p.Title},
			cited:    map[string]bool{},
		}
		for _, a := range p.Achievements {
			if confirmed[a.ID] {
				pp.Achievements = append(pp.Achievements, cvedit.Achievement{ID: a.ID, Text: a.Text})
				pp.cited[a.ID] = true
			}
		}
		if len(pp.Achievements) == 0 {
			continue
		}
		found += len(pp.Achievements)
		slots := slotsOf[p.ID]
		if len(slots) == 0 {
			return plan{}, apperr.Unprocessable("a chosen position is no longer mapped to a heading of the CV tab")
		}
		for _, s := range slots {
			pp.SlotTexts = append(pp.SlotTexts, s.Text)
			pp.slotIDs = append(pp.slotIDs, s.ID)
		}
		pl.positions = append(pl.positions, pp)
	}
	if found != len(confirmed) {
		return plan{}, apperr.Unprocessable("a chosen achievement no longer exists")
	}
	return pl, nil
}

func (pl plan) input(jobDescription string) cvedit.Input {
	in := cvedit.Input{JobDescription: jobDescription}
	for _, p := range pl.positions {
		in.Positions = append(in.Positions, p.Position)
	}
	return in
}

func (pl plan) slotIDs() docedit.PositionSlots {
	out := make(docedit.PositionSlots, len(pl.positions))
	for _, p := range pl.positions {
		out[p.ID] = p.slotIDs
	}
	return out
}

// validate rejects an edit that cites an Achievement outside the confirmed
// set, edits a Position it wasn't given, or has more bullets than slots.
func (pl plan) validate(edits cvedit.EditSet) error {
	byID := make(map[string]planned, len(pl.positions))
	for _, p := range pl.positions {
		byID[p.ID] = p
	}
	for _, pe := range edits.Positions {
		p, ok := byID[pe.PositionID]
		if !ok {
			return fmt.Errorf("%w: unknown position %q", errInvalidEdit, pe.PositionID)
		}
		if len(pe.Bullets) > len(p.slotIDs) {
			return fmt.Errorf("%w: %d bullets for %d slots", errInvalidEdit, len(pe.Bullets), len(p.slotIDs))
		}
		for _, b := range pe.Bullets {
			if strings.TrimSpace(b.Text) == "" || len(b.AchievementIDs) == 0 {
				return fmt.Errorf("%w: bullet with no text or citation", errInvalidEdit)
			}
			for _, id := range b.AchievementIDs {
				if !p.cited[id] {
					return fmt.Errorf("%w: achievement %q is not confirmed for position %q", errInvalidEdit, id, pe.PositionID)
				}
			}
		}
	}
	return nil
}

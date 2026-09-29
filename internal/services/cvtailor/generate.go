package cvtailor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/docparse"
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
	cleanupTimeout = 30 * time.Second

	maxCheckRetries = 2
	shortenBullets  = 3
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
	ExportPDF(ctx context.Context, userID, docID, tabID string) (io.ReadCloser, error)
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

	pl, err := g.plan(ctx, claim, claim.DocID)
	if err != nil {
		return "", dto.DraftResult{}, err
	}
	res, cost, err := g.editUntilClean(ctx, key, pl, pl.input(claim.JobDescription))
	if err != nil {
		return "", dto.DraftResult{}, err
	}

	docID, err := g.copyTab(ctx, claim)
	if err != nil {
		return docID, dto.DraftResult{}, err
	}
	if err := g.applyEdits(ctx, claim, docID, pl, res.Edits); err != nil {
		return docID, dto.DraftResult{}, err
	}

	basePages, err := g.pageCount(ctx, claim.UserID, claim.DocID, claim.TabID)
	if err != nil {
		return docID, dto.DraftResult{}, fmt.Errorf("count base pages: %w", err)
	}
	draftPages, err := g.pageCount(ctx, claim.UserID, docID, claim.TabID)
	if err != nil {
		return docID, dto.DraftResult{}, fmt.Errorf("count draft pages: %w", err)
	}
	if draftPages > basePages {
		short, shortCost, err := g.shorten(ctx, claim, key, docID, pl, res)
		if err != nil {
			return docID, dto.DraftResult{}, err
		}
		res, cost = short, cost+shortCost
		if draftPages, err = g.pageCount(ctx, claim.UserID, docID, claim.TabID); err != nil {
			return docID, dto.DraftResult{}, fmt.Errorf("count draft pages: %w", err)
		}
	}

	editSet, err := json.Marshal(res.Edits)
	if err != nil {
		return docID, dto.DraftResult{}, fmt.Errorf("marshal edit set: %w", err)
	}
	return docID, dto.DraftResult{
		EditSet: editSet, RawOutput: res.Raw, Model: cvedit.Model, PromptVersion: cvedit.PromptVersion,
		JobFingerprint: claim.JobFingerprint, Cost: cost,
		Findings: toDraftFindings(checks.Run(pl.draft(res.Edits, basePages, draftPages))),
	}, nil
}

func (g *Generator) editUntilClean(ctx context.Context, key string, pl plan, in cvedit.Input) (cvedit.Result, float64, error) {
	res, err := g.editValid(ctx, key, pl, in)
	if err != nil {
		return cvedit.Result{}, res.Cost, err
	}
	cost := res.Cost
	for range maxCheckRetries {
		blocks := blocking(checks.Run(pl.draft(res.Edits, 0, 0)))
		if len(blocks) == 0 {
			break
		}
		prior := res.Edits
		in.PriorEdits, in.PriorFindings = &prior, blocks
		retry, err := g.editValid(ctx, key, pl, in)
		cost += retry.Cost
		if err != nil {
			slog.WarnContext(ctx, "draft retry failed, keeping the blocked edit", slog.Any(logger.KeyErr, err))
			break
		}
		res = retry
	}
	return res, cost, nil
}

func (g *Generator) shorten(ctx context.Context, claim dto.DraftClaim, key, docID string, pl plan, prior cvedit.Result) (cvedit.Result, float64, error) {
	in := pl.input(claim.JobDescription)
	in.PriorEdits, in.ShortenBullets = &prior.Edits, longestBullets(prior.Edits, shortenBullets)
	res, cost, err := g.editUntilClean(ctx, key, pl, in)
	if err != nil {
		return cvedit.Result{}, cost, err
	}
	copyPlan, err := g.plan(ctx, claim, docID)
	if err != nil {
		return cvedit.Result{}, cost, err
	}
	if err := g.applyEdits(ctx, claim, docID, copyPlan, res.Edits); err != nil {
		return cvedit.Result{}, cost, err
	}
	return res, cost, nil
}

func (g *Generator) editValid(ctx context.Context, key string, pl plan, in cvedit.Input) (cvedit.Result, error) {
	res, err := g.editor.Edit(ctx, key, in)
	if err != nil {
		return res, fmt.Errorf("edit: %w", err)
	}
	return res, pl.validate(res.Edits)
}

func blocking(findings []checks.Finding) []cvedit.Finding {
	var out []cvedit.Finding
	for _, f := range findings {
		if f.Severity == checks.Block {
			out = append(out, cvedit.Finding{Check: f.Check, SlotID: f.SlotID, Message: f.Message})
		}
	}
	return out
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

func (g *Generator) applyEdits(ctx context.Context, claim dto.DraftClaim, docID string, pl plan, edits cvedit.EditSet) error {
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
	if err := g.drive.BatchUpdate(ctx, claim.UserID, docID, raw); err != nil {
		return fmt.Errorf("apply doc edits: %w", err)
	}
	return nil
}

type planned struct {
	cvedit.Position
	slotIDs []string
	texts   map[string]string
}

type plan struct {
	structure docparse.DocStructure
	positions []planned
	bank      []string
}

func (g *Generator) plan(ctx context.Context, claim dto.DraftClaim, docID string) (plan, error) {
	raw, err := g.docs.GetDocument(ctx, claim.UserID, docID, claim.TabID)
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
			texts:    map[string]string{},
		}
		for _, a := range p.Achievements {
			pl.bank = append(pl.bank, a.Text)
			if confirmed[a.ID] {
				pp.Achievements = append(pp.Achievements, cvedit.Achievement{ID: a.ID, Text: a.Text})
				pp.texts[a.ID] = a.Text
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
	if pl.structure.Profile != nil {
		in.HasProfile, in.BaseProfile = true, pl.structure.Profile.Text
	}
	if pl.structure.Skills != nil {
		in.HasSkills, in.BaseSkills = true, pl.structure.Skills.Items
	}
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
				if _, ok := p.texts[id]; !ok {
					return fmt.Errorf("%w: achievement %q is not confirmed for position %q", errInvalidEdit, id, pe.PositionID)
				}
			}
		}
	}
	return nil
}

func (pl plan) draft(edits cvedit.EditSet, basePages, draftPages int) checks.Draft {
	d := checks.Draft{Bank: pl.bank, Skills: edits.Skills, JobSkills: edits.JobSkills, BasePages: basePages, DraftPages: draftPages}
	if pl.structure.Skills != nil {
		d.BaseSkills = pl.structure.Skills.Items
	}
	if pl.structure.Profile != nil && edits.Profile != nil {
		d.Profile = &checks.Slot{ID: pl.structure.Profile.ID, Text: *edits.Profile, BaseText: pl.structure.Profile.Text}
	}
	byID := make(map[string]planned, len(pl.positions))
	for _, p := range pl.positions {
		byID[p.ID] = p
	}
	for _, pe := range edits.Positions {
		p := byID[pe.PositionID]
		cp := checks.Position{ID: p.ID}
		for _, a := range p.Achievements {
			cp.Achievements = append(cp.Achievements, a.Text)
		}
		for i, b := range pe.Bullets {
			slot := checks.Slot{ID: p.slotIDs[i], Text: b.Text, BaseText: p.SlotTexts[i]}
			for _, id := range b.AchievementIDs {
				slot.Cited = append(slot.Cited, p.texts[id])
			}
			cp.Bullets = append(cp.Bullets, slot)
		}
		d.Positions = append(d.Positions, cp)
	}
	return d
}

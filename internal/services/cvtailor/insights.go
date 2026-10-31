package cvtailor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
)

func (s *Service) DraftInsights(ctx context.Context, userID string, q dto.TailoringDraftQuery) (dto.DraftInsights, error) {
	draft, err := s.store.GetDraft(ctx, userID, q.ID)
	if err != nil {
		return dto.DraftInsights{}, err
	}
	return s.storedInsights(ctx, userID, draft)
}

func (s *Service) storedInsights(ctx context.Context, userID string, draft dto.Draft) (dto.DraftInsights, error) {
	if len(draft.InsightLines) == 0 {
		return dto.DraftInsights{Lines: []dto.InsightLine{}}, nil
	}
	var lines []dto.InsightLine
	if err := json.Unmarshal(draft.InsightLines, &lines); err != nil {
		return dto.DraftInsights{}, fmt.Errorf("decode insights: %w", err)
	}
	in, err := s.insightsInput(ctx, userID, draft)
	if err != nil {
		return dto.DraftInsights{}, err
	}
	hash, err := insightsHash(in)
	if err != nil {
		return dto.DraftInsights{}, err
	}
	return dto.DraftInsights{Lines: lines, Stale: hash != draft.InsightsFor}, nil
}

func (s *Service) withInsightsSummary(ctx context.Context, userID string, draft dto.Draft) (dto.Draft, error) {
	if len(draft.InsightLines) == 0 {
		return draft, nil
	}
	stored, err := s.storedInsights(ctx, userID, draft)
	if err != nil {
		return dto.Draft{}, err
	}
	summary := dto.InsightsSummary{Total: len(stored.Lines), Stale: stored.Stale}
	for _, l := range stored.Lines {
		if l.Anchored {
			summary.Anchored++
		}
	}
	draft.Insights = &summary
	return draft, nil
}

func (m *Module) ExplainDraft(ctx context.Context, userID string, in dto.ExplainDraftInput) (dto.DraftInsights, error) {
	draft, err := m.store.GetDraft(ctx, userID, in.ID)
	if err != nil {
		return dto.DraftInsights{}, err
	}
	if draft.Status != statusReady || len(draft.EditSet) == 0 {
		return dto.DraftInsights{}, apperr.Conflict("only a ready draft can be explained")
	}
	key, err := m.creds.Get(ctx, userID, jev.Provider)
	if errors.Is(err, data.ErrNotFound) {
		return dto.DraftInsights{}, apperr.Unprocessable("connect an OpenRouter key in Settings, AI to get explanations")
	}
	if err != nil {
		return dto.DraftInsights{}, fmt.Errorf("load credential: %w", err)
	}
	input, err := m.svc.insightsInput(ctx, userID, draft)
	if err != nil {
		return dto.DraftInsights{}, err
	}
	if len(input.Bullets) == 0 {
		return dto.DraftInsights{}, apperr.Conflict("this draft has no bullets to explain")
	}
	hash, err := insightsHash(input)
	if err != nil {
		return dto.DraftInsights{}, err
	}
	leans, err := m.svc.DraftLean(ctx, userID, dto.DraftLeanQuery(in))
	if err != nil {
		return dto.DraftInsights{}, err
	}
	leanOf := make(map[string]*float64, len(leans))
	for _, l := range leans {
		leanOf[l.SlotID] = l.Lean
	}
	for i := range input.Bullets {
		input.Bullets[i].Lean = leanOf[input.Bullets[i].SlotID]
	}

	res, err := m.editor.Insights(ctx, key, input)
	if err != nil {
		m.log.ErrorContext(ctx, "explain draft failed", slog.String("draft_id", in.ID), slog.Any(logger.KeyErr, err))
		return dto.DraftInsights{}, suggestError(err)
	}

	lines := anchorInsights(input, res.Lines)
	raw, err := json.Marshal(lines)
	if err != nil {
		return dto.DraftInsights{}, fmt.Errorf("encode insights: %w", err)
	}
	if err := m.store.SetDraftInsights(ctx, userID, in.ID, raw, hash); err != nil {
		return dto.DraftInsights{}, err
	}
	return dto.DraftInsights{Lines: lines}, nil
}

func (s *Service) insightsInput(ctx context.Context, userID string, draft dto.Draft) (cvedit.InsightsInput, error) {
	var edits cvedit.EditSet
	if err := json.Unmarshal(draft.EditSet, &edits); err != nil {
		return cvedit.InsightsInput{}, fmt.Errorf("decode edit set: %w", err)
	}
	description, err := s.store.JobDescription(ctx, draft.JobID)
	if err != nil {
		return cvedit.InsightsInput{}, err
	}
	notes, err := s.store.JobAnnotations(ctx, userID, draft.JobID)
	if err != nil {
		return cvedit.InsightsInput{}, err
	}
	in := cvedit.InsightsInput{JobDescription: description, JobNote: notes.Note, JobComments: notes.Comments}
	for _, p := range edits.Positions {
		for _, l := range p.Lines() {
			in.Bullets = append(in.Bullets, cvedit.InsightBullet{SlotID: l.SlotID, Text: l.Text})
		}
	}
	return in, nil
}

func insightsHash(in cvedit.InsightsInput) (string, error) {
	bullets := make([]cvedit.InsightBullet, len(in.Bullets))
	for i, b := range in.Bullets {
		bullets[i] = cvedit.InsightBullet{SlotID: b.SlotID, Text: b.Text}
	}
	in.Bullets = bullets
	raw, err := json.Marshal(in)
	if err != nil {
		return "", fmt.Errorf("hash insights input: %w", err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func anchorInsights(in cvedit.InsightsInput, got []cvedit.InsightLine) []dto.InsightLine {
	bySlot := make(map[string]cvedit.InsightLine, len(got))
	for _, l := range got {
		if _, seen := bySlot[l.SlotID]; !seen {
			bySlot[l.SlotID] = l
		}
	}
	out := make([]dto.InsightLine, len(in.Bullets))
	for i, b := range in.Bullets {
		l := bySlot[b.SlotID]
		source, known := insightSource(l.Source)
		out[i] = dto.InsightLine{
			SlotID: b.SlotID, Lean: b.Lean, Source: source, Quote: l.Quote, Why: l.Why,
			Anchored: known && quoteAnchored(in, source, l.Quote),
		}
	}
	return out
}

func insightSource(s string) (dto.InsightSource, bool) {
	switch src := dto.InsightSource(s); src {
	case dto.InsightSourceJD, dto.InsightSourceNote, dto.InsightSourceComment:
		return src, true
	}
	return dto.InsightSourceJD, false
}

func quoteAnchored(in cvedit.InsightsInput, source dto.InsightSource, quote string) bool {
	q := collapseSpace(quote)
	if q == "" {
		return false
	}
	switch source {
	case dto.InsightSourceJD:
		return strings.Contains(collapseSpace(in.JobDescription), q)
	case dto.InsightSourceNote:
		return strings.Contains(collapseSpace(in.JobNote), q)
	case dto.InsightSourceComment:
		for _, c := range in.JobComments {
			if strings.Contains(collapseSpace(c), q) {
				return true
			}
		}
	}
	return false
}

func collapseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

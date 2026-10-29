package cvtailor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"slices"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/lineedit"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
)

const suggestBuffer = 16

// SuggestEvent is one step of a suggestion stream: a Delta, then either Done
// or Err.
type SuggestEvent struct {
	Delta string
	Done  *dto.SuggestDone
	Err   error
}

// Suggest streams a model rewrite of one slot's text. Everything that can be
// refused is refused, and the upstream is opened, before the first event, so
// the caller can still answer with an error. Cancelling ctx aborts upstream.
func (m *Module) Suggest(ctx context.Context, userID string, in dto.SuggestInput) (iter.Seq[SuggestEvent], error) {
	req := lineedit.Request{
		Action: in.Action, Prompt: in.Prompt, Quote: in.Quote, Text: in.Text, MaxChars: in.MaxChars, SlotID: in.SlotID,
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	evidence, err := m.suggestEvidence(ctx, userID, in)
	if err != nil {
		return nil, err
	}
	key, err := m.creds.Get(ctx, userID, jev.Provider)
	if errors.Is(err, data.ErrNotFound) {
		return nil, apperr.Unprocessable("connect an OpenRouter key in Settings, AI to get suggestions")
	}
	if err != nil {
		return nil, fmt.Errorf("load credential: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	events := make(chan SuggestEvent, suggestBuffer)
	go m.streamSuggestion(ctx, events, key, in.ID, req, evidence)

	first, ok := <-events
	if ok && first.Err != nil {
		cancel()
		return nil, first.Err
	}
	return func(yield func(SuggestEvent) bool) {
		defer cancel()
		if ok && !yield(first) {
			return
		}
		for ev := range events {
			if !yield(ev) {
				return
			}
		}
	}, nil
}

func (m *Module) streamSuggestion(ctx context.Context, events chan<- SuggestEvent, key, draftID string, req lineedit.Request, ev lineedit.Evidence) {
	defer close(events)
	send := func(e SuggestEvent) {
		select {
		case events <- e:
		case <-ctx.Done():
		}
	}
	res, err := m.lines.Rewrite(ctx, key, req, ev, func(delta string) { send(SuggestEvent{Delta: delta}) })
	slog.InfoContext(ctx, "draft suggestion", slog.String("draft_id", draftID), slog.String("action", req.Action),
		slog.String("model", res.Model), slog.String("prompt_key", res.PromptKey),
		slog.Float64("cost", res.Cost), slog.Any(logger.KeyErr, err))
	if err == nil {
		err = res.Err()
	}
	if err != nil {
		send(SuggestEvent{Err: err})
		return
	}
	send(SuggestEvent{Done: &dto.SuggestDone{Text: res.Text, Findings: toDraftFindings(res.Findings)}})
}

func (m *Module) suggestEvidence(ctx context.Context, userID string, in dto.SuggestInput) (lineedit.Evidence, error) {
	draft, err := m.store.GetDraft(ctx, userID, in.ID)
	if err != nil {
		return lineedit.Evidence{}, err
	}
	if draft.Status != statusReady || len(draft.EditSet) == 0 {
		return lineedit.Evidence{}, apperr.Conflict("only a ready draft can be edited")
	}
	if outcome(draft) == dto.OutcomeDiscarded {
		return lineedit.Evidence{}, apperr.Conflict("this draft was discarded")
	}
	var edits cvedit.EditSet
	if err := json.Unmarshal(draft.EditSet, &edits); err != nil {
		return lineedit.Evidence{}, fmt.Errorf("decode edit set: %w", err)
	}
	positions, err := m.store.ListPositions(ctx, userID)
	if err != nil {
		return lineedit.Evidence{}, err
	}
	bank := indexBank(positions, draft.AchievementIDs)

	if in.SlotID == profileSlotID {
		if !hasProfile(draft, edits) {
			return lineedit.Evidence{}, apperr.NotFound("unknown slot")
		}
		return lineedit.Evidence{Cited: bank.chosen, Bank: bank.all, Profile: true}, nil
	}
	for _, pe := range edits.Positions {
		for _, b := range pe.Bullets {
			if b.SlotID != in.SlotID {
				continue
			}
			scope := bank.chosenByPosition[pe.PositionID]
			cited := bank.texts(b.AchievementIDs)
			if len(cited) == 0 {
				cited = scope
			}
			return lineedit.Evidence{Cited: cited, Scope: scope, Bank: bank.all}, nil
		}
	}
	return lineedit.Evidence{}, apperr.NotFound("unknown slot")
}

type bankTexts struct {
	all, chosen      []string
	byID             map[string]string
	chosenByPosition map[string][]string
}

func indexBank(positions []dto.Position, chosenIDs []string) bankTexts {
	b := bankTexts{byID: map[string]string{}, chosenByPosition: map[string][]string{}}
	for _, p := range positions {
		for _, a := range p.Achievements {
			b.byID[a.ID] = a.Text
			b.all = append(b.all, a.Text)
			if slices.Contains(chosenIDs, a.ID) {
				b.chosen = append(b.chosen, a.Text)
				b.chosenByPosition[p.ID] = append(b.chosenByPosition[p.ID], a.Text)
			}
		}
	}
	return b
}

func (b bankTexts) texts(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if t, ok := b.byID[id]; ok {
			out = append(out, t)
		}
	}
	return out
}

func hasProfile(draft dto.Draft, edits cvedit.EditSet) bool {
	if edits.Profile != nil {
		return true
	}
	var base baseContent
	return len(draft.BaseContent) > 0 && json.Unmarshal(draft.BaseContent, &base) == nil && base.Profile != nil
}

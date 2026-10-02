package cvtailor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/openrouter"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
)

const (
	maxSuggestChars = 2000
	suggestBuffer   = 16
)

var suggestActions = []string{"fit", "tighten", "verb", "ask"}

// SuggestEvent is one step of a suggestion stream: a Delta, then either Done
// or Err.
type SuggestEvent struct {
	Delta string
	Done  *dto.SuggestDone
	Err   error
}

type suggestTarget struct {
	slotID       string
	positionID   string
	achievements []string
	scope        []string
	bank         []string
}

// Suggest streams a model rewrite of one slot's text. Everything that can be
// refused is refused, and the upstream is opened, before the first event, so
// the caller can still answer with an error. Cancelling ctx aborts upstream.
func (m *Module) Suggest(ctx context.Context, userID string, in dto.SuggestInput) (iter.Seq[SuggestEvent], error) {
	in.Text = strings.TrimSpace(in.Text)
	if err := validateSuggest(in); err != nil {
		return nil, err
	}
	target, err := m.suggestTarget(ctx, userID, in)
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
	go m.streamSuggestion(ctx, events, key, in, target)

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

func (m *Module) streamSuggestion(ctx context.Context, events chan<- SuggestEvent, key string, in dto.SuggestInput, target suggestTarget) {
	defer close(events)
	send := func(ev SuggestEvent) {
		select {
		case events <- ev:
		case <-ctx.Done():
		}
	}
	res, err := m.editor.Suggest(ctx, key, cvedit.SuggestInput{
		Action: in.Action, Prompt: in.Prompt, Text: in.Text, MaxChars: in.MaxChars, Achievements: target.achievements,
	}, func(delta string) { send(SuggestEvent{Delta: delta}) })
	slog.InfoContext(ctx, "draft suggestion", slog.String("draft_id", in.ID), slog.String("action", in.Action),
		slog.String("model", cvedit.SuggestModel), slog.String("prompt_version", cvedit.SuggestPromptVersion),
		slog.Float64("cost", res.Cost), slog.Any(logger.KeyErr, err))
	if err != nil {
		send(SuggestEvent{Err: suggestError(err)})
		return
	}
	send(SuggestEvent{Done: &dto.SuggestDone{Text: res.Text, Findings: target.findings(res.Text)}})
}

func validateSuggest(in dto.SuggestInput) error {
	switch {
	case !slices.Contains(suggestActions, in.Action):
		return apperr.Invalid("unknown action")
	case in.Text == "" || utf8.RuneCountInString(in.Text) > maxSuggestChars:
		return apperr.Invalid("text must be one non-empty line")
	case in.Action == "fit" && in.MaxChars <= 0:
		return apperr.Invalid("fit needs a character limit")
	case in.Action == "ask" && strings.TrimSpace(in.Prompt) == "":
		return apperr.Invalid("ask needs a prompt")
	case utf8.RuneCountInString(in.Prompt) > maxSuggestChars:
		return apperr.Invalid("prompt is too long")
	}
	return nil
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

func (m *Module) suggestTarget(ctx context.Context, userID string, in dto.SuggestInput) (suggestTarget, error) {
	draft, err := m.store.GetDraft(ctx, userID, in.ID)
	if err != nil {
		return suggestTarget{}, err
	}
	if draft.Status != statusReady || len(draft.EditSet) == 0 {
		return suggestTarget{}, apperr.Conflict("only a ready draft can be edited")
	}
	if outcome(draft) == dto.OutcomeDiscarded {
		return suggestTarget{}, apperr.Conflict("this draft was discarded")
	}
	var edits cvedit.EditSet
	if err := json.Unmarshal(draft.EditSet, &edits); err != nil {
		return suggestTarget{}, fmt.Errorf("decode edit set: %w", err)
	}
	positions, err := m.store.ListPositions(ctx, userID)
	if err != nil {
		return suggestTarget{}, err
	}
	bank := indexBank(positions, draft.AchievementIDs)

	if in.SlotID == profileSlotID {
		if !hasProfile(draft, edits) {
			return suggestTarget{}, apperr.NotFound("unknown slot")
		}
		return suggestTarget{slotID: profileSlotID, achievements: bank.chosen, bank: bank.all}, nil
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
			return suggestTarget{slotID: b.SlotID, positionID: pe.PositionID, achievements: cited, scope: scope, bank: bank.all}, nil
		}
	}
	return suggestTarget{}, apperr.NotFound("unknown slot")
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
	var base dto.DraftContent
	return len(draft.BaseContent) > 0 && json.Unmarshal(draft.BaseContent, &base) == nil && base.Profile != nil
}

func (t suggestTarget) findings(text string) []dto.DraftFinding {
	d := checks.Draft{Bank: t.bank}
	if t.slotID == profileSlotID {
		d.Profile = &checks.Slot{ID: t.slotID, Text: text}
	} else {
		slot := checks.Slot{ID: t.slotID, Text: text, Cited: t.achievements}
		d.Positions = []checks.Position{{ID: t.positionID, Achievements: t.scope, Bullets: []checks.Slot{slot}}}
	}
	return toDraftFindings(append(checks.Grounding(d), checks.BannedWords(d)...))
}

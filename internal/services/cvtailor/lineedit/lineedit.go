package lineedit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
)

const maxChars = 2000

type Prompt struct{ Model, System, User string }

func (p Prompt) Key() string {
	h := sha256.New()
	for _, part := range []string{p.Model, p.System, p.User} {
		_, _ = fmt.Fprintf(h, "%d:%s", len(part), part)
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

type Completion struct {
	Text string
	Cost float64
}

type Model interface {
	Complete(ctx context.Context, apiKey string, p Prompt, onDelta func(string)) (Completion, error)
}

type Editor struct{ model Model }

func New(model Model) *Editor { return &Editor{model: model} }

type Request struct {
	Action, Prompt, Quote, Text, SlotID string
	MaxChars                            int
}

func (r *Request) Validate() error {
	r.Text = strings.TrimSpace(r.Text)
	switch {
	case !slices.Contains(actionNames, r.Action):
		return apperr.Invalid("unknown action")
	case r.Text == "" || utf8.RuneCountInString(r.Text) > maxChars:
		return apperr.Invalid("text must be one non-empty line")
	case r.Action == "fit" && r.MaxChars <= 0:
		return apperr.Invalid("fit needs a character limit")
	case r.Action == "ask" && strings.TrimSpace(r.Prompt) == "":
		return apperr.Invalid("ask needs a prompt")
	case r.Action == actionTailor && (strings.TrimSpace(r.Quote) == "" || utf8.RuneCountInString(r.Quote) > maxChars):
		return apperr.Invalid("tailor needs a quote")
	case utf8.RuneCountInString(r.Prompt) > maxChars:
		return apperr.Invalid("prompt is too long")
	}
	return nil
}

type Evidence struct {
	Cited, Scope, Bank []string
	Profile            bool
}

type Result struct {
	Text, Rejection string
	Findings        []checks.Finding
	Cost            float64
	Model           string
	PromptKey       string
}

func (r Result) Err() error {
	if r.Rejection == "" {
		return nil
	}
	return apperr.Unprocessable(r.Rejection + "; try again")
}

func (e *Editor) Rewrite(ctx context.Context, apiKey string, req Request, ev Evidence, onDelta func(string)) (Result, error) {
	if err := req.Validate(); err != nil {
		return Result{}, err
	}
	a := actions[req.Action]
	p := Prompt{Model: editModel, System: systemPrompt, User: a.render(req) + lineBlock(req.Text, ev.Cited)}
	res := Result{Model: p.Model, PromptKey: p.Key()}

	c, err := e.model.Complete(ctx, apiKey, p, onDelta)
	res.Cost = c.Cost
	if err != nil {
		return res, modelError(err)
	}
	res.Text = strings.TrimSpace(c.Text)
	res.Rejection = a.reject(req, res.Text)
	res.Findings = findings(req, ev, res.Text)
	return res, nil
}

func lineBlock(text string, cited []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\nLine:\n%s\n\nAchievements it may draw on:\n", text)
	for _, a := range cited {
		fmt.Fprintf(&b, "- %s\n", a)
	}
	return b.String()
}

func findings(req Request, ev Evidence, text string) []checks.Finding {
	d := checks.Draft{Bank: ev.Bank}
	if ev.Profile {
		d.Profile = &checks.Slot{ID: req.SlotID, Text: text, BaseText: req.Text}
	} else {
		slot := checks.Slot{ID: req.SlotID, Text: text, BaseText: req.Text, Cited: ev.Cited}
		d.Positions = []checks.Position{{Achievements: ev.Scope, Bullets: []checks.Slot{slot}}}
	}
	return slices.Concat(checks.Grounding(d), checks.BannedWords(d))
}

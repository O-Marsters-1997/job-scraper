package lineedit_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/openrouter"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/lineedit"
)

type stubModel struct {
	reply   string
	err     error
	prompts []lineedit.Prompt
}

func (s *stubModel) Complete(_ context.Context, _ string, p lineedit.Prompt, onDelta func(string)) (lineedit.Completion, error) {
	s.prompts = append(s.prompts, p)
	onDelta(s.reply)
	return lineedit.Completion{Text: "  " + s.reply + "\n", Cost: 0.002}, s.err
}

func rewrite(t *testing.T, reply string, req lineedit.Request, ev lineedit.Evidence) lineedit.Result {
	t.Helper()
	res, err := lineedit.New(&stubModel{reply: reply}).Rewrite(t.Context(), "key", req, ev, func(string) {})
	if err != nil {
		t.Fatalf("Rewrite(%+v) error = %v", req, err)
	}
	return res
}

func TestRewriteEnforcesEachActionsRule(t *testing.T) {
	t.Parallel()
	const text = "Built REST services in Go for the payments team"
	ev := lineedit.Evidence{Cited: []string{text}}
	cases := []struct {
		name     string
		req      lineedit.Request
		reply    string
		rejected bool
	}{
		{"fit within the limit", lineedit.Request{Action: "fit", Text: text, MaxChars: 30}, "Built REST services in Go", false},
		{"fit over the limit", lineedit.Request{Action: "fit", Text: text, MaxChars: 10}, "Built REST services in Go", true},
		{"tighten that shrinks", lineedit.Request{Action: "tighten", Text: text}, "Built REST services in Go", false},
		{"tighten that grows", lineedit.Request{Action: "tighten", Text: text}, text + " every day", true},
		{"verb has no length rule", lineedit.Request{Action: "verb", Text: text}, "Shipped REST services in Go for the payments team, daily", false},
		{"ask has no length rule", lineedit.Request{Action: "ask", Prompt: "longer", Text: text}, text + " every day", false},
		{"tailor term swap", lineedit.Request{Action: "tailor", Quote: "microservices in Go", Text: text}, "Built microservices in Go for the payments team", false},
		{"tailor with more words", lineedit.Request{Action: "tailor", Quote: "q", Text: text}, text + " daily", true},
		{"tailor longer than max chars", lineedit.Request{Action: "tailor", Quote: "q", Text: text, MaxChars: 20}, "Built microservices in Go for the payments team", true},
		{"tailor that rewrites too much", lineedit.Request{Action: "tailor", Quote: "q", Text: text}, "Owned reliability of services for payments", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res := rewrite(t, tc.reply, tc.req, ev)

			if (res.Rejection != "") != tc.rejected {
				t.Errorf("Rewrite().Rejection = %q, want rejected %t", res.Rejection, tc.rejected)
			}
			if tc.rejected != (res.Err() != nil) {
				t.Errorf("Rewrite().Err() = %v, want error %t", res.Err(), tc.rejected)
			}
		})
	}
}

func TestResultErrIsAnUnprocessableTryAgain(t *testing.T) {
	t.Parallel()

	err := lineedit.Result{Rejection: "the line is too long"}.Err()

	if !apperr.IsKind(err, apperr.KindUnprocessable) || apperr.MessageFor(err) != "the line is too long; try again" {
		t.Errorf("Err() = %v, want unprocessable %q", err, "the line is too long; try again")
	}
	if err := (lineedit.Result{}).Err(); err != nil {
		t.Errorf("Err() with no rejection = %v, want nil", err)
	}
}

func TestRewriteChecksGroundingAndBannedWords(t *testing.T) {
	t.Parallel()
	const text = "Cut p99 latency by moving queries to Postgres"
	req := lineedit.Request{Action: "ask", Prompt: "punchier", Text: text, SlotID: "s7"}
	cases := []struct {
		name  string
		reply string
		ev    lineedit.Evidence
		want  []string
	}{
		{"grounded and clean", "Cut p99 latency using Postgres", lineedit.Evidence{Cited: []string{text}, Scope: []string{text}}, nil},
		{"a number the Achievements lack", "Cut p99 latency 40% using Postgres", lineedit.Evidence{Cited: []string{text}, Scope: []string{text}}, []string{"grounding"}},
		{"a banned word", "Leveraged Postgres to cut p99 latency", lineedit.Evidence{Cited: []string{text}, Scope: []string{text}}, []string{"banned_words"}},
		{"a term only the scope has", "Cut p99 latency using Kafka", lineedit.Evidence{Cited: []string{text}, Scope: []string{text, "Ran Kafka"}}, nil},
		{"a term only the bank has", "Cut p99 latency using Kafka", lineedit.Evidence{Cited: []string{text}, Scope: []string{text}, Bank: []string{"Ran Kafka"}}, []string{"grounding"}},
		{"the profile draws terms from the bank", "Cut p99 latency using Kafka", lineedit.Evidence{Profile: true, Bank: []string{text, "Ran Kafka"}}, nil},
		{"the profile still blocks an unknown term", "Cut p99 latency using Redis", lineedit.Evidence{Profile: true, Bank: []string{text}}, []string{"grounding"}},
		{"an unchanged line has no findings", text, lineedit.Evidence{Cited: nil}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res := rewrite(t, tc.reply, req, tc.ev)

			var got []string
			for _, f := range res.Findings {
				got = append(got, f.Check)
				if f.SlotID != "s7" {
					t.Errorf("finding %+v is not on slot s7", f)
				}
			}
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Errorf("Rewrite(%q) findings = %v, want %v", tc.reply, got, tc.want)
			}
		})
	}
}

func TestRewriteBuildsThePromptAndReportsItsKey(t *testing.T) {
	t.Parallel()
	model := &stubModel{reply: "  Cut latency  "}
	var deltas []string

	res, err := lineedit.New(model).Rewrite(t.Context(), "key",
		lineedit.Request{Action: "fit", Text: "Cut p99 latency by moving queries", MaxChars: 30},
		lineedit.Evidence{Cited: []string{"Cut p99 latency by moving queries"}}, func(d string) { deltas = append(deltas, d) })

	if err != nil {
		t.Fatalf("Rewrite() error = %v", err)
	}
	p := model.prompts[0]
	if res.PromptKey != p.Key() || res.Model != p.Model || p.Model == "" {
		t.Errorf("Result key/model = %q/%q, want the prompt's %q/%q", res.PromptKey, res.Model, p.Key(), p.Model)
	}
	if res.Text != "Cut latency" || res.Cost != 0.002 || len(deltas) != 1 {
		t.Errorf("Result = %+v, deltas = %v, want trimmed text, the cost and the streamed delta", res, deltas)
	}
	if !strings.Contains(p.System, "You rewrite the bullets of an existing CV") || !strings.Contains(p.System, "Write exactly one line") {
		t.Errorf("system prompt = %q, want voice.md and suggest.md", p.System)
	}
	for _, want := range []string{"at most 30 characters", "Cut p99 latency by moving queries", "- Cut p99 latency by moving queries"} {
		if !strings.Contains(p.User, want) {
			t.Errorf("user prompt = %q, want it to contain %q", p.User, want)
		}
	}
}

func TestPromptKeyChangesWithEveryField(t *testing.T) {
	t.Parallel()
	base := lineedit.Prompt{Model: "m", System: "s", User: "u"}
	for name, p := range map[string]lineedit.Prompt{
		"model":            {Model: "m2", System: "s", User: "u"},
		"system":           {Model: "m", System: "s2", User: "u"},
		"user":             {Model: "m", System: "s", User: "u2"},
		"shifted boundary": {Model: "ms", System: "", User: "u"},
	} {
		if p.Key() == base.Key() {
			t.Errorf("Key() unchanged when the %s changes", name)
		}
	}
}

func TestRewriteMapsModelFailures(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  error
		kind apperr.Kind
	}{
		{"a refused key", &openrouter.StatusError{Code: http.StatusUnauthorized}, apperr.KindUnprocessable},
		{"no credit", &openrouter.StatusError{Code: http.StatusPaymentRequired}, apperr.KindUnprocessable},
		{"forbidden", &openrouter.StatusError{Code: http.StatusForbidden}, apperr.KindUnprocessable},
		{"an upstream outage", &openrouter.StatusError{Code: http.StatusBadGateway}, apperr.KindUpstream},
		{"a transport error", context.DeadlineExceeded, apperr.KindUpstream},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res, err := lineedit.New(&stubModel{err: tc.err}).Rewrite(t.Context(), "key",
				lineedit.Request{Action: "tighten", Text: "x"}, lineedit.Evidence{}, func(string) {})

			if !apperr.IsKind(err, tc.kind) {
				t.Errorf("Rewrite() error = %v, want kind %v", err, tc.kind)
			}
			if res.Cost != 0.002 || res.PromptKey == "" {
				t.Errorf("Rewrite() result = %+v, want the cost and key kept for logging", res)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()
	ok := lineedit.Request{Action: "tighten", Text: "Cut p99 latency"}
	long := strings.Repeat("x", 2001)
	cases := []struct {
		name   string
		mutate func(*lineedit.Request)
		valid  bool
	}{
		{"a tighten", func(*lineedit.Request) {}, true},
		{"an unknown action", func(r *lineedit.Request) { r.Action = "rewrite" }, false},
		{"blank text", func(r *lineedit.Request) { r.Text = " \n" }, false},
		{"text over the cap", func(r *lineedit.Request) { r.Text = long }, false},
		{"fit without a limit", func(r *lineedit.Request) { r.Action = "fit" }, false},
		{"fit with a limit", func(r *lineedit.Request) { r.Action, r.MaxChars = "fit", 40 }, true},
		{"ask without a prompt", func(r *lineedit.Request) { r.Action = "ask" }, false},
		{"ask with a prompt", func(r *lineedit.Request) { r.Action, r.Prompt = "ask", "punchier" }, true},
		{"tailor without a quote", func(r *lineedit.Request) { r.Action = "tailor" }, false},
		{"tailor with a long quote", func(r *lineedit.Request) { r.Action, r.Quote = "tailor", long }, false},
		{"tailor with a quote", func(r *lineedit.Request) { r.Action, r.Quote = "tailor", "Go" }, true},
		{"a prompt over the cap", func(r *lineedit.Request) { r.Prompt = long }, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := ok
			tc.mutate(&req)

			err := req.Validate()

			if tc.valid && err != nil || !tc.valid && !apperr.IsKind(err, apperr.KindInvalid) {
				t.Errorf("Validate(%s) = %v, want valid %t", tc.name, err, tc.valid)
			}
		})
	}
}

func TestValidateTrimsTheText(t *testing.T) {
	t.Parallel()
	req := lineedit.Request{Action: "verb", Text: "  Cut p99 latency \n"}

	if err := req.Validate(); err != nil || req.Text != "Cut p99 latency" {
		t.Errorf("Validate() = %v, Text = %q, want it trimmed", err, req.Text)
	}
}

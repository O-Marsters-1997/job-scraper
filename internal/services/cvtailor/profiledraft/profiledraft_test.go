package profiledraft_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/lineedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/profiledraft"
)

type scriptedModel struct {
	replies []string
	err     error
	prompts []lineedit.Prompt
}

func (m *scriptedModel) Complete(_ context.Context, _ string, p lineedit.Prompt, _ func(string)) (lineedit.Completion, error) {
	m.prompts = append(m.prompts, p)
	if m.err != nil {
		return lineedit.Completion{}, m.err
	}
	reply := m.replies[min(len(m.prompts), len(m.replies))-1]
	return lineedit.Completion{Text: reply + "\n", Cost: 0.01}, nil
}

func input() profiledraft.Input {
	return profiledraft.Input{
		Profile:        "Backend engineer building Go services for payments teams.",
		BankSkills:     []string{"Go", "PostgreSQL"},
		Bank:           []string{"Built Go billing services", "Secret achievement text about Kubernetes migration"},
		JobTitle:       "Platform Engineer",
		Company:        "Acme",
		JobDescription: "<p>Own <b>billing</b> services.</p>",
		Steer:          "shorter",
	}
}

func TestWriterRedraft(t *testing.T) {
	t.Parallel()
	const clean = "Backend engineer owning Go billing services."

	t.Run("clean first output is accepted in one attempt", func(t *testing.T) {
		t.Parallel()
		m := &scriptedModel{replies: []string{clean}}
		res, err := profiledraft.New(m).Redraft(t.Context(), "key", input())
		if err != nil {
			t.Fatalf("Redraft error = %v", err)
		}
		if res.Text != clean || len(res.Attempts) != 1 {
			t.Errorf("Redraft = %q with %d attempts, want %q with 1", res.Text, len(res.Attempts), clean)
		}
		if a := res.Attempts[0]; a.Steer != "shorter" || a.PromptKey == "" || a.Model == "" || a.Cost == 0 {
			t.Errorf("attempt = %+v, want steer, prompt key, model and cost recorded", a)
		}
	})

	t.Run("a guardrail break is retried with the findings", func(t *testing.T) {
		t.Parallel()
		m := &scriptedModel{replies: []string{"Backend engineer who cut latency 40% with Go.", clean}}
		res, err := profiledraft.New(m).Redraft(t.Context(), "key", input())
		if err != nil {
			t.Fatalf("Redraft error = %v", err)
		}
		if res.Text != clean || len(res.Attempts) != 2 {
			t.Fatalf("Redraft = %q with %d attempts, want %q with 2", res.Text, len(res.Attempts), clean)
		}
		if res.Attempts[0].Rejection == "" || res.Attempts[1].Rejection != "" {
			t.Errorf("rejections = %q, %q, want first only", res.Attempts[0].Rejection, res.Attempts[1].Rejection)
		}
		if retry := m.prompts[1].User; !strings.Contains(retry, `number "40%"`) {
			t.Errorf("retry prompt = %q, want it to carry the findings", retry)
		}
	})

	t.Run("a term from the base profile is grounded, one from nowhere is not", func(t *testing.T) {
		t.Parallel()
		in := input()
		in.Profile = "Backend engineer building HealthTech Go services for payments teams."
		m := &scriptedModel{replies: []string{"Backend engineer owning HealthTech Go services."}}
		res, err := profiledraft.New(m).Redraft(t.Context(), "key", in)
		if err != nil {
			t.Fatalf("Redraft error = %v", err)
		}
		if res.Text == "" || len(res.Attempts) != 1 {
			t.Errorf("Redraft = %q with %d attempts, want accepted in 1", res.Text, len(res.Attempts))
		}

		m = &scriptedModel{replies: []string{"Backend engineer owning FinTech Go services."}}
		res, err = profiledraft.New(m).Redraft(t.Context(), "key", in)
		if err != nil {
			t.Fatalf("Redraft error = %v", err)
		}
		if res.Text != "" || len(res.Attempts) == 0 || !strings.Contains(res.Attempts[0].Rejection, `"FinTech"`) {
			t.Errorf("Redraft = %q, rejection %q, want empty and FinTech rejected", res.Text, res.Attempts[0].Rejection)
		}
	})

	t.Run("two bad outputs leave the text empty", func(t *testing.T) {
		t.Parallel()
		m := &scriptedModel{replies: []string{"First paragraph.\n\nSecond paragraph.", "Passionate Go engineer."}}
		res, err := profiledraft.New(m).Redraft(t.Context(), "key", input())
		if err != nil {
			t.Fatalf("Redraft error = %v", err)
		}
		if res.Text != "" || len(res.Attempts) != 2 || len(m.prompts) != 2 {
			t.Errorf("Redraft = %q with %d attempts and %d calls, want empty with 2 and 2", res.Text, len(res.Attempts), len(m.prompts))
		}
	})

	t.Run("more characters on the same lines passes", func(t *testing.T) {
		t.Parallel()
		in := input()
		in.Lines = func(string) int { return 2 }
		longer := clean + " Also Go and PostgreSQL."
		m := &scriptedModel{replies: []string{longer}}
		res, err := profiledraft.New(m).Redraft(t.Context(), "key", in)
		if err != nil {
			t.Fatalf("Redraft error = %v", err)
		}
		if res.Text != longer || res.Overlong || len(res.Attempts) != 1 {
			t.Errorf("Redraft = %q overlong=%v with %d attempts, want accepted in 1", res.Text, res.Overlong, len(res.Attempts))
		}
	})

	t.Run("over the line budget is shortened then returned flagged", func(t *testing.T) {
		t.Parallel()
		in := input()
		in.Lines = func(s string) int { return len(strings.Fields(s)) }
		first := strings.Repeat("Go ", 12)
		last := strings.Repeat("Go ", 11)
		m := &scriptedModel{replies: []string{first, last}}
		res, err := profiledraft.New(m).Redraft(t.Context(), "key", in)
		if err != nil {
			t.Fatalf("Redraft error = %v", err)
		}
		if res.Text != strings.TrimSpace(last) || !res.Overlong || len(res.Attempts) != 2 {
			t.Errorf("Redraft = %q overlong=%v with %d attempts, want the last text flagged after 2", res.Text, res.Overlong, len(res.Attempts))
		}
		if p := m.prompts[0].User; !strings.Contains(p, "at most 8 words") {
			t.Errorf("first prompt = %q, want a word budget", p)
		}
		if retry := m.prompts[1].User; !strings.Contains(retry, "Shorten that exact text") || !strings.Contains(retry, strings.TrimSpace(first)) {
			t.Errorf("retry prompt = %q, want it to shorten attempt 1", retry)
		}
	})

	t.Run("a grounding break on the last attempt still blocks", func(t *testing.T) {
		t.Parallel()
		in := input()
		in.Lines = func(s string) int { return len(strings.Fields(s)) }
		m := &scriptedModel{replies: []string{strings.Repeat("Go ", 12), "Backend engineer who cut latency 40% with Go."}}
		res, err := profiledraft.New(m).Redraft(t.Context(), "key", in)
		if err != nil {
			t.Fatalf("Redraft error = %v", err)
		}
		if res.Text != "" || res.Overlong {
			t.Errorf("Redraft = %q overlong=%v, want empty", res.Text, res.Overlong)
		}
	})

	t.Run("the prompt holds the inputs and no Achievement text", func(t *testing.T) {
		t.Parallel()
		in := input()
		in.JobDescription = "<p>" + strings.Repeat("x", 7000) + "</p>"
		m := &scriptedModel{replies: []string{clean}}
		if _, err := profiledraft.New(m).Redraft(t.Context(), "key", in); err != nil {
			t.Fatalf("Redraft error = %v", err)
		}
		p := m.prompts[0]
		for _, want := range []string{in.Profile, "- PostgreSQL", "Platform Engineer", "Acme", strings.Repeat("x", 6000)} {
			if !strings.Contains(p.User, want) {
				t.Errorf("prompt lacks %.40q", want)
			}
		}
		if strings.Contains(p.User, strings.Repeat("x", 6001)) || strings.Contains(p.User, "<p>") {
			t.Error("job description was not stripped and capped")
		}
		if strings.Contains(p.System+p.User, "Kubernetes") {
			t.Error("prompt carries Achievement text")
		}
	})

	t.Run("model failure is an error", func(t *testing.T) {
		t.Parallel()
		boom := errors.New("boom")
		_, err := profiledraft.New(&scriptedModel{err: boom}).Redraft(t.Context(), "key", input())
		if !errors.Is(err, boom) {
			t.Errorf("Redraft error = %v, want wrapping %v", err, boom)
		}
	})
}

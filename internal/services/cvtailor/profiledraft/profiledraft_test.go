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

	t.Run("longer than the base is rejected", func(t *testing.T) {
		t.Parallel()
		m := &scriptedModel{replies: []string{strings.Repeat("Go engineer. ", 10), strings.Repeat("Go engineer. ", 10)}}
		res, err := profiledraft.New(m).Redraft(t.Context(), "key", input())
		if err != nil {
			t.Fatalf("Redraft error = %v", err)
		}
		if res.Text != "" {
			t.Errorf("Redraft text = %q, want empty", res.Text)
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

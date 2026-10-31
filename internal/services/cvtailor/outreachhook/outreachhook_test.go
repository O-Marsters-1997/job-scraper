package outreachhook_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/lineedit"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/outreachhook"
)

type scriptedModel struct {
	reply  string
	err    error
	prompt lineedit.Prompt
}

func (m *scriptedModel) Complete(_ context.Context, _ string, p lineedit.Prompt, _ func(string)) (lineedit.Completion, error) {
	m.prompt = p
	return lineedit.Completion{Text: m.reply, Cost: 0.002}, m.err
}

func TestWriter_Write(t *testing.T) {
	t.Parallel()
	in := outreachhook.Input{
		JobTitle: "Platform Engineer", JobDescription: "Own billing services.", Company: "Acme",
		ContactRole: "CTO", Achievement: "cut p99 latency by 40%",
	}

	t.Run("the prompt holds the inputs and the reply is cleaned", func(t *testing.T) {
		t.Parallel()
		m := &scriptedModel{reply: "  \"Acme's billing work\n looks like my kind of problem.\"\n"}
		got, err := outreachhook.New(m).Write(t.Context(), "key", in)
		if err != nil {
			t.Fatalf("Write() error = %v", err)
		}
		if got.Text != "Acme's billing work looks like my kind of problem." || got.Cost != 0.002 {
			t.Errorf("Write() = %+v, want the cleaned line and its cost", got)
		}
		for _, want := range []string{"Platform Engineer", "Acme", "CTO", "Own billing services.", "cut p99 latency by 40%"} {
			if !strings.Contains(m.prompt.User, want) {
				t.Errorf("prompt lacks %q", want)
			}
		}
		if m.prompt.Model != outreachhook.Model {
			t.Errorf("model = %q, want %q", m.prompt.Model, outreachhook.Model)
		}
	})

	t.Run("a model failure is wrapped", func(t *testing.T) {
		t.Parallel()
		boom := errors.New("boom")
		_, err := outreachhook.New(&scriptedModel{err: boom}).Write(t.Context(), "key", in)
		if !errors.Is(err, boom) {
			t.Errorf("Write() error = %v, want wrapping %v", err, boom)
		}
	})
}

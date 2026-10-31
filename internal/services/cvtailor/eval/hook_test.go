package eval_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/eval"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/lineedit"
)

type fixedModel string

func (m fixedModel) Complete(context.Context, string, lineedit.Prompt, func(string)) (lineedit.Completion, error) {
	return lineedit.Completion{Text: string(m), Cost: 0.001}, nil
}

func TestHookFixtures(t *testing.T) {
	t.Parallel()
	all, err := eval.HookFixtures()
	if err != nil {
		t.Fatalf("HookFixtures() error = %v", err)
	}
	if len(all) < 4 {
		t.Errorf("HookFixtures() = %d fixtures, want at least 4", len(all))
	}
	for _, f := range all {
		if f.Scenario == "" || f.Company == "" || f.Achievement == "" {
			t.Errorf("hook fixture %s is incomplete", f.Name)
		}
	}
}

func TestRunHook(t *testing.T) {
	t.Parallel()
	f := eval.HookFixture{Name: "x", Company: "Acme", JobTitle: "Engineer", Achievement: "shipped it"}

	t.Run("a line in range naming the company passes", func(t *testing.T) {
		t.Parallel()
		line := "Acme " + strings.Repeat("builds things. ", 15)
		got, err := eval.RunHook(t.Context(), fixedModel(line), "key", f)
		if err != nil || !got.Passed() || got.PromptKey == "" {
			t.Errorf("RunHook() = %+v, %v, want a pass with a prompt key", got, err)
		}
	})

	t.Run("a banned word fails", func(t *testing.T) {
		t.Parallel()
		line := "Acme " + strings.Repeat("builds things. ", 15) + "Passionate."
		got, err := eval.RunHook(t.Context(), fixedModel(line), "key", f)
		if err != nil || got.Passed() {
			t.Errorf("RunHook() = %+v, %v, want a fail", got, err)
		}
	})
}

package scoringtest

import (
	"context"
	"errors"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
)

// RunStoreContract proves newStore's scoring.Store behaves the same whether
// it's the fake or the real store (ADR 0012). Cases needing a real foreign
// key, exact effect ordering or precise timing stay in store_test.go.
func RunStoreContract(t *testing.T, newStore func(t *testing.T) scoring.Store) {
	t.Helper()

	t.Run("claim on an empty queue returns not found", func(t *testing.T) {
		st := newStore(t)
		_, err := st.ClaimAnswerEffect(context.Background())
		if !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("ClaimAnswerEffect(...) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("get job for scoring on an unknown id returns not found", func(t *testing.T) {
		st := newStore(t)
		_, err := st.GetJobForScoring(context.Background(), missingID)
		if !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("GetJobForScoring(...) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("get search config on an unknown user returns not found", func(t *testing.T) {
		st := newStore(t)
		_, err := st.GetSearchConfig(context.Background(), missingID)
		if !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("GetSearchConfig(...) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("reword an unknown option returns not found", func(t *testing.T) {
		st := newStore(t)
		err := st.RewordScoringOption(context.Background(), missingID, "question")
		if !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("RewordScoringOption(...) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("retire an unknown option returns not found", func(t *testing.T) {
		st := newStore(t)
		err := st.RetireScoringOption(context.Background(), missingID)
		if !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("RetireScoringOption(...) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("add, reword then retire an option round trips", func(t *testing.T) {
		st := newStore(t)
		ctx := context.Background()
		if err := st.AddScoringOption(ctx, optionID, "tech", "Go", "Does the role use Go?"); err != nil {
			t.Fatalf("AddScoringOption(...) = %v", err)
		}
		options, err := st.ListScoringOptions(ctx)
		if err != nil || len(options) != 1 || options[0].Question != "Does the role use Go?" {
			t.Fatalf("ListScoringOptions(...) = %+v, %v, want one Go option", options, err)
		}

		if err := st.RewordScoringOption(ctx, optionID, "Does the role primarily use Go?"); err != nil {
			t.Fatalf("RewordScoringOption(...) = %v", err)
		}
		options, err = st.ListScoringOptions(ctx)
		if err != nil || options[0].Question != "Does the role primarily use Go?" {
			t.Fatalf("ListScoringOptions(...) = %+v, %v, want the reworded question", options, err)
		}

		if err := st.RetireScoringOption(ctx, optionID); err != nil {
			t.Fatalf("RetireScoringOption(...) = %v", err)
		}
		options, err = st.ListScoringOptions(ctx)
		if err != nil || options[0].RetiredAt == nil {
			t.Fatalf("ListScoringOptions(...) = %+v, %v, want retired_at set", options, err)
		}
	})

	t.Run("ops state on an empty queue is zero pending", func(t *testing.T) {
		st := newStore(t)
		state, err := st.OpsState(context.Background())
		if err != nil || state.OutboxPending != 0 {
			t.Fatalf("OpsState(...) = %+v, %v, want zero pending", state, err)
		}
	})
}

const (
	missingID = "00000000-0000-0000-0000-000000000000"
	optionID  = "tech:go"
)

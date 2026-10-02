package scoringtest

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
)

// Fixture is a store under test plus a way to mint users the store accepts:
// the real one needs a users row for each user_id (a foreign key).
type Fixture struct {
	Store   scoring.Store
	NewUser func() string
}

// RunStoreContract proves newStore's scoring.Store behaves the same whether
// it's the fake or the real store (ADR 0012). Cases needing a real foreign
// key, exact effect ordering or precise timing stay in store_test.go.
func RunStoreContract(t *testing.T, newFixture func(t *testing.T) Fixture) {
	t.Helper()

	newStore := func(t *testing.T) scoring.Store {
		t.Helper()
		return newFixture(t).Store
	}

	t.Run("claim on an empty queue returns not found", func(t *testing.T) {
		st := newStore(t)
		_, err := st.ClaimAnswerEffect(t.Context())
		if !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("ClaimAnswerEffect(...) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("get job for scoring on an unknown id returns not found", func(t *testing.T) {
		st := newStore(t)
		_, err := st.GetJobForScoring(t.Context(), missingID)
		if !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("GetJobForScoring(...) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("get search config on an unknown user returns not found", func(t *testing.T) {
		st := newStore(t)
		_, err := st.GetSearchConfig(t.Context(), missingID)
		if !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("GetSearchConfig(...) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("reword an unknown option returns not found", func(t *testing.T) {
		st := newStore(t)
		err := st.RewordScoringOption(t.Context(), missingID, "question")
		if !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("RewordScoringOption(...) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("retire an unknown option returns not found", func(t *testing.T) {
		st := newStore(t)
		err := st.RetireScoringOption(t.Context(), missingID)
		if !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("RetireScoringOption(...) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("add, reword then retire an option round trips", func(t *testing.T) {
		st := newStore(t)
		ctx := t.Context()
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

	t.Run("retire an already retired option returns not found", func(t *testing.T) {
		st := newStore(t)
		ctx := t.Context()
		if err := st.AddScoringOption(ctx, optionID, "tech", "Go", "Does the role use Go?"); err != nil {
			t.Fatalf("AddScoringOption(...) = %v", err)
		}
		if err := st.RetireScoringOption(ctx, optionID); err != nil {
			t.Fatalf("RetireScoringOption(...) = %v", err)
		}
		if err := st.RetireScoringOption(ctx, optionID); !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("RetireScoringOption(...) again err = %v, want ErrNotFound", err)
		}
	})

	t.Run("ops state on an empty store reports nothing", func(t *testing.T) {
		st := newStore(t)
		state, err := st.OpsState(t.Context())
		if err != nil {
			t.Fatalf("OpsState(...) error = %v", err)
		}
		if state.OutboxPending != 0 || state.BoardsOverdue != 0 || state.BoardsFailing != 0 ||
			state.SourceTargetsFailed != 0 || len(state.HarvestAge) != 0 {
			t.Errorf("OpsState(...) = %+v, want no pending effects, overdue or failing boards, failed targets or harvests", state)
		}
	})

	t.Run("score feedback lists newest first within the limit", func(t *testing.T) {
		f := newFixture(t)
		ctx := t.Context()
		user := f.NewUser()
		for _, reason := range []string{"first", "second", "third"} {
			entry := dto.ScoreFeedback{Kind: "overall", Reason: reason, Picks: []dto.Pick{}, Model: "m"}
			if _, err := f.Store.InsertScoreFeedback(ctx, user, entry); err != nil {
				t.Fatalf("InsertScoreFeedback(%q) = %v", reason, err)
			}
		}

		got, err := f.Store.ListScoreFeedback(ctx, user, 2)
		if err != nil {
			t.Fatalf("ListScoreFeedback(...) = %v", err)
		}
		var reasons []string
		for _, e := range got {
			reasons = append(reasons, e.Reason)
		}
		if diff := cmp.Diff([]string{"third", "second"}, reasons); diff != "" {
			t.Errorf("ListScoreFeedback(...) reasons (-want +got):\n%s", diff)
		}
	})

	t.Run("clear score feedback deletes only that user's rows and counts them", func(t *testing.T) {
		f := newFixture(t)
		ctx := t.Context()
		user, other := f.NewUser(), f.NewUser()
		entry := dto.ScoreFeedback{Kind: "overall", Reason: "r", Picks: []dto.Pick{}, Model: "m"}
		for _, u := range []string{user, user, other} {
			if _, err := f.Store.InsertScoreFeedback(ctx, u, entry); err != nil {
				t.Fatalf("InsertScoreFeedback(...) = %v", err)
			}
		}

		n, err := f.Store.ClearScoreFeedback(ctx, user)
		if err != nil || n != 2 {
			t.Fatalf("ClearScoreFeedback(...) = %d, %v, want 2, nil", n, err)
		}
		left, err := f.Store.ListScoreFeedback(ctx, other, 10)
		if err != nil || len(left) != 1 {
			t.Fatalf("ListScoreFeedback(other) = %+v, %v, want the other user's one entry", left, err)
		}
	})
}

const (
	missingID = "00000000-0000-0000-0000-000000000000"
	optionID  = "tech:go"
)

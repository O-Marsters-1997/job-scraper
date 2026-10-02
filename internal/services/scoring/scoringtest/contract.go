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

	t.Run("score feedback lists newest first, pages, and filters by kind", func(t *testing.T) {
		f := newFixture(t)
		ctx := t.Context()
		user := f.NewUser()
		for _, e := range []struct{ kind, reason string }{
			{"overall", "first"}, {"collection", "second"}, {"overall", "third"},
		} {
			entry := dto.ScoreFeedback{Kind: e.kind, Reason: e.reason, Picks: []dto.Pick{}, Model: "m"}
			if _, err := f.Store.InsertScoreFeedback(ctx, user, entry); err != nil {
				t.Fatalf("InsertScoreFeedback(%q) = %v", e.reason, err)
			}
		}
		reasonsOf := func(kind string, limit, offset int) []string {
			t.Helper()
			got, err := f.Store.ListScoreFeedback(ctx, user, dto.ScoreFeedbackFilter{Kind: kind, Model: "m"}, limit, offset)
			if err != nil {
				t.Fatalf("ListScoreFeedback(%q, %d, %d) = %v", kind, limit, offset, err)
			}
			var reasons []string
			for _, e := range got {
				reasons = append(reasons, e.Reason)
			}
			return reasons
		}

		if diff := cmp.Diff([]string{"third", "second"}, reasonsOf("", 2, 0)); diff != "" {
			t.Errorf("first page (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff([]string{"first"}, reasonsOf("", 2, 2)); diff != "" {
			t.Errorf("second page (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff([]string{"third", "first"}, reasonsOf("overall", 10, 0)); diff != "" {
			t.Errorf("overall only (-want +got):\n%s", diff)
		}
		if got := reasonsOf("", 2, 5); len(got) != 0 {
			t.Errorf("page past the end = %v, want empty", got)
		}
		for kind, want := range map[string]int{"": 3, "overall": 2, "collection": 1, "job": 0} {
			if n, _, err := f.Store.CountScoreFeedback(ctx, user, dto.ScoreFeedbackFilter{Kind: kind, Model: "m"}); err != nil || n != want {
				t.Errorf("CountScoreFeedback(%q) = %d, %v, want %d, nil", kind, n, err, want)
			}
		}
	})

	t.Run("score feedback flags drift from the current Picks and model", func(t *testing.T) {
		f := newFixture(t)
		ctx := t.Context()
		user := f.NewUser()
		goPick := []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "manual"}}
		cobolPick := []dto.Pick{{OptionID: "tech:cobol", Stance: "avoid", Source: "manual"}}
		setPicks := func(picks []dto.Pick) {
			t.Helper()
			cfg := dto.SearchConfig{UserID: user, NotifyThreshold: 70, Preferences: dto.Preferences{Picks: picks}}
			if _, err := f.Store.UpsertSearchConfig(ctx, cfg); err != nil {
				t.Fatalf("UpsertSearchConfig() = %v", err)
			}
		}
		insert := func(reason, model string) {
			t.Helper()
			entry := dto.ScoreFeedback{Kind: "overall", Reason: reason, Picks: goPick, Model: model}
			if _, err := f.Store.InsertScoreFeedback(ctx, user, entry); err != nil {
				t.Fatalf("InsertScoreFeedback(%q) = %v", reason, err)
			}
		}
		listed := func(includeOutdated bool) map[string][2]bool {
			t.Helper()
			filter := dto.ScoreFeedbackFilter{Model: "m", IncludeOutdated: includeOutdated}
			got, err := f.Store.ListScoreFeedback(ctx, user, filter, 10, 0)
			if err != nil {
				t.Fatalf("ListScoreFeedback(%+v) = %v", filter, err)
			}
			drift := map[string][2]bool{}
			for _, e := range got {
				drift[e.Reason] = [2]bool{e.PicksChanged, e.ModelChanged}
			}
			return drift
		}
		counts := func() [2]int {
			t.Helper()
			current, outdated, err := f.Store.CountScoreFeedback(ctx, user, dto.ScoreFeedbackFilter{Model: "m"})
			if err != nil {
				t.Fatalf("CountScoreFeedback() = %v", err)
			}
			return [2]int{current, outdated}
		}

		setPicks(goPick)
		insert("current", "m")
		insert("old model", "other")
		if diff := cmp.Diff(map[string][2]bool{"current": {false, false}}, listed(false)); diff != "" {
			t.Errorf("default list (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(map[string][2]bool{"current": {false, false}, "old model": {false, true}}, listed(true)); diff != "" {
			t.Errorf("list with outdated (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff([2]int{1, 1}, counts()); diff != "" {
			t.Errorf("counts after a model change (-want +got):\n%s", diff)
		}

		setPicks(cobolPick)
		if got := listed(false); len(got) != 0 {
			t.Errorf("default list after changing Picks = %v, want empty", got)
		}
		if diff := cmp.Diff(map[string][2]bool{"current": {true, false}, "old model": {true, true}}, listed(true)); diff != "" {
			t.Errorf("list with outdated after changing Picks (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff([2]int{0, 2}, counts()); diff != "" {
			t.Errorf("counts after changing Picks (-want +got):\n%s", diff)
		}
	})

	t.Run("delete score feedback removes only the caller's entry", func(t *testing.T) {
		f := newFixture(t)
		ctx := t.Context()
		user, other := f.NewUser(), f.NewUser()
		entry := dto.ScoreFeedback{Kind: "overall", Reason: "r", Picks: []dto.Pick{}, Model: "m"}
		mine, err := f.Store.InsertScoreFeedback(ctx, user, entry)
		if err != nil {
			t.Fatalf("InsertScoreFeedback(...) = %v", err)
		}

		if err := f.Store.DeleteScoreFeedback(ctx, other, mine.ID); !errors.Is(err, data.ErrNotFound) {
			t.Errorf("DeleteScoreFeedback(other user's id) = %v, want ErrNotFound", err)
		}
		if err := f.Store.DeleteScoreFeedback(ctx, user, missingID); !errors.Is(err, data.ErrNotFound) {
			t.Errorf("DeleteScoreFeedback(unknown id) = %v, want ErrNotFound", err)
		}
		if err := f.Store.DeleteScoreFeedback(ctx, user, mine.ID); err != nil {
			t.Fatalf("DeleteScoreFeedback(own id) = %v", err)
		}
		if n, _, _ := f.Store.CountScoreFeedback(ctx, user, dto.ScoreFeedbackFilter{Model: "m"}); n != 0 {
			t.Errorf("CountScoreFeedback after delete = %d, want 0", n)
		}
	})

	t.Run("job score for feedback missing returns not found", func(t *testing.T) {
		f := newFixture(t)
		_, err := f.Store.GetJobScoreForFeedback(t.Context(), f.NewUser(), missingID)
		if !errors.Is(err, data.ErrNotFound) {
			t.Errorf("GetJobScoreForFeedback(missing) err = %v, want ErrNotFound", err)
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
		left, err := f.Store.ListScoreFeedback(ctx, other, dto.ScoreFeedbackFilter{Model: "m"}, 10, 0)
		if err != nil || len(left) != 1 {
			t.Fatalf("ListScoreFeedback(other) = %+v, %v, want the other user's one entry", left, err)
		}
	})
}

const (
	missingID = "00000000-0000-0000-0000-000000000000"
	optionID  = "tech:go"
)

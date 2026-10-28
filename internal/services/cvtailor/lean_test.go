package cvtailor_test

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestService_DraftLean(t *testing.T) {
	t.Parallel()
	t.Run("a cited bullet carries the lean of its achievement", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		e.asker.answers = map[string]dto.Answer{
			questionPrefix + e.pos.Achievements[0].Text: {PYes: 0.6, PNo: 0.1},
		}

		got, err := e.svc.DraftLean(t.Context(), userID, dto.DraftLeanQuery{ID: id})
		if err != nil {
			t.Fatalf("DraftLean() error = %v", err)
		}

		if len(got) != 1 || got[0].AchievementID == nil || *got[0].AchievementID != e.pos.Achievements[0].ID {
			t.Fatalf("DraftLean() = %+v, want one bullet linked to the cited achievement", got)
		}
		if got[0].Lean == nil || math.Abs(*got[0].Lean-0.5) > 1e-9 || got[0].State != dto.SuggestionFit {
			t.Errorf("lean = %v, state = %q, want 0.5 and fit", got[0].Lean, got[0].State)
		}
	})

	t.Run("a replaced bullet carries the old text and its lean", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		e.asker.answers = map[string]dto.Answer{
			questionPrefix + e.pos.Achievements[0].Text:                              {PYes: 0.6, PNo: 0.1},
			questionPrefix + "Built and maintained the public APIs for the platform": {PYes: 0.05, PNo: 0.4},
		}

		got, err := e.svc.DraftLean(t.Context(), userID, dto.DraftLeanQuery{ID: id})
		if err != nil {
			t.Fatalf("DraftLean() error = %v", err)
		}

		if len(got) != 1 || got[0].ReplacedFrom == nil || *got[0].ReplacedFrom != "Built and maintained the public APIs for the platform" {
			t.Fatalf("DraftLean() = %+v, want the base text as ReplacedFrom", got)
		}
		if got[0].ReplacedLean == nil || math.Abs(*got[0].ReplacedLean+0.35) > 1e-9 {
			t.Errorf("ReplacedLean = %v, want -0.35", got[0].ReplacedLean)
		}
	})

	t.Run("an unused same-position achievement that outscores the bullet is a closer match", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		other := e.pos.Achievements[1]
		e.asker.answers = map[string]dto.Answer{
			questionPrefix + e.pos.Achievements[0].Text: {PYes: 0.4, PNo: 0.1},
			questionPrefix + other.Text:                 {PYes: 0.8, PNo: 0.06},
		}

		got, err := e.svc.DraftLean(t.Context(), userID, dto.DraftLeanQuery{ID: id})
		if err != nil {
			t.Fatalf("DraftLean() error = %v", err)
		}

		if len(got) != 1 || got[0].CloserMatch == nil || got[0].CloserMatch.AchievementID != other.ID || math.Abs(got[0].CloserMatch.Lean-0.74) > 1e-9 {
			t.Fatalf("DraftLean() = %+v, want the unused achievement at 0.74", got)
		}

		e.asker.answers[questionPrefix+other.Text] = dto.Answer{PYes: 0.2, PNo: 0.1}
		got, err = e.svc.DraftLean(t.Context(), userID, dto.DraftLeanQuery{ID: id})
		if err != nil {
			t.Fatalf("DraftLean() error = %v", err)
		}
		if got[0].CloserMatch != nil {
			t.Errorf("CloserMatch = %+v, want none when the bullet scores higher", got[0].CloserMatch)
		}
	})

	t.Run("a bullet matching no achievement has no lean", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		if err := e.store.DeleteAchievement(t.Context(), userID, e.pos.Achievements[0].ID); err != nil {
			t.Fatal(err)
		}

		got, err := e.svc.DraftLean(t.Context(), userID, dto.DraftLeanQuery{ID: id})
		if err != nil {
			t.Fatalf("DraftLean() error = %v", err)
		}

		if len(got) != 1 || got[0].AchievementID != nil || got[0].Lean != nil || got[0].State != "" {
			t.Fatalf("DraftLean() = %+v, want one unmatched bullet", got)
		}
		out, err := json.Marshal(got[0])
		if err != nil {
			t.Fatal(err)
		}
		var decoded map[string]any
		if err := json.Unmarshal(out, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded["state"] != nil || decoded["lean"] != nil || decoded["achievementId"] != nil {
			t.Errorf("JSON = %s, want null achievementId, lean and state", out)
		}
	})

	t.Run("a bullet whose citation is gone links by exact text", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		if err := e.store.DeleteAchievement(t.Context(), userID, e.pos.Achievements[0].ID); err != nil {
			t.Fatal(err)
		}
		a := addAchievements(t, e.pos.ID, e.store, "Cut p99 latency by quickly moving queries to Postgres")[0]

		got, err := e.svc.DraftLean(t.Context(), userID, dto.DraftLeanQuery{ID: id})
		if err != nil {
			t.Fatalf("DraftLean() error = %v", err)
		}

		if len(got) != 1 || got[0].AchievementID == nil || *got[0].AchievementID != a.ID {
			t.Errorf("DraftLean() = %+v, want the bullet linked to the achievement with its text", got)
		}
	})
}

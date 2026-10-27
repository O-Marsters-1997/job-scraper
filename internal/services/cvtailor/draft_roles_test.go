package cvtailor_test

import (
	"encoding/json"
	"flag"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/linefit"
)

var update = flag.Bool("update", false, "rewrite golden files")

const draftRolesGolden = "testdata/draft_roles.golden.json"

var sourceBullets = []string{"Built and maintained the public APIs for the platform", "Ran on-call", "Wrote docs"}

func sourceBudget() int {
	total := 0
	for _, text := range sourceBullets {
		total += linefit.Lines(text, linefit.Style{Font: "Arial", Size: 11}, 468)
	}
	return total
}

func TestService_DraftRoles(t *testing.T) {
	t.Parallel()
	t.Run("budget is the source bullets' lines and ignores the draft's edits", func(t *testing.T) {
		t.Parallel()
		var budgets, lines []int
		for _, text := range []string{"Cut p99 latency", strings.Repeat("Cut p99 latency by moving queries to Postgres. ", 12)} {
			e := newDraftEnv(t)
			id := e.readyWith(t, tick{}, e.citedEdit(text))
			got, err := e.svc.DraftRoles(t.Context(), userID, dto.DraftQuery{ID: id})
			if err != nil {
				t.Fatalf("DraftRoles(%s) error = %v", id, err)
			}
			if len(got.Roles) != 1 {
				t.Fatalf("DraftRoles(%s) = %+v, want one role", id, got)
			}
			budgets = append(budgets, got.Roles[0].Budget)
			lines = append(lines, got.Roles[0].Bullets[0].Lines)
		}

		if want := sourceBudget(); budgets[0] != want || budgets[1] != want {
			t.Errorf("budgets = %v, want both %d", budgets, want)
		}
		if lines[0] != 1 || lines[1] <= lines[0] {
			t.Errorf("bullet lines = %v, want the long edit to wrap past one line", lines)
		}
	})

	t.Run("bank lists unused achievements by lean, best first", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		extra := addAchievements(t, e.pos.ID, e.store, "Reduced cloud spend by a third")[0]
		id := e.readyDrafts(t, 1)[0]
		e.asker.answers = map[string]dto.Answer{
			questionPrefix + e.pos.Achievements[0].Text: {PYes: 0.9, PNo: 0},
			questionPrefix + e.pos.Achievements[1].Text: {PYes: 0.2, PNo: 0.1},
			questionPrefix + extra.Text:                 {PYes: 0.7, PNo: 0.1},
		}

		got, err := e.svc.DraftRoles(t.Context(), userID, dto.DraftQuery{ID: id})
		if err != nil {
			t.Fatalf("DraftRoles() error = %v", err)
		}

		var texts []string
		for _, b := range got.Roles[0].Bank {
			texts = append(texts, b.Text)
			if b.Lines < 1 || b.SlotID != "" || b.OnSource {
				t.Errorf("bank bullet = %+v, want lines set, no slot, not on source", b)
			}
		}
		want := []string{extra.Text, e.pos.Achievements[1].Text}
		if diff := cmp.Diff(want, texts); diff != "" {
			t.Errorf("bank texts (-want +got):\n%s", diff)
		}
	})

	t.Run("a reworded heading with no saved link auto-links to its one position and saves it", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		const reworded = "Engineer at Acme Ltd"
		docs := cvtailortest.Docs{TabJSON: tabJSON(t, head(reworded), bullet("Built and maintained the public APIs for the platform"), bullet("Ran on-call"), bullet("Wrote docs"))}
		id := e.readyWith(t, tick{docs: docs}, e.citedEdit("Cut p99 latency"))
		svc := cvtailor.NewService(e.store, docs, e.asker, e.drive)

		got, err := svc.DraftRoles(t.Context(), userID, dto.DraftQuery{ID: id})
		if err != nil {
			t.Fatalf("DraftRoles() error = %v", err)
		}

		if len(got.Roles) != 1 || got.Roles[0].PositionID == nil || *got.Roles[0].PositionID != e.pos.ID {
			t.Errorf("DraftRoles() roles = %+v, want one role for %s", got.Roles, e.pos.ID)
		}
		saved, err := e.store.ListHeadingMappings(t.Context(), userID, docID, tabID)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.ContainsFunc(saved, func(m dto.HeadingMapping) bool {
			return m.HeadingText == reworded && m.PositionID != nil && *m.PositionID == e.pos.ID
		}) {
			t.Errorf("saved mappings = %+v, want %q linked to %s", saved, reworded, e.pos.ID)
		}
	})

	t.Run("a heading whose link was nulled by a deleted position auto-links again", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		id := e.readyWith(t, tick{}, e.citedEdit("Cut p99 latency"))
		if err := e.store.SaveHeadingMappings(t.Context(), userID, docID, tabID, []dto.HeadingMapping{{HeadingText: heading}}); err != nil {
			t.Fatal(err)
		}

		got, err := e.svc.DraftRoles(t.Context(), userID, dto.DraftQuery{ID: id})
		if err != nil {
			t.Fatalf("DraftRoles() error = %v", err)
		}

		if len(got.Roles) != 1 || got.Roles[0].Budget != sourceBudget() {
			t.Errorf("DraftRoles() roles = %+v, want one linked role with budget %d", got.Roles, sourceBudget())
		}
	})

	t.Run("an achievement whose text is on the source CV is marked on source", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		addAchievements(t, e.pos.ID, e.store, "Ran on-call")
		id := e.readyDrafts(t, 1)[0]

		got, err := e.svc.DraftRoles(t.Context(), userID, dto.DraftQuery{ID: id})
		if err != nil {
			t.Fatalf("DraftRoles() error = %v", err)
		}

		for _, b := range got.Roles[0].Bank {
			if b.OnSource != (b.Text == "Ran on-call") {
				t.Errorf("bank bullet %q OnSource = %v", b.Text, b.OnSource)
			}
		}
	})

	t.Run("spills only on a blocking page_count finding", func(t *testing.T) {
		t.Parallel()
		for _, tt := range []struct {
			name     string
			findings []dto.DraftFinding
			want     bool
		}{
			{"blocking page count", []dto.DraftFinding{{Check: "page_count", Severity: "block"}}, true},
			{"info page count", []dto.DraftFinding{{Check: "page_count", Severity: "info"}}, false},
			{"other blocking check", []dto.DraftFinding{{Check: "banned_words", Severity: "block"}}, false},
		} {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				e := newDraftEnv(t)
				id := e.readyDrafts(t, 1)[0]
				d, err := e.store.GetDraft(t.Context(), userID, id)
				if err != nil {
					t.Fatal(err)
				}
				if err := e.store.SetDraftEdits(t.Context(), userID, id, d.EditSet, tt.findings, d.AchievementIDs); err != nil {
					t.Fatal(err)
				}

				got, err := e.svc.DraftRoles(t.Context(), userID, dto.DraftQuery{ID: id})
				if err != nil {
					t.Fatalf("DraftRoles() error = %v", err)
				}

				if got.Spills != tt.want {
					t.Errorf("Spills = %v, want %v", got.Spills, tt.want)
				}
			})
		}
	})

	t.Run("kept reflects a kept outcome", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		if _, err := e.store.SetDraftOutcome(t.Context(), userID, id, dto.OutcomeKept); err != nil {
			t.Fatal(err)
		}

		got, err := e.svc.DraftRoles(t.Context(), userID, dto.DraftQuery{ID: id})
		if err != nil {
			t.Fatalf("DraftRoles() error = %v", err)
		}

		if !got.Kept {
			t.Errorf("Kept = false, want true")
		}
	})

	t.Run("an unlinked heading keeps its source bullets and has no bank", func(t *testing.T) {
		t.Parallel()
		e := newDraftEnv(t)
		id := e.readyDrafts(t, 1)[0]
		docs := cvtailortest.Docs{TabJSON: baseTab(t, head("Side project"), bullet("Built a tool"))}
		svc := cvtailor.NewService(e.store, docs, e.asker, e.drive)

		got, err := svc.DraftRoles(t.Context(), userID, dto.DraftQuery{ID: id})
		if err != nil {
			t.Fatalf("DraftRoles() error = %v", err)
		}

		if len(got.Roles) != 2 {
			t.Fatalf("DraftRoles() = %+v, want the linked role then the unlinked one", got.Roles)
		}
		role := got.Roles[1]
		if role.PositionID != nil || role.Title != "Side project" || len(role.Bank) != 0 || len(role.Bullets) != 1 || role.Bullets[0].Text != "Built a tool" || role.Budget != role.Bullets[0].Lines {
			t.Errorf("unlinked role = %+v, want its source bullet, budget of its lines and no bank", role)
		}
	})

	t.Run("a draft without edits conflicts", func(t *testing.T) {
		t.Parallel()
		e, id := newQueuedDraft(t)

		_, err := e.svc.DraftRoles(t.Context(), userID, dto.DraftQuery{ID: id})

		if !apperr.IsKind(err, apperr.KindConflict) {
			t.Errorf("DraftRoles() error = %v, want kind %v", err, apperr.KindConflict)
		}
	})
}

func TestDraftRolesGolden(t *testing.T) {
	t.Parallel()
	e := newDraftEnv(t)
	id := e.readyDrafts(t, 1)[0]
	e.asker.answers = map[string]dto.Answer{
		questionPrefix + e.pos.Achievements[0].Text: {PYes: 0.6, PNo: 0.1},
		questionPrefix + e.pos.Achievements[1].Text: {PYes: 0.3, PNo: 0.1},
	}

	got, err := e.svc.DraftRoles(t.Context(), userID, dto.DraftQuery{ID: id})
	if err != nil {
		t.Fatalf("DraftRoles() error = %v", err)
	}
	got.DraftID = "draft-1"
	for i := range got.Roles {
		got.Roles[i].PositionID = new("position-1")
		for j := range got.Roles[i].Bank {
			got.Roles[i].Bank[j].AchievementID = new("achievement-1")
		}
		for j := range got.Roles[i].Bullets {
			got.Roles[i].Bullets[j].AchievementID = new("achievement-2")
		}
	}
	raw, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if *update {
		if err := os.WriteFile(draftRolesGolden, append(raw, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(draftRolesGolden)
	if err != nil {
		t.Fatalf("read golden: %v (run with -update)", err)
	}
	var wantJSON, gotJSON any
	if err := json.Unmarshal(want, &wantJSON); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &gotJSON); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(wantJSON, gotJSON); diff != "" {
		t.Errorf("DraftRoles JSON (-want +got):\n%s", diff)
	}
}

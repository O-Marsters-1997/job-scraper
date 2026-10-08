package cvtailor_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
)

const skillQuestionPrefix = "Would this job value a candidate with experience of "

type skillLean struct {
	name      string
	pYes, pNo float64
}

func skillAnswers(leans ...skillLean) map[string]dto.Answer {
	out := map[string]dto.Answer{}
	for _, l := range leans {
		out[skillQuestionPrefix+l.name+"?"] = dto.Answer{PYes: l.pYes, PNo: l.pNo, Confidence: 0.5}
	}
	return out
}

func TestSkillSuggestions(t *testing.T) {
	baseDocs := func(t *testing.T) cvtailortest.Docs {
		t.Helper()
		return cvtailortest.Docs{TabJSON: tabJSON(t,
			head("Skills"),
			prose("Languages: Go, PHP, Perl"),
			prose("Databases: Postgres, Oracle"),
		)}
	}
	addBank := func(t *testing.T, st *cvtailortest.FakeStore, category string, names ...string) map[string]string {
		t.Helper()
		ids := map[string]string{}
		for _, n := range names {
			b, err := st.CreateBankSkill(t.Context(), userID, dto.BankSkillInput{Name: n, Category: category})
			if err != nil {
				t.Fatal(err)
			}
			ids[n] = b.ID
		}
		return ids
	}
	q := dto.SkillSuggestionsQuery{JobID: "job-1", DocID: "doc", TabID: "tab"}

	t.Run("fit Bank Skill replaces the line's lowest-lean item", func(t *testing.T) {
		asker := &fakeAsker{answers: skillAnswers(
			skillLean{"Rust", 0.8, 0.1},
			skillLean{"Go", 0.6, 0.1}, skillLean{"PHP", 0.1, 0.5}, skillLean{"Perl", 0.2, 0.3},
		)}
		svc, st := newService(t, baseDocs(t), asker)
		ids := addBank(t, st, "languages", "Rust", "Go")

		got, err := svc.SkillSuggestions(t.Context(), userID, q)
		if err != nil {
			t.Fatal(err)
		}
		want := []dto.SkillCandidate{{BankSkillID: ids["Rust"], Name: "Rust", Score: 0.7, State: dto.SuggestionFit, Preselected: true, Replaces: "PHP"}}
		if diff := cmp.Diff(want, got.Lines[0].Candidates, cmp.Comparer(func(a, b float64) bool { return a-b < 1e-9 && b-a < 1e-9 })); diff != "" {
			t.Errorf("SkillSuggestions() languages candidates (-want +got):\n%s", diff)
		}
		if len(got.Unplaced) != 0 {
			t.Errorf("SkillSuggestions() unplaced = %v, want none", got.Unplaced)
		}
	})

	t.Run("a Bank Skill already on the base CV is never offered", func(t *testing.T) {
		asker := &fakeAsker{answers: skillAnswers(skillLean{"Rust", 0.8, 0.1})}
		svc, st := newService(t, baseDocs(t), asker)
		addBank(t, st, "Languages", "go", "Rust")

		got, err := svc.SkillSuggestions(t.Context(), userID, q)
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, c := range got.Lines[0].Candidates {
			names = append(names, c.Name)
		}
		if diff := cmp.Diff([]string{"Rust"}, names); diff != "" {
			t.Errorf("SkillSuggestions() candidates (-want +got):\n%s", diff)
		}
	})

	t.Run("at most three candidates are preselected, each replacing a distinct item", func(t *testing.T) {
		asker := &fakeAsker{answers: skillAnswers(
			skillLean{"A", 0.9, 0.0}, skillLean{"B", 0.8, 0.0}, skillLean{"C", 0.7, 0.0}, skillLean{"D", 0.6, 0.0},
			skillLean{"Go", 0.3, 0.3}, skillLean{"PHP", 0.2, 0.3}, skillLean{"Perl", 0.1, 0.3},
		)}
		svc, st := newService(t, baseDocs(t), asker)
		addBank(t, st, "Languages", "D", "C", "B", "A")

		got, err := svc.SkillSuggestions(t.Context(), userID, q)
		if err != nil {
			t.Fatal(err)
		}
		type row struct {
			Name     string
			Selected bool
			Replaces string
		}
		var rows []row
		for _, c := range got.Lines[0].Candidates {
			rows = append(rows, row{c.Name, c.Preselected, c.Replaces})
		}
		want := []row{{"A", true, "Perl"}, {"B", true, "PHP"}, {"C", true, "Go"}, {"D", false, ""}}
		if diff := cmp.Diff(want, rows); diff != "" {
			t.Errorf("SkillSuggestions() candidates (-want +got):\n%s", diff)
		}
	})

	t.Run("a category matching no line is unplaced and not preselected", func(t *testing.T) {
		asker := &fakeAsker{answers: skillAnswers(skillLean{"Kubernetes", 0.9, 0.0})}
		svc, st := newService(t, baseDocs(t), asker)
		ids := addBank(t, st, "Infra", "Kubernetes")

		got, err := svc.SkillSuggestions(t.Context(), userID, q)
		if err != nil {
			t.Fatal(err)
		}
		want := []dto.SkillCandidate{{BankSkillID: ids["Kubernetes"], Name: "Kubernetes", Score: 0.9, State: dto.SuggestionFit}}
		if diff := cmp.Diff(want, got.Unplaced); diff != "" {
			t.Errorf("SkillSuggestions() unplaced (-want +got):\n%s", diff)
		}
	})

	t.Run("an empty Bank asks nothing", func(t *testing.T) {
		asker := &fakeAsker{}
		svc, _ := newService(t, baseDocs(t), asker)

		got, err := svc.SkillSuggestions(t.Context(), userID, q)
		if err != nil {
			t.Fatal(err)
		}
		if len(asker.asked) != 0 || len(got.Unplaced) != 0 {
			t.Errorf("SkillSuggestions() asked %v, unplaced %v, want none", asker.asked, got.Unplaced)
		}
	})
}

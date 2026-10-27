package cvtailor_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
)

func TestPreviewImportSplitsTitleEmployerAndDropsLocation(t *testing.T) {
	docs := cvtailortest.Docs{TabJSON: tabJSON(t,
		head("Software Engineer, Eucalyptus – London  Jan 2026 – Jul 2026"), bullet("a"),
		head("Software Engineer | Eucalyptus | London  Jan 2026 – Jul 2026"), bullet("b"),
		head("Engineer at Acme, London"), bullet("c"),
	)}
	svc, _ := newService(t, docs, nil)

	got, err := svc.PreviewImport(t.Context(), userID, dto.ImportPreviewInput{DocID: "d", TabID: "t"})
	if err != nil {
		t.Fatal(err)
	}

	want := []dto.ImportPosition{
		{Employer: "Eucalyptus", Title: "Software Engineer", StartDate: ptr("2026-01-01"), EndDate: ptr("2026-07-01"), Achievements: []string{"a"}},
		{Employer: "Eucalyptus", Title: "Software Engineer", StartDate: ptr("2026-01-01"), EndDate: ptr("2026-07-01"), Achievements: []string{"b"}},
		{Employer: "Acme", Title: "Engineer", Achievements: []string{"c"}},
	}
	if diff := cmp.Diff(want, got.Positions); diff != "" {
		t.Fatalf("PreviewImport() positions (-want +got):\n%s", diff)
	}
}

func TestImportPositionsRepairsMisparsedPositions(t *testing.T) {
	imp := dto.ImportPosition{
		Employer: "Eucalyptus", Title: "Software Engineer", StartDate: ptr("2026-01-01"), EndDate: ptr("2026-07-01"),
		Achievements: []string{"old", "new"},
	}
	bad := dto.PositionInput{Employer: "London", Title: "Software Engineer, Eucalyptus", StartDate: ptr("2026-01-01"), EndDate: ptr("2026-07-01")}

	t.Run("single match is updated in place", func(t *testing.T) {
		svc, st := newService(t, nil, nil)
		p, err := st.CreatePosition(t.Context(), userID, bad)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := st.CreateAchievement(t.Context(), userID, dto.AchievementInput{PositionID: p.ID, Text: "old"}); err != nil {
			t.Fatal(err)
		}

		if _, err := svc.ImportPositions(t.Context(), userID, dto.ImportInput{Positions: []dto.ImportPosition{imp}}); err != nil {
			t.Fatal(err)
		}

		got, err := st.ListPositions(t.Context(), userID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].ID != p.ID || got[0].Employer != "Eucalyptus" || got[0].Title != "Software Engineer" || len(got[0].Achievements) != 2 {
			t.Fatalf("Bank after repair = %+v, want position %s renamed with two achievements", got, p.ID)
		}
	})

	t.Run("ambiguous or absent match creates a fresh position", func(t *testing.T) {
		svc, st := newService(t, nil, nil)
		if _, err := st.CreatePosition(t.Context(), userID, bad); err != nil {
			t.Fatal(err)
		}
		twin := bad
		twin.Employer = "Remote"
		if _, err := st.CreatePosition(t.Context(), userID, twin); err != nil {
			t.Fatal(err)
		}
		other := imp
		other.Employer = "Elsewhere"

		for _, in := range []dto.ImportPosition{imp, other} {
			if _, err := svc.ImportPositions(t.Context(), userID, dto.ImportInput{Positions: []dto.ImportPosition{in}}); err != nil {
				t.Fatal(err)
			}
		}

		got, err := st.ListPositions(t.Context(), userID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 4 {
			t.Fatalf("Bank = %d positions, want 4 (2 untouched + 2 fresh)", len(got))
		}
	})
}

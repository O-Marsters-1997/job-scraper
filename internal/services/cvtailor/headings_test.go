package cvtailor_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
)

type roleBlock struct {
	heading string
	bullets int
}

func cvTab(t *testing.T, roles ...roleBlock) json.RawMessage {
	t.Helper()
	lines := []cvLine{head("Experience")}
	for _, r := range roles {
		lines = append(lines, head(r.heading))
		for i := range r.bullets {
			lines = append(lines, bullet(fmt.Sprintf("bullet %d of %s", i, r.heading)))
		}
	}
	return tabJSON(t, lines...)
}

func newService(t *testing.T, docs cvtailor.DocFetcher, asker cvtailor.Asker) (*cvtailor.Service, *cvtailortest.FakeStore) {
	t.Helper()
	st := cvtailortest.NewFakeStore()
	return cvtailor.NewService(st, docs, asker, nil), st
}

func addPosition(t *testing.T, st *cvtailortest.FakeStore, employer, title string) dto.Position {
	t.Helper()
	p, err := st.CreatePosition(t.Context(), userID, dto.PositionInput{Employer: employer, Title: title})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestHeadingsAutoMatch(t *testing.T) {
	ctx := t.Context()
	docs := cvtailortest.Docs{TabJSON: cvTab(t,
		roleBlock{"Senior Engineer, ACME Corp.", 2},
		roleBlock{"Staff Engineer - Globex", 1},
		roleBlock{"Initech (2015-2018)", 1},
		roleBlock{"Lead Developer | Hooli", 1},
		roleBlock{"Lead Designer | Hooli", 1},
	)}
	svc, st := newService(t, docs, nil)
	acme := addPosition(t, st, "Acme Corp", "Senior Engineer")
	hooliDev := addPosition(t, st, "Hooli", "Lead Developer")
	hooliDesign := addPosition(t, st, "Hooli", "Lead Designer")
	addPosition(t, st, "Initech", "Engineer")
	addPosition(t, st, "Initech", "Engineer")

	got, err := svc.Headings(ctx, userID, dto.CVTabQuery{DocID: "doc", TabID: "tab"})
	if err != nil {
		t.Fatal(err)
	}
	want := []dto.CVHeading{
		{Text: "Senior Engineer, ACME Corp.", PositionID: &acme.ID, SlotCount: 2},
		{Text: "Staff Engineer - Globex", SlotCount: 1},
		{Text: "Initech (2015-2018)", SlotCount: 1},
		{Text: "Lead Developer | Hooli", PositionID: &hooliDev.ID, SlotCount: 1},
		{Text: "Lead Designer | Hooli", PositionID: &hooliDesign.ID, SlotCount: 1},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Headings() (-want +got):\n%s", diff)
	}
}

func TestHeadingsSkipStepOnceSaved(t *testing.T) {
	ctx := t.Context()
	svc, st := newService(t, cvtailortest.Docs{TabJSON: cvTab(t, roleBlock{"Engineer, Acme", 1}, roleBlock{"Volunteer, Nowhere", 1})}, nil)
	acme := addPosition(t, st, "Acme", "Engineer")

	first, err := svc.Headings(ctx, userID, dto.CVTabQuery{DocID: "doc", TabID: "tab"})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range first {
		if h.Confirmed {
			t.Fatalf("first Headings() confirmed %q before any save", h.Text)
		}
	}

	_, err = svc.SaveHeadings(ctx, userID, dto.HeadingMappingsInput{DocID: "doc", TabID: "tab", Mappings: []dto.HeadingMapping{
		{HeadingText: "Engineer, Acme", PositionID: &acme.ID},
		{HeadingText: "Volunteer, Nowhere"},
	}})
	if err != nil {
		t.Fatal(err)
	}

	second, err := svc.Headings(ctx, userID, dto.CVTabQuery{DocID: "doc", TabID: "tab"})
	if err != nil {
		t.Fatal(err)
	}
	want := []dto.CVHeading{
		{Text: "Engineer, Acme", PositionID: &acme.ID, Confirmed: true, SlotCount: 1},
		{Text: "Volunteer, Nowhere", Confirmed: true, SlotCount: 1},
	}
	if diff := cmp.Diff(want, second); diff != "" {
		t.Errorf("Headings() after save (-want +got):\n%s", diff)
	}
}

func TestSaveHeadingsRejectsAnotherUsersPosition(t *testing.T) {
	svc, st := newService(t, cvtailortest.Docs{}, nil)
	other, err := st.CreatePosition(t.Context(), otherUserID, dto.PositionInput{Employer: "Acme", Title: "Engineer"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.SaveHeadings(t.Context(), userID, dto.HeadingMappingsInput{
		Mappings: []dto.HeadingMapping{{HeadingText: "Engineer, Acme", PositionID: &other.ID}},
	})
	if err == nil {
		t.Fatal("SaveHeadings() with another user's position = nil error, want invalid")
	}
}

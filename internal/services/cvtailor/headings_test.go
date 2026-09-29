package cvtailor_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
)

const userID = "user-1"

type fakeDocs struct{ raw json.RawMessage }

func (f fakeDocs) GetDocument(context.Context, string, string, string) (json.RawMessage, error) {
	return f.raw, nil
}

type roleBlock struct {
	heading string
	bullets int
}

func cvTab(t *testing.T, roles ...roleBlock) json.RawMessage {
	t.Helper()
	var content []string
	idx := 1
	para := func(text, style string, bullet bool) {
		end := idx + len(text) + 1
		b := ""
		if bullet {
			b = `,"bullet":{"listId":"l"}`
		}
		content = append(content, fmt.Sprintf(
			`{"startIndex":%d,"endIndex":%d,"paragraph":{"elements":[{"textRun":{"content":%q}}],"paragraphStyle":{"namedStyleType":%q}%s}}`,
			idx, end, text+"\n", style, b))
		idx = end
	}
	para("Experience", "HEADING_1", false)
	for _, r := range roles {
		para(r.heading, "HEADING_2", false)
		for i := range r.bullets {
			para(fmt.Sprintf("bullet %d of %s", i, r.heading), "NORMAL_TEXT", true)
		}
	}
	return json.RawMessage(`{"documentTab":{"body":{"content":[` + strings.Join(content, ",") + `]}}}`)
}

func newService(t *testing.T, docs cvtailor.DocFetcher, asker cvtailor.Asker) (*cvtailor.Service, *cvtailortest.FakeStore) {
	t.Helper()
	st := cvtailortest.NewFakeStore()
	return cvtailor.NewService(st, docs, asker), st
}

func addPosition(t *testing.T, st *cvtailortest.FakeStore, employer, title string) dto.Position {
	t.Helper()
	p, err := st.CreatePosition(context.Background(), userID, dto.PositionInput{Employer: employer, Title: title})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestHeadingsAutoMatch(t *testing.T) {
	ctx := context.Background()
	docs := fakeDocs{cvTab(t,
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
	ctx := context.Background()
	svc, st := newService(t, fakeDocs{cvTab(t, roleBlock{"Engineer, Acme", 1}, roleBlock{"Volunteer, Nowhere", 1})}, nil)
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
	svc, st := newService(t, fakeDocs{}, nil)
	other, err := st.CreatePosition(context.Background(), "user-2", dto.PositionInput{Employer: "Acme", Title: "Engineer"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.SaveHeadings(context.Background(), userID, dto.HeadingMappingsInput{
		Mappings: []dto.HeadingMapping{{HeadingText: "Engineer, Acme", PositionID: &other.ID}},
	})
	if err == nil {
		t.Fatal("SaveHeadings() with another user's position = nil error, want invalid")
	}
}

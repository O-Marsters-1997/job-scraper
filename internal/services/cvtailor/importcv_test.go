package cvtailor_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
)

type cvLine struct {
	heading bool
	text    string
}

func head(text string) cvLine   { return cvLine{heading: true, text: text} }
func bullet(text string) cvLine { return cvLine{text: text} }

func tabJSON(t *testing.T, lines ...cvLine) json.RawMessage {
	t.Helper()
	var content []string
	idx := 1
	for _, l := range lines {
		style, bulletJSON := "NORMAL_TEXT", ""
		if l.heading {
			style = "HEADING_2"
		} else {
			bulletJSON = `,"bullet":{}`
		}
		end := idx + len(l.text) + 1
		content = append(content, fmt.Sprintf(
			`{"startIndex":%d,"endIndex":%d,"paragraph":{"elements":[{"textRun":{"content":%q}}],"paragraphStyle":{"namedStyleType":%q}%s}}`,
			idx, end, l.text+"\n", style, bulletJSON))
		idx = end
	}
	return json.RawMessage(`{"documentTab":{"body":{"content":[` + strings.Join(content, ",") + `]}}}`)
}

func TestPreviewImportParsesHeadingsAndFlagsExistingEmployers(t *testing.T) {
	store := cvtailortest.NewFakeStore()
	if _, err := store.CreatePosition(context.Background(), "u1", dto.PositionInput{Employer: "acme ltd", Title: "Dev"}); err != nil {
		t.Fatal(err)
	}
	docs := cvtailortest.Docs{TabJSON: tabJSON(t,
		head("Senior Engineer, Acme Ltd (Jan 2021 - Present)"),
		bullet("Cut latency."),
		bullet("Mentored four."),
		head("Analyst at Widgets Inc | sometime back then"),
		bullet("Built reports."),
		head("Education"),
	)}
	svc := cvtailor.NewService(store, docs, nil)

	got, err := svc.PreviewImport(context.Background(), "u1", dto.ImportPreviewInput{DocID: "d", TabID: "t"})
	if err != nil {
		t.Fatal(err)
	}

	want := dto.ImportPreview{Positions: []dto.ImportPosition{
		{Employer: "Acme Ltd", Title: "Senior Engineer", StartDate: ptr("2021-01-01"), Achievements: []string{"Cut latency.", "Mentored four."}, EmployerExists: true},
		{Employer: "Widgets Inc | sometime back then", Title: "Analyst", Achievements: []string{"Built reports."}},
	}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("PreviewImport() (-want +got):\n%s", diff)
	}
}

func TestPreviewImportReadsDateRangesAndLeavesUnparseableEmpty(t *testing.T) {
	docs := cvtailortest.Docs{TabJSON: tabJSON(t,
		head("Engineer, Acme (2018 - 2021)"), bullet("a"),
		head("Lead, Globex (last summer - now-ish)"), bullet("b"),
	)}
	svc := cvtailor.NewService(cvtailortest.NewFakeStore(), docs, nil)

	got, err := svc.PreviewImport(context.Background(), "u1", dto.ImportPreviewInput{DocID: "d", TabID: "t"})
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Positions) != 2 {
		t.Fatalf("PreviewImport() positions = %d, want 2", len(got.Positions))
	}
	first, second := got.Positions[0], got.Positions[1]
	if first.StartDate == nil || *first.StartDate != "2018-01-01" || first.EndDate == nil || *first.EndDate != "2021-01-01" {
		t.Fatalf("first dates = %v..%v, want 2018-01-01..2021-01-01", first.StartDate, first.EndDate)
	}
	if second.StartDate != nil || second.EndDate != nil {
		t.Fatalf("second dates = %v..%v, want none", second.StartDate, second.EndDate)
	}
}

func TestPreviewImportPropagatesDocErrors(t *testing.T) {
	notFound := apperr.NotFound("no such doc")
	svc := cvtailor.NewService(cvtailortest.NewFakeStore(), cvtailortest.Docs{Err: notFound}, nil)

	_, err := svc.PreviewImport(context.Background(), "u1", dto.ImportPreviewInput{DocID: "d", TabID: "t"})

	if !errors.Is(err, notFound) {
		t.Fatalf("PreviewImport() err = %v, want %v", err, notFound)
	}
}

func TestImportPositionsAppendsAndRejectsInvalidPositions(t *testing.T) {
	store := cvtailortest.NewFakeStore()
	svc := cvtailor.NewService(store, nil, nil)
	ctx := context.Background()

	in := dto.ImportInput{Positions: []dto.ImportPosition{
		{Employer: " Acme ", Title: "Engineer", Achievements: []string{"one", "  ", "two"}},
	}}
	for range 2 {
		if _, err := svc.ImportPositions(ctx, "u1", in); err != nil {
			t.Fatal(err)
		}
	}
	got, err := store.ListPositions(ctx, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || len(got[0].Achievements) != 2 || got[0].Employer != "Acme" {
		t.Fatalf("Bank after two imports = %+v, want two Acme positions with two achievements", got)
	}

	bad := dto.ImportInput{Positions: []dto.ImportPosition{{Employer: "Acme", Title: ""}}}
	if _, err := svc.ImportPositions(ctx, "u1", bad); err == nil {
		t.Fatal("ImportPositions() with blank title: err = nil, want invalid")
	}
}

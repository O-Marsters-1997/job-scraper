package cvtailortest

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// RunHeadingMappingContract proves the store's heading mappings are scoped
// per user and Tab, upsert on re-save, and keep "none" as a saved choice.
func RunHeadingMappingContract(t *testing.T, newStore func(t *testing.T) Fixture) {
	t.Helper()
	ctx := context.Background()
	sortByText := cmpopts.SortSlices(func(a, b dto.HeadingMapping) bool { return a.HeadingText < b.HeadingText })

	t.Run("saves, upserts and scopes by user and tab", func(t *testing.T) {
		f := newStore(t)
		p, err := f.Store.CreatePosition(ctx, f.UserID, dto.PositionInput{Employer: "Acme", Title: "Engineer"})
		if err != nil {
			t.Fatal(err)
		}
		first := []dto.HeadingMapping{{HeadingText: "Acme", PositionID: &p.ID}, {HeadingText: "Other"}}
		if err := f.Store.SaveHeadingMappings(ctx, f.UserID, "doc", "tab", first); err != nil {
			t.Fatal(err)
		}
		if err := f.Store.SaveHeadingMappings(ctx, f.UserID, "doc", "tab", []dto.HeadingMapping{{HeadingText: "Acme"}}); err != nil {
			t.Fatal(err)
		}

		got, err := f.Store.ListHeadingMappings(ctx, f.UserID, "doc", "tab")
		if err != nil {
			t.Fatal(err)
		}
		want := []dto.HeadingMapping{{HeadingText: "Acme"}, {HeadingText: "Other"}}
		if diff := cmp.Diff(want, got, sortByText); diff != "" {
			t.Errorf("ListHeadingMappings() (-want +got):\n%s", diff)
		}

		for name, args := range map[string][3]string{
			"other user": {f.Other, "doc", "tab"},
			"other tab":  {f.UserID, "doc", "tab2"},
			"other doc":  {f.UserID, "doc2", "tab"},
		} {
			got, err := f.Store.ListHeadingMappings(ctx, args[0], args[1], args[2])
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if len(got) != 0 {
				t.Errorf("ListHeadingMappings(%s) = %v, want none", name, got)
			}
		}
	})
}

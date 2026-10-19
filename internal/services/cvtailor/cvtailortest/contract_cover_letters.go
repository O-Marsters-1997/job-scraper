package cvtailortest

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func runCoverLetterContract(t *testing.T, newStore func(t *testing.T) Fixture) {
	t.Helper()
	ctx := t.Context()

	t.Run("only a tracked letter tab of the user is a letter tab", func(t *testing.T) {
		f := newStore(t)
		f.TrackLetterTab(t, f.UserID, "doc", "t.0")
		for _, tc := range []struct {
			name                 string
			userID, docID, tabID string
			want                 bool
		}{
			{"tracked tab", f.UserID, "doc", "t.0", true},
			{"other tab", f.UserID, "doc", "t.1", false},
			{"other doc", f.UserID, "other", "t.0", false},
			{"other user", f.Other, "doc", "t.0", false},
		} {
			got, err := f.Store.IsLetterTab(ctx, tc.userID, tc.docID, tc.tabID)
			if err != nil || got != tc.want {
				t.Errorf("IsLetterTab(%s) = %v, %v, want %v", tc.name, got, err, tc.want)
			}
		}
	})

	t.Run("created letters list newest first and per user", func(t *testing.T) {
		f := newStore(t)
		in := dto.NewCoverLetter{JobID: f.JobID, SourceDocID: "doc", SourceTabID: "t.0", ScaffoldTabID: "t.1", AchievementIDs: []string{}, Model: "m", Cost: 0.5}
		first, err := f.Store.CreateCoverLetter(ctx, f.UserID, in)
		if err != nil {
			t.Fatal(err)
		}
		in.ScaffoldTabID = "t.2"
		second, err := f.Store.CreateCoverLetter(ctx, f.UserID, in)
		if err != nil {
			t.Fatal(err)
		}
		if first.ID == second.ID || first.SourceDocID != "doc" || first.TabURL == second.TabURL || first.Cost != 0.5 {
			t.Errorf("CreateCoverLetter() = %+v, %+v, want distinct letters echoing the input", first, second)
		}
		got, err := f.Store.ListJobCoverLetters(ctx, f.UserID, f.JobID)
		if err != nil || len(got) != 2 || got[0].ID != second.ID || got[1].ID != first.ID {
			t.Fatalf("ListJobCoverLetters() = %+v, %v, want [second first]", got, err)
		}
		other, err := f.Store.ListJobCoverLetters(ctx, f.Other, f.JobID)
		if err != nil || len(other) != 0 {
			t.Errorf("other user's ListJobCoverLetters() = %+v, %v, want none", other, err)
		}
	})

	t.Run("a missing job is not found", func(t *testing.T) {
		f := newStore(t)
		_, err := f.Store.CreateCoverLetter(ctx, f.UserID, dto.NewCoverLetter{JobID: missingID, AchievementIDs: []string{}})
		if !apperr.IsKind(err, apperr.KindNotFound) {
			t.Errorf("CreateCoverLetter(missing job) = %v, want not found", err)
		}
		if _, err := f.Store.JobBrief(ctx, missingID); !apperr.IsKind(err, apperr.KindNotFound) {
			t.Errorf("JobBrief(missing job) = %v, want not found", err)
		}
	})
}

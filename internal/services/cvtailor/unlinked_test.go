package cvtailor_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvtailortest"
)

func TestGetDraftUnlinkedHeadings(t *testing.T) {
	t.Run("carries a heading with no mapped Position, flagged unlinked", func(t *testing.T) {
		t.Parallel()
		const other = "Analyst, Globex"
		e := newDraftEnv(t)
		docs := cvtailortest.Docs{TabJSON: baseTab(t, head(other), bullet("Reported weekly numbers"))}
		id := e.queue(t)
		e.run(t, tick{docs: docs, editor: cvtailortest.Editing(e.bulletResult("Cut p99 latency by quickly moving queries to Postgres", 0))})

		got := e.draft(t, id).Content.Positions

		if len(got) != 2 || !got[0].Linked || got[0].PositionID != e.pos.ID {
			t.Fatalf("GetDraft().Content.Positions = %+v, want the linked position first", got)
		}
		want := dto.DraftPosition{HeadingText: other, Bullets: []dto.DraftBullet{{Text: "Reported weekly numbers", AchievementIDs: []string{}}}}
		if diff := cmp.Diff(want, got[1]); diff != "" {
			t.Errorf("unlinked position mismatch (-want +got):\n%s", diff)
		}
	})
}

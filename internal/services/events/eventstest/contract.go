package eventstest

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/services/events"
)

type Fixture struct {
	Store       events.Store
	UserID      string
	OtherUserID string
	SubjectID   string
}

// RunStoreContract runs the behaviour every events.Store must offer, against
// the fake and the real store alike.
func RunStoreContract(t *testing.T, newFixture func(t *testing.T) Fixture) {
	t.Helper()
	t.Run("lists inserted events oldest first with their subject and props", func(t *testing.T) {
		f := newFixture(t)
		insert(t, f, f.UserID, events.JobOpened, f.SubjectID, `{"score":70}`)
		insert(t, f, f.UserID, events.AlertOpened, "", `{}`)

		got, err := f.Store.ListEvents(t.Context(), f.UserID, "")
		if err != nil {
			t.Fatalf("ListEvents() err = %v", err)
		}
		var gotShape [][2]string
		for _, e := range got {
			gotShape = append(gotShape, [2]string{e.Type, e.SubjectID})
		}
		wantShape := [][2]string{{events.JobOpened, f.SubjectID}, {events.AlertOpened, ""}}
		if diff := cmp.Diff(wantShape, gotShape); diff != "" {
			t.Errorf("ListEvents() types and subjects (-want +got):\n%s", diff)
		}
		var props map[string]any
		if err := json.Unmarshal(got[0].Props, &props); err != nil || props["score"] != float64(70) {
			t.Errorf("ListEvents()[0].Props = %s, %v, want score 70", got[0].Props, err)
		}
	})

	t.Run("filters by type", func(t *testing.T) {
		f := newFixture(t)
		insert(t, f, f.UserID, events.JobOpened, f.SubjectID, `{}`)
		insert(t, f, f.UserID, events.JobDismissed, f.SubjectID, `{}`)

		got, err := f.Store.ListEvents(t.Context(), f.UserID, events.JobDismissed)
		if err != nil || len(got) != 1 || got[0].Type != events.JobDismissed {
			t.Errorf("ListEvents(job_dismissed) = %+v, %v, want one job_dismissed", got, err)
		}
	})

	t.Run("never lists another user's events", func(t *testing.T) {
		f := newFixture(t)
		insert(t, f, f.OtherUserID, events.JobOpened, f.SubjectID, `{}`)

		got, err := f.Store.ListEvents(t.Context(), f.UserID, "")
		if err != nil || len(got) != 0 {
			t.Errorf("ListEvents() = %+v, %v, want none", got, err)
		}
	})
}

func insert(t *testing.T, f Fixture, userID, eventType, subjectID, props string) {
	t.Helper()
	if err := f.Store.InsertEvent(t.Context(), nil, userID, eventType, subjectID, []byte(props)); err != nil {
		t.Fatalf("InsertEvent(%s) err = %v", eventType, err)
	}
}

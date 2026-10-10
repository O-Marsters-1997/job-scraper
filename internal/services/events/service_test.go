package events_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/events"
	"github.com/ollymarsters/job-scraper/internal/services/events/eventstest"
)

const (
	userID = "user-1"
	jobID  = "00000000-0000-0000-0000-000000000001"
)

var scored = dto.JobScoreEvidence{
	Score: 72, Band: "good", Model: "jev-1", Fingerprint: "fp-1",
	Breakdown: []dto.ScoreRow{{Key: "tech:go", Label: "Go"}},
}

func props(t *testing.T, e dto.Event) map[string]any {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(e.Props, &got); err != nil {
		t.Fatalf("props %s: %v", e.Props, err)
	}
	return got
}

func only(t *testing.T, st *eventstest.FakeStore) dto.Event {
	t.Helper()
	got, err := st.ListEvents(t.Context(), userID, "")
	if err != nil || len(got) != 1 {
		t.Fatalf("ListEvents() = %+v, %v, want one event", got, err)
	}
	return got[0]
}

func TestRecord(t *testing.T) {
	t.Run("stamps a job event with the job's current score", func(t *testing.T) {
		st := eventstest.NewFakeStore()
		svc := events.NewService(st, eventstest.ScoresAs(scored))

		if _, err := svc.Record(t.Context(), userID, dto.EventInput{Type: events.JobOpened, SubjectID: jobID}); err != nil {
			t.Fatalf("Record() err = %v", err)
		}

		e := only(t, st)
		if e.Type != events.JobOpened || e.SubjectID != jobID {
			t.Errorf("event = %+v, want job_opened for %s", e, jobID)
		}
		got := props(t, e)
		breakdown, _ := got["breakdown"].([]any)
		delete(got, "breakdown")
		want := map[string]any{"score": float64(72), "band": "good", "score_model": "jev-1", "score_fingerprint": "fp-1"}
		if diff := cmp.Diff(want, got); diff != "" || len(breakdown) != 1 {
			t.Errorf("props = %v, breakdown rows %d (-want +got):\n%s", got, len(breakdown), diff)
		}
	})

	t.Run("omits the snapshot when the job is unscored", func(t *testing.T) {
		st := eventstest.NewFakeStore()
		svc := events.NewService(st, eventstest.Unscored())

		if _, err := svc.Record(t.Context(), userID, dto.EventInput{Type: events.JobOpened, SubjectID: jobID}); err != nil {
			t.Fatalf("Record() err = %v", err)
		}

		if got := props(t, only(t, st)); len(got) != 0 {
			t.Errorf("props = %v, want empty", got)
		}
	})

	t.Run("keeps a dismiss reason beside the snapshot", func(t *testing.T) {
		st := eventstest.NewFakeStore()
		svc := events.NewService(st, eventstest.ScoresAs(scored))

		in := dto.EventInput{Type: events.JobDismissed, SubjectID: jobID, Reason: "wrong seniority"}
		if _, err := svc.Record(t.Context(), userID, in); err != nil {
			t.Fatalf("Record() err = %v", err)
		}

		got := props(t, only(t, st))
		if got["reason"] != "wrong seniority" || got["score"] != float64(72) {
			t.Errorf("props = %v, want reason and score", got)
		}
	})

	t.Run("records an alert open without a subject", func(t *testing.T) {
		st := eventstest.NewFakeStore()
		svc := events.NewService(st, eventstest.ScoresAs(scored))

		if _, err := svc.Record(t.Context(), userID, dto.EventInput{Type: events.AlertOpened}); err != nil {
			t.Fatalf("Record() err = %v", err)
		}

		if e := only(t, st); e.Type != events.AlertOpened || e.SubjectID != "" {
			t.Errorf("event = %+v, want subjectless alert_opened", e)
		}
	})

	for name, in := range map[string]dto.EventInput{
		"unknown type":           {Type: "job_deleted", SubjectID: jobID},
		"server-only type":       {Type: events.ApplicationCreated, SubjectID: jobID},
		"job event with no job":  {Type: events.JobOpened},
		"malformed subject":      {Type: events.JobOpened, SubjectID: "abc"},
		"oversized dismiss note": {Type: events.JobDismissed, SubjectID: jobID, Reason: strings.Repeat("x", 501)},
	} {
		t.Run("rejects "+name, func(t *testing.T) {
			st := eventstest.NewFakeStore()
			svc := events.NewService(st, eventstest.ScoresAs(scored))

			_, err := svc.Record(t.Context(), userID, in)

			if ae, ok := errors.AsType[*apperr.Error](err); !ok || ae.Kind() != apperr.KindInvalid {
				t.Errorf("Record(%+v) err = %v, want invalid", in, err)
			}
			if got, _ := st.ListEvents(t.Context(), userID, ""); len(got) != 0 {
				t.Errorf("stored %+v, want nothing", got)
			}
		})
	}

	t.Run("returns the snapshot failure and stores nothing", func(t *testing.T) {
		st := eventstest.NewFakeStore()
		wantErr := errors.New("boom")
		svc := events.NewService(st, eventstest.Fails(wantErr))

		_, err := svc.Record(t.Context(), userID, dto.EventInput{Type: events.JobOpened, SubjectID: jobID})

		if !errors.Is(err, wantErr) {
			t.Errorf("Record() err = %v, want %v", err, wantErr)
		}
		if got, _ := st.ListEvents(t.Context(), userID, ""); len(got) != 0 {
			t.Errorf("stored %+v, want nothing", got)
		}
	})
}

func TestApplicationEventRecording(t *testing.T) {
	t.Run("created carries the job and its score", func(t *testing.T) {
		st := eventstest.NewFakeStore()
		m := events.Build(events.Deps{Store: st, Snapshots: eventstest.ScoresAs(scored)})

		if err := m.RecordApplicationCreated(t.Context(), nil, userID, "app-1", jobID); err != nil {
			t.Fatalf("RecordApplicationCreated() err = %v", err)
		}

		e := only(t, st)
		got := props(t, e)
		if e.Type != events.ApplicationCreated || e.SubjectID != "app-1" || got["job_id"] != jobID || got["score"] != float64(72) {
			t.Errorf("event = %+v props %v, want application_created for app-1 on %s scored 72", e, got, jobID)
		}
	})

	t.Run("status change carries both statuses", func(t *testing.T) {
		st := eventstest.NewFakeStore()
		m := events.Build(events.Deps{Store: st, Snapshots: eventstest.Unscored()})

		if err := m.RecordApplicationStatusChanged(t.Context(), nil, userID, "app-1", jobID, "s-1", "s-2"); err != nil {
			t.Fatalf("RecordApplicationStatusChanged() err = %v", err)
		}

		want := map[string]any{"job_id": jobID, "from_status_id": "s-1", "to_status_id": "s-2"}
		if diff := cmp.Diff(want, props(t, only(t, st))); diff != "" {
			t.Errorf("props (-want +got):\n%s", diff)
		}
	})
}

package events_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/client"
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
	t.Run("stamps an open with the job's current score, without the breakdown", func(t *testing.T) {
		st := eventstest.NewFakeStore()
		svc := events.NewService(st, eventstest.ScoresAs(scored))

		if _, err := svc.Record(t.Context(), userID, dto.EventInput{Type: events.JobOpened, SubjectID: jobID}); err != nil {
			t.Fatalf("Record() err = %v", err)
		}

		e := only(t, st)
		if e.Type != events.JobOpened || e.SubjectID != jobID {
			t.Errorf("event = %+v, want job_opened for %s", e, jobID)
		}
		want := map[string]any{"score": float64(72), "band": "good", "score_model": "jev-1", "score_fingerprint": "fp-1"}
		if diff := cmp.Diff(want, props(t, e)); diff != "" {
			t.Errorf("props (-want +got):\n%s", diff)
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
		if got["reason"] != "wrong seniority" || got["score"] != float64(72) || got["breakdown"] == nil {
			t.Errorf("props = %v, want reason, score and breakdown", got)
		}
	})

	t.Run("marks an undone dismissal", func(t *testing.T) {
		st := eventstest.NewFakeStore()
		svc := events.NewService(st, eventstest.ScoresAs(scored))

		in := dto.EventInput{Type: events.JobDismissed, SubjectID: jobID, Undone: true}
		if _, err := svc.Record(t.Context(), userID, in); err != nil {
			t.Fatalf("Record() err = %v", err)
		}

		got := props(t, only(t, st))
		if got["undone"] != true || got["score"] != nil || got["breakdown"] != nil {
			t.Errorf("props = %v, want undone with no score stamp", got)
		}
	})

	t.Run("ignores undone on other event types", func(t *testing.T) {
		st := eventstest.NewFakeStore()
		svc := events.NewService(st, eventstest.ScoresAs(scored))

		in := dto.EventInput{Type: events.JobOpened, SubjectID: jobID, Undone: true}
		if _, err := svc.Record(t.Context(), userID, in); err != nil {
			t.Fatalf("Record() err = %v", err)
		}

		if got := props(t, only(t, st)); got["undone"] != nil {
			t.Errorf("props = %v, want no undone", got)
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

		snap := m.Snapshot(t.Context(), userID, jobID)
		if err := m.RecordApplicationCreated(t.Context(), nil, userID, "app-1", jobID, snap); err != nil {
			t.Fatalf("RecordApplicationCreated() err = %v", err)
		}

		e := only(t, st)
		got := props(t, e)
		if e.Type != events.ApplicationCreated || e.SubjectID != "app-1" || got["job_id"] != jobID || got["score"] != float64(72) || got["breakdown"] == nil {
			t.Errorf("event = %+v props %v, want application_created for app-1 on %s scored 72 with breakdown", e, got, jobID)
		}
	})

	t.Run("status change carries both statuses", func(t *testing.T) {
		st := eventstest.NewFakeStore()
		m := events.Build(events.Deps{Store: st, Snapshots: eventstest.Unscored()})

		if err := m.RecordApplicationStatusChanged(t.Context(), nil, userID, "app-1", jobID, "s-1", "s-2", nil); err != nil {
			t.Fatalf("RecordApplicationStatusChanged() err = %v", err)
		}

		want := map[string]any{"job_id": jobID, "from_status_id": "s-1", "to_status_id": "s-2"}
		if diff := cmp.Diff(want, props(t, only(t, st))); diff != "" {
			t.Errorf("props (-want +got):\n%s", diff)
		}
	})

	t.Run("a failed snapshot reads as no score", func(t *testing.T) {
		m := events.Build(events.Deps{Store: eventstest.NewFakeStore(), Snapshots: eventstest.Fails(errors.New("boom"))})

		if snap := m.Snapshot(t.Context(), userID, jobID); snap != nil {
			t.Errorf("Snapshot() = %+v, want nil", snap)
		}
	})
}

func TestClientStamp(t *testing.T) {
	t.Parallel()
	t.Run("a client-reported event takes the request's client", func(t *testing.T) {
		t.Parallel()
		st := eventstest.NewFakeStore()
		svc := events.NewService(st, eventstest.Unscored())
		ctx := iosContext(t)

		if _, err := svc.Record(ctx, userID, dto.EventInput{Type: events.AlertOpened}); err != nil {
			t.Fatalf("Record() err = %v", err)
		}
		if got := only(t, st).Client; got != "ios" {
			t.Errorf("Record() stored client %q, want ios", got)
		}
	})

	t.Run("a server-recorded event defaults to web", func(t *testing.T) {
		t.Parallel()
		st := eventstest.NewFakeStore()
		m := events.Build(events.Deps{Store: st, Snapshots: eventstest.Unscored()})

		if err := m.RecordApplicationCreated(t.Context(), nil, userID, "app-1", jobID, nil); err != nil {
			t.Fatalf("RecordApplicationCreated() err = %v", err)
		}
		if got := only(t, st).Client; got != "web" {
			t.Errorf("RecordApplicationCreated() stored client %q, want web", got)
		}
	})

	t.Run("a server-recorded event takes the request's client", func(t *testing.T) {
		t.Parallel()
		st := eventstest.NewFakeStore()
		m := events.Build(events.Deps{Store: st, Snapshots: eventstest.Unscored()})
		ctx := iosContext(t)

		if err := m.RecordApplicationCreated(ctx, nil, userID, "app-1", jobID, nil); err != nil {
			t.Fatalf("RecordApplicationCreated() err = %v", err)
		}
		if got := only(t, st).Client; got != "ios" {
			t.Errorf("RecordApplicationCreated() stored client %q, want ios", got)
		}
	})
}

func iosContext(t *testing.T) context.Context {
	t.Helper()
	var ctx context.Context
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil)
	req.Header.Set(client.Header, string(client.IOS))
	client.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ctx = r.Context()
	})).ServeHTTP(httptest.NewRecorder(), req)
	return ctx
}

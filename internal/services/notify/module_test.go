package notify_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/notify"
)

type fakeStore struct {
	summary dto.NotificationJob
	rows    []string
	err     error
}

func (f *fakeStore) GetNotificationJob(context.Context, string, string) (dto.NotificationJob, error) {
	return f.summary, nil
}

type failingLookup struct{ *fakeStore }

func (failingLookup) GetNotificationJob(context.Context, string, string) (dto.NotificationJob, error) {
	return dto.NotificationJob{}, errors.New("db")
}

func (f *fakeStore) InsertNotification(_ context.Context, userID, jobID, _, _ string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	f.rows = append(f.rows, userID+"/"+jobID)
	return "n1", nil
}

func (f *fakeStore) ListNotifications(context.Context, string) ([]dto.Notification, error) {
	return nil, nil
}

func (f *fakeStore) MarkNotificationRead(context.Context, string, string) error { return nil }

func (f *fakeStore) MarkAllNotificationsRead(context.Context, string) error { return nil }

type recordingPhone struct {
	sent []dto.PushMessage
	err  error
}

func (p *recordingPhone) Send(_ context.Context, msg dto.PushMessage) error {
	p.sent = append(p.sent, msg)
	return p.err
}

func TestModule_NewJob(t *testing.T) {
	t.Parallel()
	job := dto.Job{ID: "j1", Title: "Go Engineer", CompanySlug: "acme"}

	t.Run("records the row and sends its link", func(t *testing.T) {
		t.Parallel()
		st, phone := &fakeStore{}, &recordingPhone{}
		if err := notify.Build(notify.Deps{Store: st, Phone: phone}).NewJob(t.Context(), "u1", job); err != nil {
			t.Fatalf("NewJob() = %v", err)
		}
		if diff := cmp.Diff([]string{"u1/j1"}, st.rows); diff != "" {
			t.Errorf("rows (-want +got):\n%s", diff)
		}
		if len(phone.sent) != 1 || phone.sent[0].URL != "fasttrack://jobs/j1?n=n1" {
			t.Errorf("sent = %+v, want one message with the Notification link", phone.sent)
		}
	})

	t.Run("push body summarises the scored job, skipping empty parts", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			summary dto.NotificationJob
			want    string
		}{
			{"all parts", dto.NotificationJob{CompanyName: "Monzo", Location: "London", Band: "great", Score: new(92)}, "Monzo · London · Great 92"},
			{"no location", dto.NotificationJob{CompanyName: "Monzo", Band: "good", Score: new(75)}, "Monzo · Good 75"},
			{"unscored", dto.NotificationJob{CompanyName: "Monzo", Location: "London"}, "Monzo · London"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				phone := &recordingPhone{}
				m := notify.Build(notify.Deps{Store: &fakeStore{summary: tt.summary}, Phone: phone})
				if err := m.NewJob(t.Context(), "u1", job); err != nil {
					t.Fatalf("NewJob() = %v", err)
				}
				if len(phone.sent) != 1 || phone.sent[0].Body != tt.want {
					t.Errorf("sent = %+v, want one message with body %q", phone.sent, tt.want)
				}
			})
		}
	})

	t.Run("failing phone keeps the row and returns nil", func(t *testing.T) {
		t.Parallel()
		st := &fakeStore{}
		m := notify.Build(notify.Deps{Store: st, Phone: &recordingPhone{err: errors.New("down")}})
		if err := m.NewJob(t.Context(), "u1", job); err != nil {
			t.Errorf("NewJob() = %v, want nil", err)
		}
		if len(st.rows) != 1 {
			t.Errorf("rows = %v, want one", st.rows)
		}
	})

	t.Run("no phone still records", func(t *testing.T) {
		t.Parallel()
		st := &fakeStore{}
		if err := notify.Build(notify.Deps{Store: st}).NewJob(t.Context(), "u1", job); err != nil {
			t.Fatalf("NewJob() = %v", err)
		}
		if len(st.rows) != 1 {
			t.Errorf("rows = %v, want one", st.rows)
		}
	})

	t.Run("store failure sends nothing", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name  string
			store notify.Store
		}{
			{"insert fails", &fakeStore{err: errors.New("db")}},
			{"summary lookup fails", failingLookup{&fakeStore{}}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				phone := &recordingPhone{}
				m := notify.Build(notify.Deps{Store: tt.store, Phone: phone})
				if err := m.NewJob(t.Context(), "u1", job); err == nil {
					t.Error("NewJob() = nil, want error")
				}
				if len(phone.sent) != 0 {
					t.Errorf("sent = %v, want none", phone.sent)
				}
			})
		}
	})
}

func TestModule_NewJobByID(t *testing.T) {
	t.Parallel()
	st, phone := &fakeStore{summary: dto.NotificationJob{Title: "Go Engineer", CompanyName: "Monzo"}}, &recordingPhone{}
	if err := notify.Build(notify.Deps{Store: st, Phone: phone}).NewJobByID(t.Context(), "u1", "j1"); err != nil {
		t.Fatalf("NewJobByID() = %v", err)
	}
	if diff := cmp.Diff([]string{"u1/j1"}, st.rows); diff != "" {
		t.Errorf("rows (-want +got):\n%s", diff)
	}
	want := []dto.PushMessage{{Title: "New job: Go Engineer", Body: "Monzo", URL: "fasttrack://jobs/j1?n=n1"}}
	if diff := cmp.Diff(want, phone.sent); diff != "" {
		t.Errorf("sent (-want +got):\n%s", diff)
	}
}

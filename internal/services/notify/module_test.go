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
	rows []string
	err  error
}

func (f *fakeStore) InsertNotification(_ context.Context, userID, jobID, _, _ string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	f.rows = append(f.rows, userID+"/"+jobID)
	return "n1", nil
}

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
		phone := &recordingPhone{}
		m := notify.Build(notify.Deps{Store: &fakeStore{err: errors.New("db")}, Phone: phone})
		if err := m.NewJob(t.Context(), "u1", job); err == nil {
			t.Error("NewJob() = nil, want error")
		}
		if len(phone.sent) != 0 {
			t.Errorf("sent = %v, want none", phone.sent)
		}
	})
}

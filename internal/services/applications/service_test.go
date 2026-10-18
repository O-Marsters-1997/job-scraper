package applications_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/applications"
	"github.com/ollymarsters/job-scraper/internal/services/applications/applicationstest"
)

const userID = "user-1"

type failingCreate struct {
	applications.Store
	err error
}

func (f failingCreate) CreateApplication(context.Context, string, dto.CreateApplicationInput) (dto.Application, error) {
	return dto.Application{}, f.err
}

func newService(t *testing.T) (*applications.Service, *applicationstest.FakeStore) {
	t.Helper()
	st := applicationstest.NewFakeStore()
	return applications.NewService(st), st
}

func TestCreate(t *testing.T) {
	svc, _ := newService(t)

	got, err := svc.Create(t.Context(), userID, dto.CreateApplicationInput{JobID: "job-1", AppliedAt: new("2026-01-02")})
	if err != nil {
		t.Fatalf("Create(...) err = %v", err)
	}

	want := dto.Application{UserID: userID, JobID: "job-1", AppliedAt: new(time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC))}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(dto.Application{}, "ID", "CreatedAt", "UpdatedAt")); diff != "" {
		t.Errorf("Create(...) mismatch (-want +got):\n%s", diff)
	}
}

func TestCreateErrors(t *testing.T) {
	tests := []struct {
		name     string
		store    applications.Store
		in       dto.CreateApplicationInput
		wantKind apperr.Kind
	}{
		{
			name:     "requires job id",
			store:    applicationstest.NewFakeStore(),
			wantKind: apperr.KindInvalid,
		},
		{
			name:     "rejects invalid applied_at",
			store:    applicationstest.NewFakeStore(),
			in:       dto.CreateApplicationInput{JobID: "job-1", AppliedAt: new("not-a-date")},
			wantKind: apperr.KindInvalid,
		},
		{
			name:     "surfaces conflict from store",
			store:    failingCreate{Store: applicationstest.NewFakeStore(), err: apperr.Conflict("application already exists for this job")},
			in:       dto.CreateApplicationInput{JobID: "job-1"},
			wantKind: apperr.KindConflict,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := applications.NewService(tt.store).Create(t.Context(), userID, tt.in)
			if !apperr.IsKind(err, tt.wantKind) {
				t.Errorf("Create(%+v) err = %v, want kind %v", tt.in, err, tt.wantKind)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	svc, st := newService(t)
	created, err := st.CreateApplication(t.Context(), userID, dto.CreateApplicationInput{JobID: "job-1"})
	if err != nil {
		t.Fatalf("seed CreateApplication err = %v", err)
	}

	got, err := svc.Update(t.Context(), userID, dto.UpdateApplicationInput{ID: created.ID, SalaryInfo: "60k"})
	if err != nil {
		t.Fatalf("Update(...) err = %v", err)
	}
	if got.SalaryInfo != "60k" {
		t.Errorf("Update(...) salary = %q, want %q", got.SalaryInfo, "60k")
	}
}

func TestUpdateErrors(t *testing.T) {
	tests := []struct {
		name     string
		in       dto.UpdateApplicationInput
		wantKind apperr.Kind
	}{
		{
			name:     "rejects invalid applied_at",
			in:       dto.UpdateApplicationInput{ID: "app-1", AppliedAt: new("not-a-date")},
			wantKind: apperr.KindInvalid,
		},
		{
			name:     "surfaces not found from store",
			in:       dto.UpdateApplicationInput{ID: "missing"},
			wantKind: apperr.KindNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newService(t)
			_, err := svc.Update(t.Context(), userID, tt.in)
			if !apperr.IsKind(err, tt.wantKind) {
				t.Errorf("Update(%+v) err = %v, want kind %v", tt.in, err, tt.wantKind)
			}
		})
	}
}

func TestForJobs(t *testing.T) {
	svc, _ := newService(t)

	got, err := svc.ForJobs(t.Context(), userID, dto.ApplicationsForJobsQuery{})
	if err != nil {
		t.Fatalf("ForJobs(no ids) err = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("ForJobs(no ids) = %+v, want empty", got)
	}
}

func TestCreateStatus(t *testing.T) {
	svc, _ := newService(t)

	got, err := svc.CreateStatus(t.Context(), userID, dto.ApplicationStatusInput{Name: "Offer", Colour: "#00ff00"})
	if err != nil {
		t.Fatalf("CreateStatus(...) err = %v", err)
	}
	want := dto.ApplicationStatus{UserID: userID, Name: "Offer", Colour: "#00ff00"}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(dto.ApplicationStatus{}, "ID", "CreatedAt")); diff != "" {
		t.Errorf("CreateStatus(...) mismatch (-want +got):\n%s", diff)
	}
}

func TestCreateStatusReplyWindowBounds(t *testing.T) {
	for _, days := range []int{1, 60} {
		t.Run(fmt.Sprintf("accepts %d", days), func(t *testing.T) {
			svc, _ := newService(t)
			got, err := svc.CreateStatus(t.Context(), userID, dto.ApplicationStatusInput{Name: "Offer", Colour: "#00ff00", ReplyWindowDays: new(days)})
			if err != nil {
				t.Fatalf("CreateStatus(window %d) err = %v", days, err)
			}
			if got.ReplyWindowDays == nil || *got.ReplyWindowDays != days {
				t.Errorf("CreateStatus(window %d) ReplyWindowDays = %v, want %d", days, got.ReplyWindowDays, days)
			}
		})
	}
}

func TestUpdateStatusRejectsOutOfRangeWindow(t *testing.T) {
	svc, st := newService(t)
	created, err := st.CreateApplicationStatus(t.Context(), userID, "Applied", "#6366f1", nil)
	if err != nil {
		t.Fatalf("seed CreateApplicationStatus err = %v", err)
	}
	in := dto.ApplicationStatusInput{ID: created.ID, Name: "Applied", Colour: "#6366f1", ReplyWindowDays: new(61)}
	if _, err := svc.UpdateStatus(t.Context(), userID, in); !apperr.IsKind(err, apperr.KindInvalid) {
		t.Errorf("UpdateStatus(window 61) err = %v, want kind %v", err, apperr.KindInvalid)
	}
}

func TestCreateStatusErrors(t *testing.T) {
	tests := []struct {
		name string
		in   dto.ApplicationStatusInput
	}{
		{name: "requires name", in: dto.ApplicationStatusInput{Colour: "#00ff00"}},
		{name: "requires colour", in: dto.ApplicationStatusInput{Name: "Offer"}},
		{name: "rejects a window below 1", in: dto.ApplicationStatusInput{Name: "Offer", Colour: "#00ff00", ReplyWindowDays: new(0)}},
		{name: "rejects a window above 60", in: dto.ApplicationStatusInput{Name: "Offer", Colour: "#00ff00", ReplyWindowDays: new(61)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newService(t)
			_, err := svc.CreateStatus(t.Context(), userID, tt.in)
			if !apperr.IsKind(err, apperr.KindInvalid) {
				t.Errorf("CreateStatus(%+v) err = %v, want kind %v", tt.in, err, apperr.KindInvalid)
			}
		})
	}
}

func TestDeleteStatus(t *testing.T) {
	t.Run("refuses a status in use", func(t *testing.T) {
		svc, st := newService(t)
		status, err := st.CreateApplicationStatus(t.Context(), userID, "Applied", "#6366f1", nil)
		if err != nil {
			t.Fatalf("seed CreateApplicationStatus err = %v", err)
		}
		for range 3 {
			if _, err := st.CreateApplication(t.Context(), userID, dto.CreateApplicationInput{JobID: "job-1", StatusID: status.ID}); err != nil {
				t.Fatalf("seed CreateApplication err = %v", err)
			}
		}

		err = svc.DeleteStatus(t.Context(), userID, status.ID)

		if !apperr.IsKind(err, apperr.KindConflict) {
			t.Fatalf("DeleteStatus(in use) err = %v, want kind %v", err, apperr.KindConflict)
		}
		if got := apperr.FieldsFor(err)["count"]; got != int64(3) {
			t.Errorf("DeleteStatus(in use) count = %v, want 3", got)
		}
	})

	t.Run("deletes an unused status", func(t *testing.T) {
		svc, _ := newService(t)
		if err := svc.DeleteStatus(t.Context(), userID, "s1"); err != nil {
			t.Errorf("DeleteStatus(unused) err = %v", err)
		}
	})
}

func TestSetChase(t *testing.T) {
	svc, st := newService(t)
	created, err := st.CreateApplication(t.Context(), userID, dto.CreateApplicationInput{JobID: "job-1"})
	if err != nil {
		t.Fatalf("seed CreateApplication err = %v", err)
	}

	got, err := svc.SetChase(t.Context(), userID, dto.ChaseInput{ID: created.ID, ChaseBy: "2026-10-20"})
	if err != nil {
		t.Fatalf("SetChase(...) err = %v", err)
	}
	want := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	if got.ChaseBy == nil || !got.ChaseBy.Equal(want) {
		t.Errorf("SetChase(2026-10-20) ChaseBy = %v, want %v", got.ChaseBy, want)
	}
}

func TestSetChaseErrors(t *testing.T) {
	tests := []struct {
		name     string
		in       dto.ChaseInput
		wantKind apperr.Kind
	}{
		{
			name:     "rejects a malformed date",
			in:       dto.ChaseInput{ID: "app-1", ChaseBy: "20/10/2026"},
			wantKind: apperr.KindInvalid,
		},
		{
			name:     "rejects an empty date",
			in:       dto.ChaseInput{ID: "app-1"},
			wantKind: apperr.KindInvalid,
		},
		{
			name:     "surfaces not found from store",
			in:       dto.ChaseInput{ID: "missing", ChaseBy: "2026-10-20"},
			wantKind: apperr.KindNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newService(t)
			_, err := svc.SetChase(t.Context(), userID, tt.in)
			if !apperr.IsKind(err, tt.wantKind) {
				t.Errorf("SetChase(%+v) err = %v, want kind %v", tt.in, err, tt.wantKind)
			}
		})
	}
}

func TestCreateJobNote(t *testing.T) {
	t.Parallel()
	t.Run("records the note", func(t *testing.T) {
		t.Parallel()
		svc, _ := newService(t)
		got, err := svc.CreateJobNote(t.Context(), userID, dto.CreateJobNoteInput{JobID: "job-1", Body: "call back Friday"})
		if err != nil {
			t.Fatalf("CreateJobNote(...) err = %v", err)
		}
		want := dto.JobNote{JobID: "job-1", Body: "call back Friday"}
		if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(dto.JobNote{}, "ID", "CreatedAt", "UpdatedAt")); diff != "" {
			t.Errorf("CreateJobNote(...) mismatch (-want +got):\n%s", diff)
		}
	})

	for name, body := range map[string]string{
		"empty body":    "",
		"oversize body": strings.Repeat("é", 10001),
	} {
		t.Run(name+" is rejected", func(t *testing.T) {
			t.Parallel()
			svc, _ := newService(t)
			_, err := svc.CreateJobNote(t.Context(), userID, dto.CreateJobNoteInput{JobID: "job-1", Body: body})
			if !apperr.IsKind(err, apperr.KindInvalid) {
				t.Errorf("CreateJobNote(%s) err = %v, want invalid", name, err)
			}
		})
	}

	t.Run("10000 characters is accepted", func(t *testing.T) {
		t.Parallel()
		svc, _ := newService(t)
		if _, err := svc.CreateJobNote(t.Context(), userID, dto.CreateJobNoteInput{JobID: "job-1", Body: strings.Repeat("é", 10000)}); err != nil {
			t.Errorf("CreateJobNote(10000 chars) err = %v", err)
		}
	})
}

func TestLogRecruiterContact(t *testing.T) {
	valid := dto.LogRecruiterContactInput{JobID: "job-1", ContactedOn: "2026-10-01", Channel: dto.ChannelEmail, Outcome: dto.OutcomeSpoke}

	t.Run("records the contact", func(t *testing.T) {
		svc, _ := newService(t)
		got, err := svc.LogRecruiterContact(t.Context(), userID, valid)
		if err != nil {
			t.Fatalf("LogRecruiterContact(...) err = %v", err)
		}
		want := dto.RecruiterContact{JobID: "job-1", ContactedOn: "2026-10-01", Channel: dto.ChannelEmail, Outcome: dto.OutcomeSpoke}
		if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(dto.RecruiterContact{}, "ID", "CreatedAt")); diff != "" {
			t.Errorf("LogRecruiterContact(...) mismatch (-want +got):\n%s", diff)
		}
	})

	invalid := map[string]func(*dto.LogRecruiterContactInput){
		"bad date":    func(in *dto.LogRecruiterContactInput) { in.ContactedOn = "yesterday" },
		"bad channel": func(in *dto.LogRecruiterContactInput) { in.Channel = "carrier_pigeon" },
		"bad outcome": func(in *dto.LogRecruiterContactInput) { in.Outcome = "ghosted" },
	}
	for name, mutate := range invalid {
		t.Run(name+" is rejected", func(t *testing.T) {
			svc, _ := newService(t)
			in := valid
			mutate(&in)
			_, err := svc.LogRecruiterContact(t.Context(), userID, in)
			if !apperr.IsKind(err, apperr.KindInvalid) {
				t.Errorf("LogRecruiterContact(%s) err = %v, want invalid", name, err)
			}
		})
	}
}

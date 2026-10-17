package tasks_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/tasks"
	"github.com/ollymarsters/job-scraper/internal/services/tasks/taskstest"
)

const (
	userID  = "user-1"
	validID = "11111111-1111-1111-1111-111111111111"
)

func newService(t *testing.T) (*tasks.Service, *taskstest.FakeStore) {
	t.Helper()
	st := taskstest.NewFakeStore()
	return tasks.NewService(st), st
}

func TestCreate(t *testing.T) {
	t.Parallel()
	svc, st := newService(t)
	typeID := st.AddType(userID, "Apply").ID

	got, err := svc.Create(t.Context(), userID, dto.CreateTaskInput{TypeID: typeID, Link: " https://example.com/a ", Notes: "hi"})
	if err != nil {
		t.Fatalf("Create(...) err = %v", err)
	}
	if got.Link != "https://example.com/a" || got.Notes != "hi" || got.TypeName != "Apply" {
		t.Errorf("Create(...) = %+v, want trimmed link, notes and type name", got)
	}
}

func TestCreateErrors(t *testing.T) {
	t.Parallel()
	svc, st := newService(t)
	typeID := st.AddType(userID, "Apply").ID

	tests := []struct {
		name     string
		in       dto.CreateTaskInput
		wantKind apperr.Kind
	}{
		{"missing type", dto.CreateTaskInput{}, apperr.KindInvalid},
		{"malformed job id", dto.CreateTaskInput{TypeID: typeID, JobID: new("nope")}, apperr.KindInvalid},
		{"javascript link", dto.CreateTaskInput{TypeID: typeID, Link: "javascript:alert(1)"}, apperr.KindInvalid},
		{"data link", dto.CreateTaskInput{TypeID: typeID, Link: "data:text/html,<b>x</b>"}, apperr.KindInvalid},
		{"ftp link", dto.CreateTaskInput{TypeID: typeID, Link: "ftp://example.com/x"}, apperr.KindInvalid},
		{"link without host", dto.CreateTaskInput{TypeID: typeID, Link: "https://"}, apperr.KindInvalid},
		{"link too long", dto.CreateTaskInput{TypeID: typeID, Link: "https://example.com/" + strings.Repeat("a", 2048)}, apperr.KindInvalid},
		{"notes too long", dto.CreateTaskInput{TypeID: typeID, Notes: strings.Repeat("a", 10001)}, apperr.KindInvalid},
		{"another user's type", dto.CreateTaskInput{TypeID: validID}, apperr.KindNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.Create(t.Context(), userID, tt.in)
			if !apperr.IsKind(err, tt.wantKind) {
				t.Errorf("Create(%+v) err = %v, want kind %v", tt.in, err, tt.wantKind)
			}
		})
	}
}

func TestTypeValidation(t *testing.T) {
	t.Parallel()
	svc, st := newService(t)
	existing := st.AddType(userID, "Apply")

	got, err := svc.CreateType(t.Context(), userID, dto.TaskTypeInput{Name: "  Offer ", Colour: "#15803D"})
	if err != nil {
		t.Fatalf("CreateType(...) err = %v", err)
	}
	if got.Name != "Offer" {
		t.Errorf("CreateType(padded name) Name = %q, want trimmed", got.Name)
	}

	tests := []struct {
		name string
		in   dto.TaskTypeInput
	}{
		{"blank name", dto.TaskTypeInput{Name: "  ", Colour: "#15803d"}},
		{"name too long", dto.TaskTypeInput{Name: strings.Repeat("a", 51), Colour: "#15803d"}},
		{"missing colour", dto.TaskTypeInput{Name: "Offer2"}},
		{"colour not hex", dto.TaskTypeInput{Name: "Offer2", Colour: "red"}},
		{"colour injects css", dto.TaskTypeInput{Name: "Offer2", Colour: "#15803d;x"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := svc.CreateType(t.Context(), userID, tt.in); !apperr.IsKind(err, apperr.KindInvalid) {
				t.Errorf("CreateType(%+v) err = %v, want invalid", tt.in, err)
			}
			tt.in.ID = existing.ID
			if _, err := svc.UpdateType(t.Context(), userID, tt.in); !apperr.IsKind(err, apperr.KindInvalid) {
				t.Errorf("UpdateType(%+v) err = %v, want invalid", tt.in, err)
			}
		})
	}
}

func TestList(t *testing.T) {
	t.Parallel()
	svc, st := newService(t)
	typeID := st.AddType(userID, "Apply").ID
	open, err := svc.Create(t.Context(), userID, dto.CreateTaskInput{TypeID: typeID})
	if err != nil {
		t.Fatalf("seed Create err = %v", err)
	}
	done, err := svc.Create(t.Context(), userID, dto.CreateTaskInput{TypeID: typeID})
	if err != nil {
		t.Fatalf("seed Create err = %v", err)
	}
	if _, err := svc.SetDone(t.Context(), userID, dto.TaskDoneInput{ID: done.ID, Done: true}); err != nil {
		t.Fatalf("seed SetDone err = %v", err)
	}

	tests := []struct {
		name string
		q    dto.TasksQuery
		want []string
	}{
		{"defaults to open tasks", dto.TasksQuery{}, []string{open.ID}},
		{"done=true lists completed tasks", dto.TasksQuery{Done: "true"}, []string{done.ID}},
		{"page past the end is empty", dto.TasksQuery{Page: "9"}, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			page, err := svc.List(t.Context(), userID, tt.q)
			if err != nil {
				t.Fatalf("List(%+v) err = %v", tt.q, err)
			}
			got := []string{}
			for _, it := range page.Items {
				got = append(got, it.ID)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("List(%+v) ids mismatch (-want +got):\n%s", tt.q, diff)
			}
		})
	}
}

func TestListErrors(t *testing.T) {
	t.Parallel()
	svc, _ := newService(t)
	tests := []struct {
		name string
		q    dto.TasksQuery
	}{
		{"malformed type id", dto.TasksQuery{TypeID: "nope"}},
		{"malformed job id", dto.TasksQuery{JobID: "nope"}},
		{"bad done", dto.TasksQuery{Done: "maybe"}},
		{"unknown sort", dto.TasksQuery{Sort: "random"}},
		{"page zero", dto.TasksQuery{Page: "0"}},
		{"page_size not a number", dto.TasksQuery{PageSize: "many"}},
		{"page overflows the offset", dto.TasksQuery{Page: "2147483647", PageSize: "100"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := svc.List(t.Context(), userID, tt.q); !apperr.IsKind(err, apperr.KindInvalid) {
				t.Errorf("List(%+v) err = %v, want invalid", tt.q, err)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	svc, st := newService(t)
	apply := st.AddType(userID, "Apply")
	prep := st.AddType(userID, "Prep")
	created, err := svc.Create(t.Context(), userID, dto.CreateTaskInput{TypeID: apply.ID, JobID: new(validID), Notes: "old"})
	if err != nil {
		t.Fatalf("seed Create err = %v", err)
	}

	got, err := svc.Update(t.Context(), userID, dto.UpdateTaskInput{ID: created.ID, TypeID: prep.ID, Link: " https://example.com/b ", Notes: "new"})
	if err != nil {
		t.Fatalf("Update(...) err = %v", err)
	}
	if got.TypeName != "Prep" || got.Link != "https://example.com/b" || got.Notes != "new" {
		t.Errorf("Update(...) = %+v, want Prep, trimmed link and new notes", got)
	}
	if got.JobID == nil || *got.JobID != validID {
		t.Errorf("Update(...) JobID = %v, want %s", got.JobID, validID)
	}
}

func TestUpdateErrors(t *testing.T) {
	t.Parallel()
	svc, st := newService(t)
	typeID := st.AddType(userID, "Apply").ID
	created, err := svc.Create(t.Context(), userID, dto.CreateTaskInput{TypeID: typeID})
	if err != nil {
		t.Fatalf("seed Create err = %v", err)
	}

	tests := []struct {
		name     string
		userID   string
		in       dto.UpdateTaskInput
		wantKind apperr.Kind
	}{
		{"malformed type id", userID, dto.UpdateTaskInput{ID: created.ID, TypeID: "nope"}, apperr.KindInvalid},
		{"javascript link", userID, dto.UpdateTaskInput{ID: created.ID, TypeID: typeID, Link: "javascript:alert(1)"}, apperr.KindInvalid},
		{"notes too long", userID, dto.UpdateTaskInput{ID: created.ID, TypeID: typeID, Notes: strings.Repeat("a", 10001)}, apperr.KindInvalid},
		{"another user's task", "user-2", dto.UpdateTaskInput{ID: created.ID, TypeID: typeID}, apperr.KindNotFound},
		{"another user's type", userID, dto.UpdateTaskInput{ID: created.ID, TypeID: validID}, apperr.KindNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.Update(t.Context(), tt.userID, tt.in)
			if !apperr.IsKind(err, tt.wantKind) {
				t.Errorf("Update(%+v) err = %v, want kind %v", tt.in, err, tt.wantKind)
			}
		})
	}
}

func TestSetDoneOnMissingTaskReturnsNotFound(t *testing.T) {
	t.Parallel()
	svc, _ := newService(t)
	if _, err := svc.SetDone(t.Context(), userID, dto.TaskDoneInput{ID: validID, Done: true}); !apperr.IsKind(err, apperr.KindNotFound) {
		t.Errorf("SetDone(missing) err = %v, want not found", err)
	}
}

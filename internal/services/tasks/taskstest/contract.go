package taskstest

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/tasks"
	"github.com/ollymarsters/job-scraper/internal/services/tasks/store"
)

const missingID = "00000000-0000-0000-0000-000000000000"

// Fixture is one user with one Task type, a second user with their own type,
// and a Job both can reference.
type Fixture struct {
	Store       tasks.Store
	UserID      string
	OtherUserID string
	TypeID      string
	OtherTypeID string
	JobID       string
}

func createTask(t *testing.T, f Fixture, in dto.CreateTaskInput) dto.Task {
	t.Helper()
	if in.TypeID == "" {
		in.TypeID = f.TypeID
	}
	got, err := f.Store.CreateTask(t.Context(), f.UserID, in)
	if err != nil {
		t.Fatalf("CreateTask(%+v) err = %v", in, err)
	}
	return got
}

func listTasks(t *testing.T, f Fixture, opts dto.TaskPageOptions) dto.TaskPage {
	t.Helper()
	if opts.Limit == 0 {
		opts.Limit = 25
	}
	got, err := f.Store.ListTasks(t.Context(), f.UserID, opts)
	if err != nil {
		t.Fatalf("ListTasks(%+v) err = %v", opts, err)
	}
	return got
}

func ids(items []dto.Task) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.ID
	}
	return out
}

// RunStoreContract proves newStore's tasks.Store behaves the same whether it's
// the fake or the real store (ADR 0012). SeedDefaultTaskTypes is a tx-scoped
// port, covered in store/store_test.go.
func RunStoreContract(t *testing.T, newStore func(t *testing.T) Fixture) {
	t.Helper()

	t.Run("create returns the task with its type", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		got := createTask(t, f, dto.CreateTaskInput{JobID: &f.JobID, Link: "https://example.com/x", Notes: "call back"})
		want := dto.Task{
			TypeID: f.TypeID, TypeName: "Apply", TypeColour: "#6366f1",
			JobID: &f.JobID, Link: "https://example.com/x", Notes: "call back",
		}
		opts := cmpopts.IgnoreFields(dto.Task{}, "ID", "JobTitle", "JobCompanySlug", "JobPostedAt", "CreatedAt", "UpdatedAt")
		if diff := cmp.Diff(want, got, opts); diff != "" {
			t.Errorf("CreateTask(...) mismatch (-want +got):\n%s", diff)
		}
		if got.ID == "" || got.CompletedAt != nil {
			t.Errorf("CreateTask(...) = %+v, want an ID and no CompletedAt", got)
		}
	})

	t.Run("create without job, link or notes leaves them empty", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		got := createTask(t, f, dto.CreateTaskInput{})
		if got.JobID != nil || got.Link != "" || got.Notes != "" {
			t.Errorf("CreateTask(empty) = %+v, want no job, link or notes", got)
		}
	})

	t.Run("create with another user's type returns not found", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		_, err := f.Store.CreateTask(t.Context(), f.UserID, dto.CreateTaskInput{TypeID: f.OtherTypeID})
		if !errors.Is(err, data.ErrNotFound) {
			t.Errorf("CreateTask(other user's type) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("list types returns only the user's", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		got, err := f.Store.ListTaskTypes(t.Context(), f.UserID)
		if err != nil {
			t.Fatalf("ListTaskTypes(...) err = %v", err)
		}
		if len(got) != 1 || got[0].ID != f.TypeID {
			t.Errorf("ListTaskTypes(...) = %+v, want only type %s", got, f.TypeID)
		}
	})

	t.Run("task types are created, renamed and recoloured with their task count", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		created, err := f.Store.CreateTaskType(t.Context(), f.UserID, dto.TaskTypeInput{Name: "Offer", Colour: "#15803d"})
		if err != nil {
			t.Fatalf("CreateTaskType(...) err = %v", err)
		}
		if created.ID == "" || created.Name != "Offer" || created.Colour != "#15803d" || created.TaskCount != 0 {
			t.Errorf("CreateTaskType(...) = %+v, want Offer, #15803d, no tasks", created)
		}
		createTask(t, f, dto.CreateTaskInput{TypeID: created.ID})
		createTask(t, f, dto.CreateTaskInput{TypeID: created.ID})

		updated, err := f.Store.UpdateTaskType(t.Context(), f.UserID, dto.TaskTypeInput{ID: created.ID, Name: "Offers", Colour: "#000000"})
		if err != nil {
			t.Fatalf("UpdateTaskType(...) err = %v", err)
		}
		want := dto.TaskType{ID: created.ID, Name: "Offers", Colour: "#000000", TaskCount: 2}
		if diff := cmp.Diff(want, updated, cmpopts.IgnoreFields(dto.TaskType{}, "CreatedAt")); diff != "" {
			t.Errorf("UpdateTaskType(...) mismatch (-want +got):\n%s", diff)
		}
		tasks := listTasks(t, f, dto.TaskPageOptions{TypeID: created.ID, Sort: dto.TaskSortCreated})
		if tasks.Total != 2 || tasks.Items[0].TypeName != "Offers" || tasks.Items[0].TypeColour != "#000000" {
			t.Errorf("ListTasks(renamed type) = %+v, want 2 tasks carrying the new name and colour", tasks)
		}
	})

	t.Run("task type names are unique per user", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		if _, err := f.Store.CreateTaskType(t.Context(), f.UserID, dto.TaskTypeInput{Name: "Apply", Colour: "#000000"}); !errors.Is(err, store.ErrTaskTypeExists) {
			t.Errorf("CreateTaskType(duplicate) err = %v, want ErrTaskTypeExists", err)
		}
		if _, err := f.Store.CreateTaskType(t.Context(), f.OtherUserID, dto.TaskTypeInput{Name: "Reach out", Colour: "#000000"}); err != nil {
			t.Fatalf("CreateTaskType(other user) err = %v", err)
		}
		second, err := f.Store.CreateTaskType(t.Context(), f.UserID, dto.TaskTypeInput{Name: "Prep", Colour: "#000000"})
		if err != nil {
			t.Fatalf("CreateTaskType(Prep) err = %v", err)
		}
		if _, err := f.Store.UpdateTaskType(t.Context(), f.UserID, dto.TaskTypeInput{ID: second.ID, Name: "Apply", Colour: "#000000"}); !errors.Is(err, store.ErrTaskTypeExists) {
			t.Errorf("UpdateTaskType(duplicate) err = %v, want ErrTaskTypeExists", err)
		}
		if _, err := f.Store.UpdateTaskType(t.Context(), f.UserID, dto.TaskTypeInput{ID: second.ID, Name: "Prep", Colour: "#ffffff"}); err != nil {
			t.Errorf("UpdateTaskType(own name) err = %v, want nil", err)
		}
	})

	t.Run("deleting a task type deletes its tasks", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		createTask(t, f, dto.CreateTaskInput{})
		if err := f.Store.DeleteTaskType(t.Context(), f.UserID, f.TypeID); err != nil {
			t.Fatalf("DeleteTaskType(...) err = %v", err)
		}
		if got := listTasks(t, f, dto.TaskPageOptions{Sort: dto.TaskSortCreated}); got.Total != 0 {
			t.Errorf("ListTasks(after delete) = %+v, want empty", got)
		}
		types, err := f.Store.ListTaskTypes(t.Context(), f.UserID)
		if err != nil || len(types) != 0 {
			t.Errorf("ListTaskTypes(after delete) = %+v, %v, want none", types, err)
		}
	})

	t.Run("changing another user's or a missing task type returns not found", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		in := dto.TaskTypeInput{ID: f.OtherTypeID, Name: "Hijack", Colour: "#000000"}
		if _, err := f.Store.UpdateTaskType(t.Context(), f.UserID, in); !errors.Is(err, data.ErrNotFound) {
			t.Errorf("UpdateTaskType(other user's) err = %v, want ErrNotFound", err)
		}
		if err := f.Store.DeleteTaskType(t.Context(), f.UserID, f.OtherTypeID); !errors.Is(err, data.ErrNotFound) {
			t.Errorf("DeleteTaskType(other user's) err = %v, want ErrNotFound", err)
		}
		if err := f.Store.DeleteTaskType(t.Context(), f.UserID, missingID); !errors.Is(err, data.ErrNotFound) {
			t.Errorf("DeleteTaskType(missing) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("list returns open tasks newest first", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		first := createTask(t, f, dto.CreateTaskInput{})
		second := createTask(t, f, dto.CreateTaskInput{})
		done := createTask(t, f, dto.CreateTaskInput{})
		if _, err := f.Store.SetTaskDone(t.Context(), f.UserID, done.ID, true, time.Now()); err != nil {
			t.Fatalf("SetTaskDone(...) err = %v", err)
		}

		got := listTasks(t, f, dto.TaskPageOptions{Sort: dto.TaskSortCreated})
		if diff := cmp.Diff([]string{second.ID, first.ID}, ids(got.Items)); diff != "" {
			t.Errorf("ListTasks(open) ids mismatch (-want +got):\n%s", diff)
		}
		if got.Total != 2 {
			t.Errorf("ListTasks(open) Total = %d, want 2", got.Total)
		}
		doneList := listTasks(t, f, dto.TaskPageOptions{Done: true, Sort: dto.TaskSortCreated})
		if diff := cmp.Diff([]string{done.ID}, ids(doneList.Items)); diff != "" {
			t.Errorf("ListTasks(done) ids mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("list filters by type and job", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		withJob := createTask(t, f, dto.CreateTaskInput{JobID: &f.JobID})
		plain := createTask(t, f, dto.CreateTaskInput{})

		byJob := listTasks(t, f, dto.TaskPageOptions{JobID: f.JobID, Sort: dto.TaskSortCreated})
		if diff := cmp.Diff([]string{withJob.ID}, ids(byJob.Items)); diff != "" {
			t.Errorf("ListTasks(job) ids mismatch (-want +got):\n%s", diff)
		}
		byType := listTasks(t, f, dto.TaskPageOptions{TypeID: f.TypeID, Sort: dto.TaskSortCreated})
		if diff := cmp.Diff([]string{plain.ID, withJob.ID}, ids(byType.Items)); diff != "" {
			t.Errorf("ListTasks(type) ids mismatch (-want +got):\n%s", diff)
		}
		other := listTasks(t, f, dto.TaskPageOptions{TypeID: f.OtherTypeID, Sort: dto.TaskSortCreated})
		if len(other.Items) != 0 || other.Total != 0 {
			t.Errorf("ListTasks(other user's type) = %+v, want empty", other)
		}
	})

	t.Run("list pages with a total", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		oldest := createTask(t, f, dto.CreateTaskInput{})
		createTask(t, f, dto.CreateTaskInput{})
		createTask(t, f, dto.CreateTaskInput{})

		got := listTasks(t, f, dto.TaskPageOptions{Sort: dto.TaskSortCreated, Limit: 2, Offset: 2})
		if diff := cmp.Diff([]string{oldest.ID}, ids(got.Items)); diff != "" {
			t.Errorf("ListTasks(page 2) ids mismatch (-want +got):\n%s", diff)
		}
		if got.Total != 3 {
			t.Errorf("ListTasks(page 2) Total = %d, want 3", got.Total)
		}
	})

	t.Run("list never returns another user's tasks", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		if _, err := f.Store.CreateTask(t.Context(), f.OtherUserID, dto.CreateTaskInput{TypeID: f.OtherTypeID}); err != nil {
			t.Fatalf("CreateTask(other user) err = %v", err)
		}
		if got := listTasks(t, f, dto.TaskPageOptions{Sort: dto.TaskSortCreated}); len(got.Items) != 0 || got.Total != 0 {
			t.Errorf("ListTasks(...) = %+v, want empty", got)
		}
	})

	t.Run("set done stamps the time and set not done clears it", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		created := createTask(t, f, dto.CreateTaskInput{})
		now := time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)

		got, err := f.Store.SetTaskDone(t.Context(), f.UserID, created.ID, true, now)
		if err != nil {
			t.Fatalf("SetTaskDone(true) err = %v", err)
		}
		if got.CompletedAt == nil || !got.CompletedAt.Equal(now) {
			t.Errorf("SetTaskDone(true) CompletedAt = %v, want %v", got.CompletedAt, now)
		}
		got, err = f.Store.SetTaskDone(t.Context(), f.UserID, created.ID, true, now.Add(time.Hour))
		if err != nil {
			t.Fatalf("SetTaskDone(true) again err = %v", err)
		}
		if got.CompletedAt == nil || !got.CompletedAt.Equal(now) {
			t.Errorf("SetTaskDone(true) again CompletedAt = %v, want the first completion %v", got.CompletedAt, now)
		}
		got, err = f.Store.SetTaskDone(t.Context(), f.UserID, created.ID, false, now)
		if err != nil {
			t.Fatalf("SetTaskDone(false) err = %v", err)
		}
		if got.CompletedAt != nil {
			t.Errorf("SetTaskDone(false) CompletedAt = %v, want nil", got.CompletedAt)
		}
	})

	t.Run("set done on a missing or another user's task returns not found", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		created := createTask(t, f, dto.CreateTaskInput{})
		if _, err := f.Store.SetTaskDone(t.Context(), f.UserID, missingID, true, time.Now()); !errors.Is(err, data.ErrNotFound) {
			t.Errorf("SetTaskDone(missing) err = %v, want ErrNotFound", err)
		}
		if _, err := f.Store.SetTaskDone(t.Context(), f.OtherUserID, created.ID, true, time.Now()); !errors.Is(err, data.ErrNotFound) {
			t.Errorf("SetTaskDone(other user's task) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("update changes type, link and notes but never the job", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		created := createTask(t, f, dto.CreateTaskInput{JobID: &f.JobID, Link: "https://example.com/a", Notes: "old"})
		other, err := f.Store.CreateTaskType(t.Context(), f.UserID, dto.TaskTypeInput{Name: "Prep", Colour: "#64748b"})
		if err != nil {
			t.Fatalf("CreateTaskType(Prep) err = %v", err)
		}
		now := time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)

		got, err := f.Store.UpdateTask(t.Context(), f.UserID, dto.UpdateTaskInput{ID: created.ID, TypeID: other.ID, Link: "https://example.org/b", Notes: "new"}, now)
		if err != nil {
			t.Fatalf("UpdateTask(...) err = %v", err)
		}
		if got.TypeID != other.ID || got.TypeName != "Prep" || got.Link != "https://example.org/b" || got.Notes != "new" || !got.UpdatedAt.Equal(now) {
			t.Errorf("UpdateTask(...) = %+v, want Prep type, new link and notes, UpdatedAt %v", got, now)
		}
		if got.JobID == nil || *got.JobID != f.JobID {
			t.Errorf("UpdateTask(...) JobID = %v, want %s", got.JobID, f.JobID)
		}

		cleared, err := f.Store.UpdateTask(t.Context(), f.UserID, dto.UpdateTaskInput{ID: created.ID, TypeID: other.ID}, now)
		if err != nil {
			t.Fatalf("UpdateTask(clear) err = %v", err)
		}
		if cleared.Link != "" || cleared.Notes != "" {
			t.Errorf("UpdateTask(clear) = %+v, want empty link and notes", cleared)
		}
	})

	t.Run("update with a missing task, another user's task or another user's type returns not found", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		created := createTask(t, f, dto.CreateTaskInput{Notes: "keep"})
		cases := map[string]struct {
			userID string
			in     dto.UpdateTaskInput
		}{
			"missing task":      {f.UserID, dto.UpdateTaskInput{ID: missingID, TypeID: f.TypeID}},
			"other user's task": {f.OtherUserID, dto.UpdateTaskInput{ID: created.ID, TypeID: f.OtherTypeID}},
			"other user's type": {f.UserID, dto.UpdateTaskInput{ID: created.ID, TypeID: f.OtherTypeID}},
		}
		for name, tc := range cases {
			if _, err := f.Store.UpdateTask(t.Context(), tc.userID, tc.in, time.Now()); !errors.Is(err, data.ErrNotFound) {
				t.Errorf("UpdateTask(%s) err = %v, want ErrNotFound", name, err)
			}
		}
		got := listTasks(t, f, dto.TaskPageOptions{Sort: dto.TaskSortCreated})
		if len(got.Items) != 1 || got.Items[0].Notes != "keep" || got.Items[0].TypeID != f.TypeID {
			t.Errorf("ListTasks after rejected updates = %+v, want the task untouched", got.Items)
		}
	})
}

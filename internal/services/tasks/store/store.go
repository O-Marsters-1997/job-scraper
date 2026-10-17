// Package store is the tasks context's Postgres store: tasks and task_types,
// plus a read-join onto jobs for each Task's Job summary.
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/tasks/store/sqlc"
)

type DBTX = sqlc.DBTX

var ErrTaskTypeExists = errors.New("task type name already exists")

type Store struct {
	queries *sqlc.Queries
}

func New(db DBTX) *Store {
	return &Store{queries: sqlc.New(db)}
}

func (s *Store) SeedDefaultTaskTypes(ctx context.Context, tx pgx.Tx, userID string) error {
	uid, err := data.UUID(userID)
	if err != nil {
		return err
	}
	if err := s.queries.WithTx(tx).SeedDefaultTaskTypes(ctx, uid); err != nil {
		return fmt.Errorf("store.SeedDefaultTaskTypes: %w", err)
	}
	return nil
}

func (s *Store) ListTaskTypes(ctx context.Context, userID string) ([]dto.TaskType, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListTaskTypes(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListTaskTypes: %w", err)
	}
	out := make([]dto.TaskType, len(rows))
	for i, row := range rows {
		out[i] = toTaskTypeDTO(row)
	}
	return out, nil
}

func (s *Store) CreateTaskType(ctx context.Context, userID string, in dto.TaskTypeInput) (dto.TaskType, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.TaskType{}, err
	}
	row, err := s.queries.CreateTaskType(ctx, sqlc.CreateTaskTypeParams{UserID: uid, Name: in.Name, Colour: in.Colour})
	if err != nil {
		return dto.TaskType{}, taskTypeErr("CreateTaskType", err)
	}
	return toTaskTypeDTO(sqlc.ListTaskTypesRow(row)), nil
}

func (s *Store) UpdateTaskType(ctx context.Context, userID string, in dto.TaskTypeInput) (dto.TaskType, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.TaskType{}, err
	}
	id, err := data.UUID(in.ID)
	if err != nil {
		return dto.TaskType{}, data.ErrNotFound
	}
	row, err := s.queries.UpdateTaskType(ctx, sqlc.UpdateTaskTypeParams{ID: id, UserID: uid, Name: in.Name, Colour: in.Colour})
	if err != nil {
		return dto.TaskType{}, taskTypeErr("UpdateTaskType", err)
	}
	return toTaskTypeDTO(sqlc.ListTaskTypesRow(row)), nil
}

func (s *Store) DeleteTaskType(ctx context.Context, userID, id string) error {
	uid, err := data.UUID(userID)
	if err != nil {
		return err
	}
	tid, err := data.UUID(id)
	if err != nil {
		return data.ErrNotFound
	}
	n, err := s.queries.DeleteTaskType(ctx, sqlc.DeleteTaskTypeParams{ID: tid, UserID: uid})
	if err != nil {
		return data.QueryErr("DeleteTaskType", err)
	}
	if n == 0 {
		return data.ErrNotFound
	}
	return nil
}

func taskTypeErr(op string, err error) error {
	if data.IsUniqueViolation(err) {
		return ErrTaskTypeExists
	}
	return data.QueryErr(op, err)
}

func (s *Store) ListTasks(ctx context.Context, userID string, opts dto.TaskPageOptions) (dto.TaskPage, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.TaskPage{}, err
	}
	typeID, err := optionalUUID(opts.TypeID)
	if err != nil {
		return dto.TaskPage{}, err
	}
	jobID, err := optionalUUID(opts.JobID)
	if err != nil {
		return dto.TaskPage{}, err
	}
	rows, err := s.queries.PageTasks(ctx, sqlc.PageTasksParams{
		UserID: uid, TypeID: typeID, JobID: jobID, Done: opts.Done,
		Sort: string(opts.Sort), PageLimit: opts.Limit, PageOffset: opts.Offset,
	})
	if err != nil {
		return dto.TaskPage{}, fmt.Errorf("store.ListTasks page: %w", err)
	}
	total, err := s.queries.CountTasks(ctx, sqlc.CountTasksParams{UserID: uid, TypeID: typeID, JobID: jobID, Done: opts.Done})
	if err != nil {
		return dto.TaskPage{}, fmt.Errorf("store.ListTasks count: %w", err)
	}
	items := make([]dto.Task, len(rows))
	for i, row := range rows {
		items[i] = toTaskDTO(row)
	}
	return dto.TaskPage{Items: items, Total: int(total)}, nil
}

func (s *Store) CreateTask(ctx context.Context, userID string, in dto.CreateTaskInput) (dto.Task, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.Task{}, err
	}
	typeID, err := data.UUID(in.TypeID)
	if err != nil {
		return dto.Task{}, data.ErrNotFound
	}
	var jobID pgtype.UUID
	if in.JobID != nil {
		if jobID, err = data.UUID(*in.JobID); err != nil {
			return dto.Task{}, data.ErrNotFound
		}
	}
	row, err := s.queries.CreateTask(ctx, sqlc.CreateTaskParams{
		UserID: uid, TypeID: typeID, JobID: jobID, Link: data.Text(in.Link), Notes: data.Text(in.Notes),
	})
	if err != nil {
		if data.IsForeignKeyViolation(err, "tasks_job_id_fkey") {
			return dto.Task{}, data.ErrNotFound
		}
		return dto.Task{}, data.QueryErr("CreateTask", err)
	}
	return toTaskDTO(sqlc.PageTasksRow(row)), nil
}

func (s *Store) UpdateTask(ctx context.Context, userID string, in dto.UpdateTaskInput, now time.Time) (dto.Task, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.Task{}, err
	}
	id, err := data.UUID(in.ID)
	if err != nil {
		return dto.Task{}, data.ErrNotFound
	}
	typeID, err := data.UUID(in.TypeID)
	if err != nil {
		return dto.Task{}, data.ErrNotFound
	}
	row, err := s.queries.UpdateTask(ctx, sqlc.UpdateTaskParams{
		ID: id, UserID: uid, TypeID: typeID, Link: data.Text(in.Link), Notes: data.Text(in.Notes),
		Now: pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		return dto.Task{}, data.QueryErr("UpdateTask", err)
	}
	return toTaskDTO(sqlc.PageTasksRow(row)), nil
}

func (s *Store) SetTaskDone(ctx context.Context, userID, id string, done bool, now time.Time) (dto.Task, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.Task{}, err
	}
	tid, err := data.UUID(id)
	if err != nil {
		return dto.Task{}, data.ErrNotFound
	}
	row, err := s.queries.SetTaskDone(ctx, sqlc.SetTaskDoneParams{
		ID: tid, UserID: uid, Done: done, Now: pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		return dto.Task{}, data.QueryErr("SetTaskDone", err)
	}
	return toTaskDTO(sqlc.PageTasksRow(row)), nil
}

func optionalUUID(s string) (pgtype.UUID, error) {
	if s == "" {
		return pgtype.UUID{}, nil
	}
	return data.UUID(s)
}

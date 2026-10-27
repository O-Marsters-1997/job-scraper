package tasks

import (
	"cmp"
	"context"
	"errors"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/tasks/store"
)

const (
	defaultPageSize = 25
	maxPageSize     = 100
	maxLinkLen      = 2048
	maxNotesLen     = 10000
	maxTypeNameLen  = 50
)

var hexColour = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

type Store interface {
	ListTaskTypes(ctx context.Context, userID string) ([]dto.TaskType, error)
	CreateTaskType(ctx context.Context, userID string, in dto.TaskTypeInput) (dto.TaskType, error)
	UpdateTaskType(ctx context.Context, userID string, in dto.TaskTypeInput) (dto.TaskType, error)
	DeleteTaskType(ctx context.Context, userID, id string) error
	ListTasks(ctx context.Context, userID string, opts dto.TaskPageOptions) (dto.TaskPage, error)
	CreateTask(ctx context.Context, userID string, in dto.CreateTaskInput) (dto.Task, error)
	UpdateTask(ctx context.Context, userID string, in dto.UpdateTaskInput, now time.Time) (dto.Task, error)
	SetTaskDone(ctx context.Context, userID, id string, done bool, now time.Time) (dto.Task, error)
	DeleteTask(ctx context.Context, userID, id string) error
	SeedDefaultTaskTypes(ctx context.Context, tx pgx.Tx, userID string) error
	CreateApplyTask(ctx context.Context, tx pgx.Tx, userID, jobID string) error
	CompleteApplyTasks(ctx context.Context, tx pgx.Tx, userID, jobID string, now time.Time) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) CreateType(ctx context.Context, userID string, in dto.TaskTypeInput) (dto.TaskType, error) {
	in, err := validateTypeInput(in)
	if err != nil {
		return dto.TaskType{}, err
	}
	return mapTypeExists(s.store.CreateTaskType(ctx, userID, in))
}

func (s *Service) UpdateType(ctx context.Context, userID string, in dto.TaskTypeInput) (dto.TaskType, error) {
	in, err := validateTypeInput(in)
	if err != nil {
		return dto.TaskType{}, err
	}
	return mapTypeExists(s.store.UpdateTaskType(ctx, userID, in))
}

func (s *Service) DeleteType(ctx context.Context, userID, id string) error {
	err := s.store.DeleteTaskType(ctx, userID, id)
	if errors.Is(err, store.ErrSystemTaskType) {
		return apperr.Invalid("this task type is used by the app and can't be deleted")
	}
	return err
}

func mapTypeExists(t dto.TaskType, err error) (dto.TaskType, error) {
	if errors.Is(err, store.ErrTaskTypeExists) {
		return t, apperr.Conflict("a task type with that name already exists")
	}
	return t, err
}

func validateTypeInput(in dto.TaskTypeInput) (dto.TaskTypeInput, error) {
	in.Name = strings.TrimSpace(in.Name)
	if n := utf8.RuneCountInString(in.Name); n < 1 || n > maxTypeNameLen {
		return in, apperr.Invalid("name must be 1 to 50 characters")
	}
	if !hexColour.MatchString(in.Colour) {
		return in, apperr.Invalid("colour must be a hex colour like #6366f1")
	}
	return in, nil
}

func (s *Service) List(ctx context.Context, userID string, q dto.TasksQuery) (dto.TaskPage, error) {
	opts, err := parseTasksQuery(q)
	if err != nil {
		return dto.TaskPage{}, err
	}
	return s.store.ListTasks(ctx, userID, opts)
}

func (s *Service) Create(ctx context.Context, userID string, in dto.CreateTaskInput) (dto.Task, error) {
	if in.JobID != nil {
		if err := validUUID(*in.JobID, "job_id"); err != nil {
			return dto.Task{}, err
		}
	}
	link, err := validateTaskFields(in.TypeID, in.Link, in.Notes)
	if err != nil {
		return dto.Task{}, err
	}
	in.Link = link
	return s.store.CreateTask(ctx, userID, in)
}

func (s *Service) Update(ctx context.Context, userID string, in dto.UpdateTaskInput) (dto.Task, error) {
	link, err := validateTaskFields(in.TypeID, in.Link, in.Notes)
	if err != nil {
		return dto.Task{}, err
	}
	in.Link = link
	return s.store.UpdateTask(ctx, userID, in, time.Now())
}

func validateTaskFields(typeID, link, notes string) (string, error) {
	if err := validUUID(typeID, "type_id"); err != nil {
		return "", err
	}
	link = strings.TrimSpace(link)
	if err := validateLink(link); err != nil {
		return "", err
	}
	if utf8.RuneCountInString(notes) > maxNotesLen {
		return "", apperr.Invalid("notes are too long")
	}
	return link, nil
}

func (s *Service) SetDone(ctx context.Context, userID string, in dto.TaskDoneInput) (dto.Task, error) {
	return s.store.SetTaskDone(ctx, userID, in.ID, in.Done, time.Now())
}

func parseTasksQuery(q dto.TasksQuery) (dto.TaskPageOptions, error) {
	if q.TypeID != "" {
		if err := validUUID(q.TypeID, "type_id"); err != nil {
			return dto.TaskPageOptions{}, err
		}
	}
	if q.JobID != "" {
		if err := validUUID(q.JobID, "job_id"); err != nil {
			return dto.TaskPageOptions{}, err
		}
	}
	done, err := strconv.ParseBool(cmp.Or(q.Done, "false"))
	if err != nil {
		return dto.TaskPageOptions{}, apperr.Invalid("done must be true or false")
	}
	sort := dto.TaskSort(cmp.Or(q.Sort, string(dto.TaskSortCreated)))
	if sort != dto.TaskSortCreated && sort != dto.TaskSortJobPosted {
		return dto.TaskPageOptions{}, apperr.Invalid("sort must be created or job_posted")
	}
	page, err := parsePositiveInt(q.Page, 1, "page")
	if err != nil {
		return dto.TaskPageOptions{}, err
	}
	size, err := parsePositiveInt(q.PageSize, defaultPageSize, "page_size")
	if err != nil {
		return dto.TaskPageOptions{}, err
	}
	size = min(size, maxPageSize)
	if page-1 > math.MaxInt32/size {
		return dto.TaskPageOptions{}, apperr.Invalid("page is too large")
	}
	return dto.TaskPageOptions{
		TypeID: q.TypeID, JobID: q.JobID, Done: done, Sort: sort,
		Limit: size, Offset: (page - 1) * size,
	}, nil
}

func parsePositiveInt(raw string, fallback int32, name string) (int32, error) {
	if raw == "" {
		return fallback, nil
	}
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || n < 1 {
		return 0, apperr.Invalid(name + " must be a positive integer")
	}
	return int32(n), nil
}

func validUUID(s, name string) error {
	if _, err := uuid.Parse(s); err != nil {
		return apperr.Invalid(name + " must be a valid id")
	}
	return nil
}

func validateLink(link string) error {
	if link == "" {
		return nil
	}
	if len(link) > maxLinkLen {
		return apperr.Invalid("link is too long")
	}
	u, err := url.Parse(link)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return apperr.Invalid("link must be an http or https URL")
	}
	return nil
}

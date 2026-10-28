package applicationstest

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/applications"
)

var defaultStatuses = []dto.ApplicationStatus{
	{Name: "Draft", SystemKey: "draft"},
	{Name: "Applied", SystemKey: "applied"},
	{Name: "Interviewing"},
	{Name: "Rejected"},
	{Name: "Offer"},
}

type FakeStore struct {
	mu       sync.Mutex
	apps     map[string]dto.Application
	statuses map[string]dto.ApplicationStatus
	contacts map[string][]dto.RecruiterContact
	notes    map[noteKey]string
	comments map[string][]dto.JobComment

	nextContact int
	nextComment int
}

type noteKey struct{ userID, jobID string }

func NewFakeStore() *FakeStore {
	return &FakeStore{
		apps:     make(map[string]dto.Application),
		statuses: make(map[string]dto.ApplicationStatus),
		contacts: make(map[string][]dto.RecruiterContact),
		notes:    make(map[noteKey]string),
		comments: make(map[string][]dto.JobComment),
	}
}

func (f *FakeStore) ListDue(context.Context, string) (dto.DueList, error) {
	return dto.DueList{Jobs: []dto.DueJob{}}, nil
}

func (f *FakeStore) CreateApplication(_ context.Context, userID string, in dto.CreateApplicationInput) (dto.Application, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.ownsStatus(userID, in.StatusID) {
		return dto.Application{}, data.ErrNotFound
	}
	statusID := in.StatusID
	if statusID == "" {
		for _, s := range f.statuses {
			if s.UserID == userID && s.SystemKey == "draft" {
				statusID = s.ID
			}
		}
	}
	app := dto.Application{
		ID:         fmt.Sprintf("app-%d", len(f.apps)+1),
		UserID:     userID,
		JobID:      in.JobID,
		StatusID:   statusID,
		AppliedAt:  parseDate(in.AppliedAt),
		SalaryInfo: in.SalaryInfo,
		ChaseBy:    f.chaseFor(statusID, time.Now()),
	}
	f.apps[app.ID] = app
	return app, nil
}

func (f *FakeStore) ListApplications(_ context.Context, userID string, q dto.ApplicationsQuery) ([]dto.ApplicationWithDetails, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []dto.ApplicationWithDetails{}
	for _, a := range f.apps {
		if a.UserID != userID || (q.StatusID != "" && a.StatusID != q.StatusID) || (q.Chase && a.ChaseBy == nil) {
			continue
		}
		out = append(out, toDetails(a))
	}
	if q.Chase {
		slices.SortFunc(out, func(a, b dto.ApplicationWithDetails) int { return a.ChaseBy.Compare(*b.ChaseBy) })
	}
	return out, nil
}

func (f *FakeStore) SetChase(_ context.Context, userID, id string, chaseBy time.Time) (dto.Application, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	app, ok := f.apps[id]
	if !ok || app.UserID != userID {
		return dto.Application{}, data.ErrNotFound
	}
	app.ChaseBy = &chaseBy
	f.apps[id] = app
	return app, nil
}

func (f *FakeStore) ClearChase(_ context.Context, userID, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if app, ok := f.apps[id]; ok && app.UserID == userID {
		app.ChaseBy = nil
		f.apps[id] = app
	}
	return nil
}

func (f *FakeStore) ownsStatus(userID, statusID string) bool {
	if statusID == "" {
		return true
	}
	s, ok := f.statuses[statusID]
	return ok && s.UserID == userID
}

func (f *FakeStore) chaseFor(statusID string, from time.Time) *time.Time {
	s, ok := f.statuses[statusID]
	if !ok || s.ReplyWindowDays == nil {
		return nil
	}
	y, m, d := from.UTC().Date()
	chaseBy := time.Date(y, m, d+*s.ReplyWindowDays, 0, 0, 0, 0, time.UTC)
	return &chaseBy
}

func parseDate(s *string) *time.Time {
	if s == nil {
		return nil
	}
	d, err := time.Parse(time.DateOnly, *s)
	if err != nil {
		return nil
	}
	return &d
}

func toDetails(a dto.Application) dto.ApplicationWithDetails {
	return dto.ApplicationWithDetails{
		ID: a.ID, UserID: a.UserID, JobID: a.JobID, StatusID: a.StatusID,
		AppliedAt: a.AppliedAt, SalaryInfo: a.SalaryInfo, ChaseBy: a.ChaseBy,
	}
}

func (f *FakeStore) UpdateApplication(_ context.Context, userID, id string, in dto.UpdateApplicationInput) (dto.Application, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	app, ok := f.apps[id]
	if !ok || app.UserID != userID {
		return dto.Application{}, data.ErrNotFound
	}
	if !f.ownsStatus(userID, in.StatusID) {
		return dto.Application{}, data.ErrNotFound
	}
	if app.StatusID != in.StatusID {
		app.ChaseBy = f.chaseFor(in.StatusID, time.Now())
	}
	app.StatusID = in.StatusID
	app.AppliedAt = parseDate(in.AppliedAt)
	app.SalaryInfo = in.SalaryInfo
	f.apps[id] = app
	return app, nil
}

func (f *FakeStore) MarkApplied(_ context.Context, _ pgx.Tx, userID, jobID string, now time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	var appliedID string
	for _, s := range f.statuses {
		if s.UserID == userID && s.SystemKey == "applied" {
			appliedID = s.ID
		}
	}
	for id, a := range f.apps {
		if a.UserID != userID || a.JobID != jobID || a.StatusID == appliedID {
			continue
		}
		a.StatusID, a.ChaseBy = appliedID, f.chaseFor(appliedID, now)
		if a.AppliedAt == nil {
			a.AppliedAt = &now
		}
		f.apps[id] = a
	}
	return nil
}

// DeleteApplication is a no-op for an unknown id, matching the real store's
// unconditional DELETE.
func (f *FakeStore) DeleteApplication(_ context.Context, _, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.apps, id)
	return nil
}

func (f *FakeStore) DeleteApplications(_ context.Context, userID string, ids []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, id := range ids {
		if f.apps[id].UserID == userID {
			delete(f.apps, id)
		}
	}
	return nil
}

func (f *FakeStore) GetApplicationsForJobs(_ context.Context, userID string, jobIDs []string) (map[string]dto.JobApplicationSummary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	wanted := make(map[string]bool, len(jobIDs))
	for _, id := range jobIDs {
		wanted[id] = true
	}
	out := map[string]dto.JobApplicationSummary{}
	for _, a := range f.apps {
		if a.UserID == userID && wanted[a.JobID] {
			out[a.JobID] = dto.JobApplicationSummary{ApplicationID: a.ID, StatusID: a.StatusID}
		}
	}
	return out, nil
}

func (f *FakeStore) CreateApplicationStatus(_ context.Context, userID, name, colour string, replyWindowDays *int, countsAsReply bool) (dto.ApplicationStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := dto.ApplicationStatus{ID: fmt.Sprintf("status-%d", len(f.statuses)+1), UserID: userID, Name: name, Colour: colour, ReplyWindowDays: replyWindowDays, CountsAsReply: countsAsReply}
	f.statuses[s.ID] = s
	return s, nil
}

func (f *FakeStore) UpdateApplicationStatus(_ context.Context, id, userID, name, colour string, replyWindowDays *int, countsAsReply bool) (dto.ApplicationStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.statuses[id]
	if !ok || s.UserID != userID {
		return dto.ApplicationStatus{}, apperr.NotFound("status not found")
	}
	s.Name, s.Colour, s.ReplyWindowDays, s.CountsAsReply = name, colour, replyWindowDays, countsAsReply
	f.statuses[id] = s
	return s, nil
}

// DeleteApplicationStatus is a no-op for an unknown id, matching the real
// store's unconditional DELETE.
func (f *FakeStore) DeleteApplicationStatus(_ context.Context, id, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.statuses, id)
	return nil
}

func (f *FakeStore) CountApplicationsUsingStatus(_ context.Context, statusID, userID string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var n int64
	for _, a := range f.apps {
		if a.StatusID == statusID && a.UserID == userID {
			n++
		}
	}
	return n, nil
}

func (f *FakeStore) ListApplicationStatusesByUser(_ context.Context, userID string) ([]dto.ApplicationStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []dto.ApplicationStatus{}
	for _, s := range f.statuses {
		if s.UserID == userID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (f *FakeStore) SeedDefaultStatuses(_ context.Context, _ pgx.Tx, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range defaultStatuses {
		s.ID, s.UserID = fmt.Sprintf("status-%d", len(f.statuses)+1), userID
		f.statuses[s.ID] = s
	}
	return nil
}

var _ applications.Store = (*FakeStore)(nil)

func (f *FakeStore) CreateRecruiterContact(_ context.Context, userID string, in dto.LogRecruiterContactInput) (dto.RecruiterContact, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextContact++
	c := dto.RecruiterContact{
		ID: fmt.Sprintf("contact-%d", f.nextContact), JobID: in.JobID, ContactedOn: in.ContactedOn,
		Channel: in.Channel, Outcome: in.Outcome, CreatedAt: time.Now(),
	}
	f.contacts[userID] = append(f.contacts[userID], c)
	return c, nil
}

func (f *FakeStore) ListRecruiterContacts(_ context.Context, userID, jobID string) ([]dto.RecruiterContact, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []dto.RecruiterContact{}
	for _, c := range f.contacts[userID] {
		if c.JobID == jobID {
			out = append(out, c)
		}
	}
	slices.SortStableFunc(out, func(a, b dto.RecruiterContact) int {
		return cmp.Or(strings.Compare(b.ContactedOn, a.ContactedOn), b.CreatedAt.Compare(a.CreatedAt))
	})
	return out, nil
}

func (f *FakeStore) GetJobNote(_ context.Context, userID, jobID string) (dto.JobNote, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return dto.JobNote{JobID: jobID, Body: f.notes[noteKey{userID, jobID}]}, nil
}

func (f *FakeStore) PutJobNote(_ context.Context, userID string, in dto.PutJobNoteInput) (dto.JobNote, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.notes[noteKey{userID, in.JobID}] = in.Body
	return dto.JobNote(in), nil
}

func (f *FakeStore) CreateJobComment(_ context.Context, userID string, in dto.CreateJobCommentInput) (dto.JobComment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now()
	f.nextComment++
	c := dto.JobComment{ID: fmt.Sprintf("comment-%d", f.nextComment), JobID: in.JobID, Body: in.Body, Quote: in.Quote, CreatedAt: now, UpdatedAt: now}
	f.comments[userID] = append(f.comments[userID], c)
	return c, nil
}

func (f *FakeStore) ListJobComments(_ context.Context, userID, jobID string) ([]dto.JobComment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []dto.JobComment{}
	for _, c := range f.comments[userID] {
		if c.JobID == jobID {
			out = append(out, c)
		}
	}
	slices.Reverse(out)
	return out, nil
}

func (f *FakeStore) UpdateJobComment(_ context.Context, userID string, in dto.UpdateJobCommentInput) (dto.JobComment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, c := range f.comments[userID] {
		if c.ID == in.ID {
			c.Body, c.UpdatedAt = in.Body, time.Now()
			f.comments[userID][i] = c
			return c, nil
		}
	}
	return dto.JobComment{}, data.ErrNotFound
}

func (f *FakeStore) DeleteJobComment(_ context.Context, userID, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	before := len(f.comments[userID])
	f.comments[userID] = slices.DeleteFunc(f.comments[userID], func(c dto.JobComment) bool { return c.ID == id })
	if len(f.comments[userID]) == before {
		return data.ErrNotFound
	}
	return nil
}

var outcomeRank = map[dto.ContactOutcome]int{
	dto.OutcomeNoReply: 1, dto.OutcomeRejected: 2, dto.OutcomeSpoke: 3, dto.OutcomeSubmitted: 4,
}

func (f *FakeStore) GetRecruiterContactSummaries(_ context.Context, userID string, jobIDs []string) (map[string]dto.RecruiterContactSummary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string]dto.RecruiterContactSummary{}
	for _, c := range f.contacts[userID] {
		if !slices.Contains(jobIDs, c.JobID) {
			continue
		}
		sum := out[c.JobID]
		sum.Count++
		sum.LastContactedOn = max(sum.LastContactedOn, c.ContactedOn)
		if outcomeRank[c.Outcome] > outcomeRank[sum.BestOutcome] {
			sum.BestOutcome = c.Outcome
		}
		out[c.JobID] = sum
	}
	return out, nil
}

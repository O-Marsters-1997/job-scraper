package cvtailortest

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

var (
	errDraftNotFound = apperr.NotFound("draft not found")
	errKeptExists    = apperr.Conflict("this job already has a kept draft")
)

const statusKeeping = "keeping"

type draft struct {
	dto.Draft
	userID   string
	input    dto.DraftInput
	attempts int
	dueAt    time.Time
	leased   bool
	result   dto.DraftResult
}

// SetJob gives the Job the facts a claim carries; CompanySlug stands in for
// the company name.
func (f *FakeStore) SetJob(job dto.Job) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.jobs[job.ID] = job
}

// MakeDraftDue skips the retry backoff so the next tick claims the Draft.
func (f *FakeStore) MakeDraftDue(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.drafts[id].dueAt = time.Now()
}

// DraftResult returns what CompleteDraft recorded for the Draft.
func (f *FakeStore) DraftResult(id string) dto.DraftResult {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.drafts[id].result
}

func (f *FakeStore) CreateDraft(_ context.Context, userID string, in dto.DraftInput) (dto.Draft, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d := &draft{Draft: dto.Draft{ID: f.nextID(), JobID: in.JobID, Status: "pending", Findings: []dto.DraftFinding{}, CreatedAt: time.Now()}, userID: userID, input: in, dueAt: time.Now()}
	f.drafts[d.ID] = d
	return d.Draft, nil
}

func (f *FakeStore) GetDraft(_ context.Context, userID, id string) (dto.Draft, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.drafts[id]
	if !ok || d.userID != userID {
		return dto.Draft{}, errDraftNotFound
	}
	return d.Draft, nil
}

func (f *FakeStore) ListJobDrafts(_ context.Context, userID, jobID string) ([]dto.Draft, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var owned []*draft
	for _, d := range f.drafts {
		if d.userID == userID && d.JobID == jobID {
			owned = append(owned, d)
		}
	}
	slices.SortFunc(owned, func(a, b *draft) int {
		if c := b.CreatedAt.Compare(a.CreatedAt); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})
	out := make([]dto.Draft, len(owned))
	for i, d := range owned {
		out[i] = d.Draft
	}
	return out, nil
}

func (f *FakeStore) SetDraftOutcome(_ context.Context, userID, id, outcome string) (dto.Draft, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.drafts[id]
	if !ok || d.userID != userID {
		return dto.Draft{}, errDraftNotFound
	}
	if outcome == dto.OutcomeKept {
		for _, other := range f.drafts {
			if other != d && other.userID == userID && other.JobID == d.JobID && other.Outcome != nil && *other.Outcome == dto.OutcomeKept {
				return dto.Draft{}, errKeptExists
			}
		}
	}
	d.Outcome = &outcome
	if outcome == dto.OutcomeDiscarded {
		d.DraftDocID = ""
	}
	return d.Draft, nil
}

func (f *FakeStore) ClaimDraft(_ context.Context) (dto.DraftClaim, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var due []*draft
	for _, d := range f.drafts {
		isDue := !d.dueAt.After(time.Now())
		if (d.Status == "pending" && isDue) || (d.Status == statusKeeping && isDue && !d.leased) {
			due = append(due, d)
		}
	}
	if len(due) == 0 {
		return dto.DraftClaim{}, data.ErrNotFound
	}
	slices.SortFunc(due, func(a, b *draft) int { return a.dueAt.Compare(b.dueAt) })
	d := due[0]
	keeping := d.Status == statusKeeping
	if keeping {
		d.leased = true
	} else {
		d.Status = "running"
	}
	d.attempts++
	job := f.jobs[d.input.JobID]
	return dto.DraftClaim{
		ID: d.ID, UserID: d.userID, JobID: d.input.JobID, DocID: d.input.DocID, TabID: d.input.TabID,
		AchievementIDs: slices.Clone(d.input.AchievementIDs), Attempts: d.attempts, DraftDocID: d.DraftDocID,
		JobDescription: job.Description, JobFingerprint: job.ContentFingerprint,
		Keeping: keeping, JobTitle: job.Title, CompanyName: job.CompanySlug,
	}, nil
}

func (f *FakeStore) running(claim dto.DraftClaim) (*draft, error) {
	d, ok := f.drafts[claim.ID]
	if !ok || d.Status != "running" || d.attempts != claim.Attempts {
		return nil, errDraftNotFound
	}
	return d, nil
}

func (f *FakeStore) SetDraftDoc(_ context.Context, claim dto.DraftClaim, docID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, err := f.running(claim)
	if err != nil {
		return err
	}
	d.DraftDocID = docID
	return nil
}

func (f *FakeStore) CompleteDraft(_ context.Context, claim dto.DraftClaim, res dto.DraftResult) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, err := f.running(claim)
	if err != nil {
		return err
	}
	d.Status, d.LastError, d.DraftDocID, d.Findings, d.result, d.EditSet = "ready", "", res.DraftDocID, slices.Clone(res.Findings), res, res.EditSet
	return nil
}

func (f *FakeStore) FailDraft(_ context.Context, claim dto.DraftClaim, failure dto.DraftFailure) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, err := f.running(claim)
	if err != nil {
		return err
	}
	d.LastError = failure.Reason
	if failure.ClearDoc {
		d.DraftDocID = ""
	}
	if failure.Terminal || d.attempts >= dto.MaxDraftAttempts {
		d.Status = "failed"
		return nil
	}
	d.Status = "pending"
	d.dueAt = time.Now().Add(30 * time.Second << d.attempts)
	return nil
}

func (f *FakeStore) QueueKeep(_ context.Context, userID, id string) (dto.Draft, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.drafts[id]
	if !ok || d.userID != userID || d.Status != "ready" || d.Outcome != nil {
		return dto.Draft{}, errDraftNotFound
	}
	d.Status, d.attempts, d.dueAt, d.leased, d.LastError = statusKeeping, 0, time.Now(), false, ""
	return d.Draft, nil
}

func (f *FakeStore) keeping(claim dto.DraftClaim) (*draft, error) {
	d, ok := f.drafts[claim.ID]
	if !ok || d.Status != statusKeeping || d.attempts != claim.Attempts {
		return nil, errDraftNotFound
	}
	return d, nil
}

func (f *FakeStore) CompleteKeep(_ context.Context, claim dto.DraftClaim) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, err := f.keeping(claim)
	if err != nil {
		return err
	}
	for _, other := range f.drafts {
		if other != d && other.userID == d.userID && other.JobID == d.JobID && other.Outcome != nil && *other.Outcome == dto.OutcomeKept {
			return errKeptExists
		}
	}
	kept := dto.OutcomeKept
	d.Status, d.Outcome, d.KeptAs, d.leased, d.LastError = "ready", &kept, "doc", false, ""
	return nil
}

func (f *FakeStore) FailKeep(_ context.Context, claim dto.DraftClaim, failure dto.DraftFailure) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, err := f.keeping(claim)
	if err != nil {
		return err
	}
	d.LastError, d.leased = failure.Reason, false
	d.dueAt = time.Now().Add(30 * time.Second << d.attempts)
	if failure.Terminal || d.attempts >= dto.MaxDraftAttempts {
		d.Status = "ready"
	}
	return nil
}

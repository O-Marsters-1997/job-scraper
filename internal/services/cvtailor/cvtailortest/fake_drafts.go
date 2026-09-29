package cvtailortest

import (
	"context"
	"slices"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

var errDraftNotFound = apperr.NotFound("draft not found")

type draft struct {
	dto.Draft
	userID   string
	input    dto.DraftInput
	attempts int
	dueAt    time.Time
	result   dto.DraftResult
}

// SetJob gives the Job the description and fingerprint a claim carries.
func (f *FakeStore) SetJob(jobID, description, fingerprint string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.jobs[jobID] = dto.Job{ID: jobID, Description: description, ContentFingerprint: fingerprint}
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
	d := &draft{Draft: dto.Draft{ID: f.nextID(), JobID: in.JobID, Status: "pending", Findings: []dto.DraftFinding{}}, userID: userID, input: in, dueAt: time.Now()}
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

func (f *FakeStore) ClaimDraft(_ context.Context) (dto.DraftClaim, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var due []*draft
	for _, d := range f.drafts {
		if d.Status == "pending" && !d.dueAt.After(time.Now()) {
			due = append(due, d)
		}
	}
	if len(due) == 0 {
		return dto.DraftClaim{}, data.ErrNotFound
	}
	slices.SortFunc(due, func(a, b *draft) int { return a.dueAt.Compare(b.dueAt) })
	d := due[0]
	d.Status = "running"
	d.attempts++
	job := f.jobs[d.input.JobID]
	return dto.DraftClaim{
		ID: d.ID, UserID: d.userID, JobID: d.input.JobID, DocID: d.input.DocID, TabID: d.input.TabID,
		AchievementIDs: slices.Clone(d.input.AchievementIDs), Attempts: d.attempts, DraftDocID: d.DraftDocID,
		JobDescription: job.Description, JobFingerprint: job.ContentFingerprint,
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
	d.Status, d.LastError, d.DraftDocID, d.Findings, d.result = "ready", "", res.DraftDocID, slices.Clone(res.Findings), res
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

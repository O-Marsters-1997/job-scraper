package cvtailortest

import (
	"context"
	"slices"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

var (
	errJobNotFound         = apperr.NotFound("job not found")
	errCoverLetterNotFound = apperr.NotFound("cover letter not found")
)

type coverLetter struct {
	dto.CoverLetter
	userID        string
	scaffoldTabID string
}

type letterTab struct{ userID, docID, tabID string }

func (f *FakeStore) TrackLetterTab(userID, docID, tabID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.letterTabs[letterTab{userID, docID, tabID}] = true
	f.letterTabOrder = append(f.letterTabOrder, letterTab{userID, docID, tabID})
}

func (f *FakeStore) DefaultLetterTab(_ context.Context, userID string) (string, string, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, lt := range f.letterTabOrder {
		if lt.userID == userID {
			return lt.docID, lt.tabID, true, nil
		}
	}
	return "", "", false, nil
}

func (f *FakeStore) IsLetterTab(_ context.Context, userID, docID, tabID string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.letterTabs[letterTab{userID, docID, tabID}], nil
}

func (f *FakeStore) JobBrief(_ context.Context, jobID string) (dto.JobBrief, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	job, ok := f.jobs[jobID]
	if !ok {
		return dto.JobBrief{}, errJobNotFound
	}
	return dto.JobBrief{Title: job.Title, Company: job.CompanySlug, Description: job.Description}, nil
}

func (f *FakeStore) CreateCoverLetter(_ context.Context, userID string, in dto.NewCoverLetter) (dto.CoverLetter, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.jobs[in.JobID]; !ok {
		return dto.CoverLetter{}, errJobNotFound
	}
	cl := dto.CoverLetter{
		ID: f.nextID(), JobID: in.JobID, SourceDocID: in.SourceDocID, SourceTabID: in.SourceTabID,
		TabURL:         "https://docs.google.com/document/d/" + in.SourceDocID + "/edit?tab=" + in.ScaffoldTabID,
		AchievementIDs: slices.Clone(in.AchievementIDs), Cost: in.Cost, CreatedAt: time.Now(),
	}
	f.coverLetters = append(f.coverLetters, coverLetter{CoverLetter: cl, userID: userID, scaffoldTabID: in.ScaffoldTabID})
	return cl, nil
}

func (f *FakeStore) GetCoverLetter(_ context.Context, userID, id string) (dto.CoverLetterTab, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, cl := range f.coverLetters {
		if cl.userID == userID && cl.ID == id {
			return dto.CoverLetterTab{CoverLetter: cl.CoverLetter, ScaffoldTabID: cl.scaffoldTabID}, nil
		}
	}
	return dto.CoverLetterTab{}, errCoverLetterNotFound
}

func (f *FakeStore) MarkCoverLetterDone(_ context.Context, userID, id string) (dto.CoverLetter, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, cl := range f.coverLetters {
		if cl.userID != userID || cl.ID != id {
			continue
		}
		if cl.DoneAt == nil {
			now := time.Now()
			f.coverLetters[i].DoneAt = &now
		}
		return f.coverLetters[i].CoverLetter, nil
	}
	return dto.CoverLetter{}, errCoverLetterNotFound
}

func (f *FakeStore) ListJobCoverLetters(_ context.Context, userID, jobID string) ([]dto.CoverLetter, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []dto.CoverLetter{}
	for _, cl := range slices.Backward(f.coverLetters) {
		if cl.userID == userID && cl.JobID == jobID {
			out = append(out, cl.CoverLetter)
		}
	}
	return out, nil
}

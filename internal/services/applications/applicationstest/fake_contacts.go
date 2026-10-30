package applicationstest

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/applications/store"
)

type fakeContact struct {
	userID string
	dto.Contact
}

func dayString(t time.Time) string { return t.UTC().Format(time.DateOnly) }

func (f *FakeStore) addEvent(c *fakeContact, e dto.ContactEvent) {
	e.ID = fmt.Sprintf("event-%d", len(c.History)+1)
	e.JobID = c.JobID
	c.History = append(c.History, e)
	c.SentSinceReply = 0
	for i := len(c.History) - 1; i >= 0 && c.History[i].Kind != dto.ContactEventReply; i-- {
		if c.History[i].Kind == dto.ContactEventSent {
			c.SentSinceReply++
		}
	}
}

func (f *FakeStore) viewContact(c fakeContact) dto.Contact {
	out := c.Contact
	out.History = slices.Clone(c.History)
	out.ApplicationID = nil
	if c.JobID != nil {
		for _, a := range f.apps {
			if a.UserID == c.userID && a.JobID == *c.JobID {
				out.ApplicationID = &a.ID
			}
		}
	}
	return out
}

func (f *FakeStore) CreateContact(_ context.Context, userID string, in dto.ContactInput, today time.Time) (dto.Contact, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextOutreach++
	c := fakeContact{userID: userID, Contact: dto.Contact{
		ID: fmt.Sprintf("contact-%d", f.nextOutreach), Name: in.Name, Role: in.Role,
		CompanyID: in.CompanyID, CompanyName: in.CompanyName, JobID: in.JobID,
		LinkedInURL: in.LinkedInURL, Email: in.Email, Stage: dto.ContactStageToContact,
		History: []dto.ContactEvent{}, CreatedAt: time.Now(),
	}}
	f.addEvent(&c, dto.ContactEvent{Kind: dto.ContactEventAdded, OccurredOn: dayString(today)})
	f.outreach[c.ID] = c
	return f.viewContact(c), nil
}

func (f *FakeStore) ListContacts(_ context.Context, userID string) ([]dto.Contact, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []dto.Contact{}
	for _, c := range f.outreach {
		if c.userID == userID {
			out = append(out, f.viewContact(c))
		}
	}
	slices.SortFunc(out, func(a, b dto.Contact) int { return b.CreatedAt.Compare(a.CreatedAt) })
	return out, nil
}

func (f *FakeStore) DeleteContact(_ context.Context, userID, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if c, ok := f.outreach[id]; !ok || c.userID != userID {
		return data.ErrNotFound
	}
	delete(f.outreach, id)
	return nil
}

func (f *FakeStore) MarkContactSent(_ context.Context, userID string, in dto.MarkSentInput, today time.Time) (dto.ContactSent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.outreach[in.ID]
	if !ok || c.userID != userID {
		return dto.ContactSent{}, data.ErrNotFound
	}
	followUpBy, closes := store.NextSend(c.SentSinceReply, today)
	f.addEvent(&c, dto.ContactEvent{
		Kind: dto.ContactEventSent, OccurredOn: dayString(today), Channel: in.Channel,
		TemplateKind: in.TemplateKind, TemplateName: in.TemplateName,
	})
	c.Stage, c.Outcome, c.FollowUpBy = dto.ContactStageMessaged, "", nil
	if followUpBy != nil {
		day := dayString(*followUpBy)
		c.FollowUpBy = &day
	}
	if closes {
		c.Stage, c.Outcome = dto.ContactStageClosed, dto.ContactOutcomeNoReply
		f.addEvent(&c, dto.ContactEvent{Kind: dto.ContactEventClosed, OccurredOn: dayString(today), Outcome: dto.ContactOutcomeNoReply})
	}
	f.outreach[c.ID] = c

	moved := false
	if c.JobID != nil {
		for id, a := range f.apps {
			if a.UserID == userID && a.JobID == *c.JobID {
				a.ChaseBy, moved = followUpBy, true
				f.apps[id] = a
			}
		}
	}
	return dto.ContactSent{Contact: f.viewContact(c), MovedApplicationChase: moved}, nil
}

func (f *FakeStore) RecordContactReply(_ context.Context, userID string, in dto.RecordReplyInput, today time.Time) (dto.Contact, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.outreach[in.ID]
	if !ok || c.userID != userID {
		return dto.Contact{}, data.ErrNotFound
	}
	f.addEvent(&c, dto.ContactEvent{Kind: dto.ContactEventReply, OccurredOn: dayString(today), Outcome: in.Outcome})
	stage, followUpBy := store.ReplyState(in.Outcome, today)
	c.Stage, c.Outcome, c.FollowUpBy = stage, in.Outcome, nil
	if followUpBy != nil {
		day := dayString(*followUpBy)
		c.FollowUpBy = &day
	}
	if in.Reply != "" {
		c.Reply = in.Reply
	}
	f.outreach[c.ID] = c
	return f.viewContact(c), nil
}

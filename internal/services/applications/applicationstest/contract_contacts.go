package applicationstest

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func contactDay() time.Time { return time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC) }

func addContact(t *testing.T, f Fixture, in dto.ContactInput) dto.Contact {
	t.Helper()
	got, err := f.Store.CreateContact(t.Context(), f.UserID, in, contactDay())
	if err != nil {
		t.Fatalf("CreateContact(%+v) err = %v", in, err)
	}
	return got
}

func sendContact(t *testing.T, f Fixture, id string, day int) dto.ContactSent {
	t.Helper()
	in := dto.MarkSentInput{ID: id, Channel: dto.ContactChannelEmail, TemplateKind: dto.OutreachRecruiter, TemplateName: "Intro"}
	got, err := f.Store.MarkContactSent(t.Context(), f.UserID, in, contactDay().AddDate(0, 0, day))
	if err != nil {
		t.Fatalf("MarkContactSent(day %d) err = %v", day, err)
	}
	return got
}

func historyKinds(c dto.Contact) []dto.ContactEventKind {
	kinds := make([]dto.ContactEventKind, len(c.History))
	for i, e := range c.History {
		kinds[i] = e.Kind
	}
	return kinds
}

func followUp(c dto.Contact) string {
	if c.FollowUpBy == nil {
		return ""
	}
	return *c.FollowUpBy
}

func runContactContract(t *testing.T, newStore func(t *testing.T) Fixture) {
	t.Helper()

	t.Run("a new contact starts in to_contact with an added event", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		got := addContact(t, f, dto.ContactInput{Name: "Sam Recruiter", JobID: &f.JobID})
		if got.Stage != dto.ContactStageToContact || got.FollowUpBy != nil {
			t.Errorf("CreateContact(...) stage = %q follow-up = %v, want to_contact and none", got.Stage, got.FollowUpBy)
		}
		if diff := cmp.Diff([]dto.ContactEventKind{dto.ContactEventAdded}, historyKinds(got)); diff != "" {
			t.Errorf("CreateContact(...) history mismatch (-want +got):\n%s", diff)
		}
		if got.History[0].OccurredOn != "2026-10-01" {
			t.Errorf("added event occurred_on = %q, want 2026-10-01", got.History[0].OccurredOn)
		}
		list, err := f.Store.ListContacts(t.Context(), f.UserID)
		if err != nil || len(list) != 1 || list[0].ID != got.ID {
			t.Errorf("ListContacts(...) = %+v, %v, want the new contact", list, err)
		}
	})

	t.Run("a send marks the contact messaged and never sets a follow-up or closes it", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		c := addContact(t, f, dto.ContactInput{Name: "Sam"})
		var got dto.Contact
		for _, day := range []int{0, 5, 12} {
			got = sendContact(t, f, c.ID, day).Contact
		}
		if got.Stage != dto.ContactStageMessaged || got.FollowUpBy != nil || got.SentSinceReply != 3 {
			t.Errorf("third send = stage %q follow-up %v sent %d, want messaged, none, 3", got.Stage, got.FollowUpBy, got.SentSinceReply)
		}
		wantKinds := []dto.ContactEventKind{
			dto.ContactEventAdded, dto.ContactEventSent, dto.ContactEventSent, dto.ContactEventSent,
		}
		if diff := cmp.Diff(wantKinds, historyKinds(got)); diff != "" {
			t.Errorf("history after third send mismatch (-want +got):\n%s", diff)
		}
		if e := got.History[1]; e.Channel != dto.ContactChannelEmail || e.TemplateName != "Intro" || e.TemplateKind != dto.OutreachRecruiter {
			t.Errorf("sent event = %+v, want email, Intro, recruiter", e)
		}
	})

	t.Run("a reply restarts the cadence", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		c := addContact(t, f, dto.ContactInput{Name: "Sam"})
		sendContact(t, f, c.ID, 0)
		sendContact(t, f, c.ID, 5)
		in := dto.RecordReplyInput{ID: c.ID, Outcome: dto.ContactOutcomeNotNow, Reply: "ping me later"}
		if _, err := f.Store.RecordContactReply(t.Context(), f.UserID, in, contactDay().AddDate(0, 0, 6)); err != nil {
			t.Fatalf("RecordContactReply(...) err = %v", err)
		}
		got := sendContact(t, f, c.ID, 7).Contact
		if got.SentSinceReply != 1 || got.Stage != dto.ContactStageMessaged {
			t.Errorf("send after reply = sent %d stage %q, want 1, messaged", got.SentSinceReply, got.Stage)
		}
		if got.Reply != "ping me later" {
			t.Errorf("reply after a later send = %q, want it kept", got.Reply)
		}
	})

	t.Run("reply outcomes place the contact", func(t *testing.T) {
		t.Parallel()
		for _, tt := range []struct {
			outcome  dto.ContactOutcome
			stage    dto.ContactStage
			followUp string
		}{
			{dto.ContactOutcomeInterested, dto.ContactStageReplied, ""},
			{dto.ContactOutcomeReferred, dto.ContactStageReplied, ""},
			{dto.ContactOutcomeNotNow, dto.ContactStageReplied, "2027-01-01"},
			{dto.ContactOutcomeNoThanks, dto.ContactStageClosed, ""},
		} {
			t.Run(string(tt.outcome), func(t *testing.T) {
				t.Parallel()
				f := newStore(t)
				c := addContact(t, f, dto.ContactInput{Name: "Sam"})
				sendContact(t, f, c.ID, 0)
				in := dto.RecordReplyInput{ID: c.ID, Outcome: tt.outcome, Reply: "first"}
				got, err := f.Store.RecordContactReply(t.Context(), f.UserID, in, contactDay().AddDate(0, 0, 2))
				if err != nil {
					t.Fatalf("RecordContactReply(%s) err = %v", tt.outcome, err)
				}
				if got.Stage != tt.stage || followUp(got) != tt.followUp || got.Outcome != tt.outcome || got.Reply != "first" {
					t.Errorf("RecordContactReply(%s) = stage %q follow-up %q outcome %q reply %q", tt.outcome, got.Stage, followUp(got), got.Outcome, got.Reply)
				}
				last := got.History[len(got.History)-1]
				if last.Kind != dto.ContactEventReply || last.Outcome != tt.outcome {
					t.Errorf("last event = %+v, want a reply with outcome %s", last, tt.outcome)
				}
				in.Reply = ""
				kept, err := f.Store.RecordContactReply(t.Context(), f.UserID, in, contactDay().AddDate(0, 0, 3))
				if err != nil || kept.Reply != "first" {
					t.Errorf("RecordContactReply(no text) = %q, %v, want the earlier reply kept", kept.Reply, err)
				}
				in.Reply = "second"
				replaced, err := f.Store.RecordContactReply(t.Context(), f.UserID, in, contactDay().AddDate(0, 0, 4))
				if err != nil || replaced.Reply != "second" {
					t.Errorf("RecordContactReply(new text) = %q, %v, want it replaced", replaced.Reply, err)
				}
			})
		}
	})

	t.Run("a send leaves the linked application's chase alone", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		createApp(t, f, dto.CreateApplicationInput{})
		before, err := f.Store.ListApplications(t.Context(), f.UserID, dto.ApplicationsQuery{})
		if err != nil || len(before) != 1 {
			t.Fatalf("ListApplications(before) = %+v, %v, want one", before, err)
		}
		c := addContact(t, f, dto.ContactInput{Name: "Sam", JobID: &f.JobID})
		sendContact(t, f, c.ID, 0)
		after, err := f.Store.ListApplications(t.Context(), f.UserID, dto.ApplicationsQuery{})
		if err != nil || len(after) != 1 || !cmp.Equal(before[0].ChaseBy, after[0].ChaseBy) {
			t.Errorf("ListApplications(after send) = %+v, %v, want chase_by unchanged from %+v", after, err, before)
		}
	})

	t.Run("delete removes the contact and its history", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		c := addContact(t, f, dto.ContactInput{Name: "Sam"})
		if err := f.Store.DeleteContact(t.Context(), f.UserID, c.ID); err != nil {
			t.Fatalf("DeleteContact(...) err = %v", err)
		}
		if list, err := f.Store.ListContacts(t.Context(), f.UserID); err != nil || len(list) != 0 {
			t.Errorf("ListContacts(after delete) = %+v, %v, want none", list, err)
		}
	})

	t.Run("contacts are scoped to the user", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		c := addContact(t, f, dto.ContactInput{Name: "Sam"})
		if list, err := f.Store.ListContacts(t.Context(), f.OtherUserID); err != nil || len(list) != 0 {
			t.Errorf("ListContacts(other user) = %+v, %v, want none", list, err)
		}
		if err := f.Store.DeleteContact(t.Context(), f.OtherUserID, c.ID); !errors.Is(err, data.ErrNotFound) {
			t.Errorf("DeleteContact(other user) err = %v, want ErrNotFound", err)
		}
		in := dto.MarkSentInput{ID: c.ID, Channel: dto.ContactChannelEmail}
		if _, err := f.Store.MarkContactSent(t.Context(), f.OtherUserID, in, contactDay()); !errors.Is(err, data.ErrNotFound) {
			t.Errorf("MarkContactSent(other user) err = %v, want ErrNotFound", err)
		}
		reply := dto.RecordReplyInput{ID: c.ID, Outcome: dto.ContactOutcomeInterested}
		if _, err := f.Store.RecordContactReply(t.Context(), f.OtherUserID, reply, contactDay()); !errors.Is(err, data.ErrNotFound) {
			t.Errorf("RecordContactReply(other user) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("missing contact is not found", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		if err := f.Store.DeleteContact(t.Context(), f.UserID, missingID); !errors.Is(err, data.ErrNotFound) {
			t.Errorf("DeleteContact(missing) err = %v, want ErrNotFound", err)
		}
	})
}

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

	t.Run("sends book follow-ups at five then seven days and the third send closes", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		c := addContact(t, f, dto.ContactInput{Name: "Sam"})
		first := sendContact(t, f, c.ID, 0).Contact
		if first.Stage != dto.ContactStageMessaged || followUp(first) != "2026-10-06" {
			t.Errorf("first send = stage %q follow-up %q, want messaged 2026-10-06", first.Stage, followUp(first))
		}
		second := sendContact(t, f, c.ID, 5).Contact
		if followUp(second) != "2026-10-13" || second.SentSinceReply != 2 {
			t.Errorf("second send = follow-up %q sent %d, want 2026-10-13 and 2", followUp(second), second.SentSinceReply)
		}
		third := sendContact(t, f, c.ID, 12).Contact
		if third.Stage != dto.ContactStageClosed || third.Outcome != dto.ContactOutcomeNoReply || third.FollowUpBy != nil {
			t.Errorf("third send = %q %q %v, want closed no_reply and no follow-up", third.Stage, third.Outcome, third.FollowUpBy)
		}
		wantKinds := []dto.ContactEventKind{
			dto.ContactEventAdded, dto.ContactEventSent, dto.ContactEventSent, dto.ContactEventSent, dto.ContactEventClosed,
		}
		if diff := cmp.Diff(wantKinds, historyKinds(third)); diff != "" {
			t.Errorf("history after third send mismatch (-want +got):\n%s", diff)
		}
		if got := third.History[1]; got.Channel != dto.ContactChannelEmail || got.TemplateName != "Intro" || got.TemplateKind != dto.OutreachRecruiter {
			t.Errorf("sent event = %+v, want email, Intro, recruiter", got)
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
		if got.SentSinceReply != 1 || followUp(got) != "2026-10-13" || got.Stage != dto.ContactStageMessaged {
			t.Errorf("send after reply = sent %d follow-up %q stage %q, want 1, 2026-10-13, messaged", got.SentSinceReply, followUp(got), got.Stage)
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

	t.Run("a send moves the linked application's chase", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		app := createApp(t, f, dto.CreateApplicationInput{})
		c := addContact(t, f, dto.ContactInput{Name: "Sam", JobID: &f.JobID})
		if c.ApplicationID == nil || *c.ApplicationID != app.ID {
			t.Errorf("CreateContact(...) application = %v, want %s", c.ApplicationID, app.ID)
		}
		got := sendContact(t, f, c.ID, 0)
		if !got.MovedApplicationChase {
			t.Errorf("MarkContactSent(...) moved_application_chase = false, want true")
		}
		apps, err := f.Store.ListApplications(t.Context(), f.UserID, dto.ApplicationsQuery{})
		if err != nil || len(apps) != 1 || apps[0].ChaseBy == nil || apps[0].ChaseBy.Format(time.DateOnly) != "2026-10-06" {
			t.Errorf("ListApplications(...) = %+v, %v, want chase_by 2026-10-06", apps, err)
		}
	})

	t.Run("a send without a linked application moves nothing", func(t *testing.T) {
		t.Parallel()
		f := newStore(t)
		c := addContact(t, f, dto.ContactInput{Name: "Sam", JobID: &f.JobID})
		if got := sendContact(t, f, c.ID, 0); got.MovedApplicationChase {
			t.Errorf("MarkContactSent(no application) moved_application_chase = true, want false")
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

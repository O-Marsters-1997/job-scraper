package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/applications/store/sqlc"
)

func pgDate(t time.Time) pgtype.Date {
	y, m, d := t.UTC().Date()
	return pgtype.Date{Time: time.Date(y, m, d, 0, 0, 0, 0, time.UTC), Valid: true}
}

func optDate(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{}
	}
	return pgDate(*t)
}

func optUUID(s *string) (pgtype.UUID, error) {
	if s == nil || *s == "" {
		return pgtype.UUID{}, nil
	}
	return data.UUID(*s)
}

func (s *Store) CreateContact(ctx context.Context, userID string, in dto.ContactInput, today time.Time) (dto.Contact, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.Contact{}, err
	}
	companyID, err := optUUID(in.CompanyID)
	if err != nil {
		return dto.Contact{}, err
	}
	jobID, err := optUUID(in.JobID)
	if err != nil {
		return dto.Contact{}, err
	}
	var out dto.Contact
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.queries.WithTx(tx)
		id, err := q.CreateContact(ctx, sqlc.CreateContactParams{
			UserID: uid, Name: in.Name, Role: in.Role, CompanyID: companyID, CompanyName: in.CompanyName,
			JobID: jobID, LinkedinUrl: data.Text(in.LinkedInURL), Email: data.Text(in.Email),
		})
		if err != nil {
			return err
		}
		if err := q.InsertContactEvent(ctx, sqlc.InsertContactEventParams{
			ContactID: id, Kind: string(dto.ContactEventAdded), OccurredOn: pgDate(today),
		}); err != nil {
			return err
		}
		out, err = readContact(ctx, q, uid, id)
		return err
	})
	switch {
	case data.IsForeignKeyViolation(err, "contacts_job_id_fkey"):
		return dto.Contact{}, ErrContactJobNotFound
	case data.IsForeignKeyViolation(err, "contacts_company_id_fkey"):
		return dto.Contact{}, ErrContactCompanyNotFound
	case err != nil:
		return dto.Contact{}, data.QueryErr("CreateContact", err)
	}
	return out, nil
}

func (s *Store) ListContacts(ctx context.Context, userID string) ([]dto.Contact, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, err
	}
	out, err := readContacts(ctx, s.queries, uid, pgtype.UUID{})
	if err != nil {
		return nil, fmt.Errorf("store.ListContacts: %w", err)
	}
	return out, nil
}

func (s *Store) DeleteContact(ctx context.Context, userID, id string) error {
	uid, err := data.UUID(userID)
	if err != nil {
		return err
	}
	cid, err := data.UUID(id)
	if err != nil {
		return err
	}
	n, err := s.queries.DeleteContact(ctx, sqlc.DeleteContactParams{ID: cid, UserID: uid})
	if err != nil {
		return fmt.Errorf("store.DeleteContact: %w", err)
	}
	if n == 0 {
		return data.ErrNotFound
	}
	return nil
}

func (s *Store) MarkContactSent(ctx context.Context, userID string, in dto.MarkSentInput, today time.Time) (dto.ContactSent, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.ContactSent{}, err
	}
	cid, err := data.UUID(in.ID)
	if err != nil {
		return dto.ContactSent{}, err
	}
	var out dto.ContactSent
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.queries.WithTx(tx)
		if _, err := q.LockContact(ctx, sqlc.LockContactParams{ID: cid, UserID: uid}); err != nil {
			return err
		}
		sentBefore, err := q.CountSentSinceReply(ctx, cid)
		if err != nil {
			return err
		}
		if err := q.InsertContactEvent(ctx, sqlc.InsertContactEventParams{
			ContactID: cid, Kind: string(dto.ContactEventSent), OccurredOn: pgDate(today),
			Channel: data.Text(string(in.Channel)), TemplateKind: data.Text(string(in.TemplateKind)),
			TemplateName: in.TemplateName,
		}); err != nil {
			return err
		}
		followUpBy, closes := NextSend(int(sentBefore), today)
		state := sqlc.SetContactStateParams{ID: cid, UserID: uid, Stage: string(dto.ContactStageMessaged), FollowUpBy: optDate(followUpBy)}
		if closes {
			state.Stage = string(dto.ContactStageClosed)
			state.Outcome = data.Text(string(dto.ContactOutcomeNoReply))
			if err := q.InsertContactEvent(ctx, sqlc.InsertContactEventParams{
				ContactID: cid, Kind: string(dto.ContactEventClosed), OccurredOn: pgDate(today),
				Outcome: state.Outcome,
			}); err != nil {
				return err
			}
		}
		if err := q.SetContactState(ctx, state); err != nil {
			return err
		}
		moved, err := q.SetLinkedApplicationChase(ctx, sqlc.SetLinkedApplicationChaseParams{
			ChaseBy: state.FollowUpBy, ContactID: cid, UserID: uid,
		})
		if err != nil {
			return err
		}
		out.MovedApplicationChase = moved > 0
		out.Contact, err = readContact(ctx, q, uid, cid)
		return err
	})
	if err != nil {
		return dto.ContactSent{}, data.QueryErr("MarkContactSent", err)
	}
	return out, nil
}

func (s *Store) RecordContactReply(ctx context.Context, userID string, in dto.RecordReplyInput, today time.Time) (dto.Contact, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.Contact{}, err
	}
	cid, err := data.UUID(in.ID)
	if err != nil {
		return dto.Contact{}, err
	}
	var out dto.Contact
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.queries.WithTx(tx)
		if _, err := q.LockContact(ctx, sqlc.LockContactParams{ID: cid, UserID: uid}); err != nil {
			return err
		}
		outcome := data.Text(string(in.Outcome))
		if err := q.InsertContactEvent(ctx, sqlc.InsertContactEventParams{
			ContactID: cid, Kind: string(dto.ContactEventReply), OccurredOn: pgDate(today), Outcome: outcome,
		}); err != nil {
			return err
		}
		stage, followUpBy := ReplyState(in.Outcome, today)
		if err := q.SetContactState(ctx, sqlc.SetContactStateParams{
			ID: cid, UserID: uid, Stage: string(stage), Outcome: outcome,
			FollowUpBy: optDate(followUpBy), Reply: data.Text(in.Reply),
		}); err != nil {
			return err
		}
		out, err = readContact(ctx, q, uid, cid)
		return err
	})
	if err != nil {
		return dto.Contact{}, data.QueryErr("RecordContactReply", err)
	}
	return out, nil
}

func readContact(ctx context.Context, q *sqlc.Queries, userID, id pgtype.UUID) (dto.Contact, error) {
	cs, err := readContacts(ctx, q, userID, id)
	if err != nil {
		return dto.Contact{}, err
	}
	if len(cs) == 0 {
		return dto.Contact{}, data.ErrNotFound
	}
	return cs[0], nil
}

func readContacts(ctx context.Context, q *sqlc.Queries, userID, id pgtype.UUID) ([]dto.Contact, error) {
	rows, err := q.ListContacts(ctx, sqlc.ListContactsParams{UserID: userID, ID: id})
	if err != nil {
		return nil, err
	}
	ids := make([]pgtype.UUID, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	events, err := q.ListContactEvents(ctx, ids)
	if err != nil {
		return nil, err
	}
	return toContactDTOs(rows, events), nil
}

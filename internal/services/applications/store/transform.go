package store

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/applications/store/sqlc"
)

func fromOptionalDate(d pgtype.Date) *time.Time {
	if !d.Valid {
		return nil
	}
	t := d.Time
	return &t
}

func fromOptionalTimestamp(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}

func toApplicationDTO(a sqlc.Application) dto.Application {
	return dto.Application{
		ID:         a.ID.String(),
		UserID:     a.UserID.String(),
		JobID:      a.JobID.String(),
		StatusID:   a.StatusID.String(),
		AppliedAt:  fromOptionalDate(a.AppliedAt),
		SalaryInfo: a.SalaryInfo.String,
		CreatedAt:  a.CreatedAt.Time,
		UpdatedAt:  a.UpdatedAt.Time,
		ChaseBy:    fromOptionalDate(a.ChaseBy),
	}
}

func toLinkedDraft(r sqlc.ListApplicationsRow) *dto.LinkedDraft {
	if !r.DraftID.Valid {
		return nil
	}
	var outcome *string
	if r.DraftOutcome.Valid {
		outcome = &r.DraftOutcome.String
	}
	return &dto.LinkedDraft{
		ID:        r.DraftID.String(),
		Status:    r.DraftStatus,
		Outcome:   outcome,
		LastError: r.DraftLastError,
	}
}

func toApplicationWithDetailsDTO(r sqlc.ListApplicationsRow) dto.ApplicationWithDetails {
	return dto.ApplicationWithDetails{
		ID:             r.ID.String(),
		UserID:         r.UserID.String(),
		JobID:          r.JobID.String(),
		JobTitle:       r.JobTitle,
		JobCompanySlug: r.JobCompanySlug,
		JobLocation:    r.JobLocation,
		JobURL:         r.JobUrl,
		JobClosedAt:    fromOptionalTimestamp(r.JobClosedAt),
		StatusID:       r.StatusID.String(),
		StatusName:     r.StatusName.String,
		StatusColour:   r.StatusColour.String,
		AppliedAt:      fromOptionalDate(r.AppliedAt),
		SalaryInfo:     r.SalaryInfo.String,
		CreatedAt:      r.CreatedAt.Time,
		UpdatedAt:      r.UpdatedAt.Time,
		ChaseBy:        fromOptionalDate(r.ChaseBy),
		Draft:          toLinkedDraft(r),
	}
}

func toApplicationStatusDTO(s sqlc.ApplicationStatus) dto.ApplicationStatus {
	return dto.ApplicationStatus{
		ID:              s.ID.String(),
		UserID:          s.UserID.String(),
		Name:            s.Name,
		Colour:          s.Colour,
		ReplyWindowDays: fromInt4(s.ReplyWindowDays),
		CountsAsReply:   s.CountsAsReply,
		SystemKey:       s.SystemKey.String,
		CreatedAt:       s.CreatedAt.Time,
	}
}

func toJobApplicationSummaryDTO(r sqlc.GetApplicationsForJobsRow) dto.JobApplicationSummary {
	return dto.JobApplicationSummary{
		ApplicationID: r.ID.String(),
		StatusID:      r.StatusID.String(),
		StatusName:    r.StatusName.String,
		StatusColour:  r.StatusColour.String,
	}
}

func toInt4(n *int) pgtype.Int4 {
	if n == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(*n), Valid: true}
}

func fromInt4(n pgtype.Int4) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int32)
	return &v
}

func toJobNoteDTO(n sqlc.JobNote) dto.JobNote {
	return dto.JobNote{
		JobID: n.JobID.String(),
		Body:  n.Body,
	}
}

func toJobCommentDTO(c sqlc.JobComment) dto.JobComment {
	return dto.JobComment{
		ID:        c.ID.String(),
		JobID:     c.JobID.String(),
		Body:      c.Body,
		Quote:     dto.CommentQuote{Exact: c.Quote, Prefix: c.QuotePrefix, Suffix: c.QuoteSuffix},
		CreatedAt: c.CreatedAt.Time,
		UpdatedAt: c.UpdatedAt.Time,
	}
}

func optIDString(id pgtype.UUID) *string {
	if !id.Valid {
		return nil
	}
	s := id.String()
	return &s
}

func toContactDTOs(rows []sqlc.ListContactsRow, events []sqlc.ListContactEventsRow) []dto.Contact {
	history := make(map[pgtype.UUID][]dto.ContactEvent, len(rows))
	for _, e := range events {
		history[e.ContactID] = append(history[e.ContactID], dto.ContactEvent{
			ID:           e.ID.String(),
			Kind:         dto.ContactEventKind(e.Kind),
			OccurredOn:   e.OccurredOn.Time.Format(time.DateOnly),
			Channel:      dto.ContactChannel(e.Channel.String),
			TemplateKind: dto.OutreachKind(e.TemplateKind.String),
			TemplateName: e.TemplateName,
			Outcome:      dto.ContactOutcome(e.Outcome.String),
			Note:         e.Note,
			JobID:        optIDString(e.JobID),
			JobTitle:     e.JobTitle,
		})
	}
	out := make([]dto.Contact, len(rows))
	for i, r := range rows {
		c := dto.Contact{
			ID: r.ID.String(), Name: r.Name, Role: r.Role,
			CompanyID: optIDString(r.CompanyID), CompanyName: r.CompanyName, CompanyKind: r.CompanyKind,
			JobID: optIDString(r.JobID), JobTitle: r.JobTitle, ApplicationID: optIDString(r.ApplicationID),
			LinkedInURL: r.LinkedinUrl, Email: r.Email,
			Stage: dto.ContactStage(r.Stage), Outcome: dto.ContactOutcome(r.Outcome.String),
			Reply: r.Reply, SentSinceReply: int(r.SentSinceReply),
			History: history[r.ID], CreatedAt: r.CreatedAt.Time,
		}
		if c.History == nil {
			c.History = []dto.ContactEvent{}
		}
		if t := fromOptionalDate(r.FollowUpBy); t != nil {
			day := t.Format(time.DateOnly)
			c.FollowUpBy = &day
		}
		out[i] = c
	}
	return out
}

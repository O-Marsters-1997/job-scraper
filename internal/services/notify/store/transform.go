package store

import (
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/notify/store/sqlc"
)

func toNotificationDTO(r sqlc.ListNotificationsRow) dto.Notification {
	return dto.Notification{
		ID:    r.ID.String(),
		Kind:  r.Kind,
		JobID: r.JobID.String(),
		Title: r.Title,
		Body:  r.Body,
		Job: dto.NotificationJob{
			Title:       r.JobTitle,
			CompanyName: r.CompanyName,
			Location:    r.Location,
			SalaryRaw:   r.SalaryRaw,
			Band:        r.Band,
			Score:       intPtr(r.SuitabilityScore),
		},
		CreatedAt: r.CreatedAt.Time,
		ReadAt:    data.TimePtr(r.ReadAt),
	}
}

func toNotificationJobDTO(r sqlc.GetNotificationJobRow) dto.NotificationJob {
	return dto.NotificationJob{
		Title:       r.Title,
		CompanyName: r.CompanyName,
		Location:    r.Location,
		SalaryRaw:   r.SalaryRaw,
		Band:        r.Band,
		Score:       intPtr(r.SuitabilityScore),
	}
}

func intPtr(n pgtype.Int4) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int32)
	return &v
}

package store

import (
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates/store/sqlc"
)

func toTrackedDocDTO(t sqlc.TrackedDoc) dto.TrackedDoc {
	return dto.TrackedDoc{
		ID:      t.ID.String(),
		UserID:  t.UserID.String(),
		DocID:   t.DocID,
		AddedAt: t.AddedAt.Time,
	}
}

func toTabDTO(t sqlc.TrackedDocTab) dto.Tab {
	return dto.Tab{
		ID:           t.ID.String(),
		TrackedDocID: t.TrackedDocID.String(),
		TabID:        t.TabID,
		Title:        t.Title,
		Visible:      t.Visible,
		CreatedAt:    t.CreatedAt.Time,
	}
}

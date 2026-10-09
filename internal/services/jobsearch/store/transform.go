package store

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/runwindow"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store/sqlc"
)

func optionalInt32(v pgtype.Int4) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int32)
	return &n
}

func unmarshalBreakdown(raw []byte, out *[]dto.ScoreRow) error {
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("unmarshal job score breakdown: %w", err)
	}
	return nil
}

func toPageJobDTO(row sqlc.PageJobsRow) (dto.Job, error) {
	j := dto.Job{
		ID:                row.ID.String(),
		Title:             row.Title,
		Location:          row.Location,
		URL:               row.Url,
		CompanySlug:       row.CompanySlug,
		Source:            row.Source,
		UpdatedAt:         row.UpdatedAt.Time,
		ScrapedAt:         row.ScrapedAt.Time,
		FirstDiscoveredAt: row.FirstDiscoveredAt.Time,
		SalaryRaw:         row.SalaryRaw,
		WorkArrangement:   row.WorkArrangement,
	}
	if err := unmarshalBreakdown(row.Breakdown, &j.Breakdown); err != nil {
		return dto.Job{}, err
	}
	j.SuitabilityScore = optionalInt32(row.SuitabilityScore)
	j.Band = row.Band.String
	j.Grade = row.Grade
	j.Seen = row.Seen
	j.CompanyFavourite = row.CompanyFavourite
	j.CompanyID = row.CompanyID.String()
	j.BoardID = row.PrimaryBoardID.String()
	j.ProviderPostingID = row.ProviderPostingID.String
	j.ContentFingerprint = row.ContentFingerprint.String
	return j, nil
}

func toGetJobDTO(row sqlc.GetJobRow) (dto.Job, error) {
	j := dto.Job{
		ID:                row.ID.String(),
		Title:             row.Title,
		Location:          row.Location,
		URL:               row.Url,
		CompanySlug:       row.CompanySlug,
		Source:            row.Source,
		UpdatedAt:         row.UpdatedAt.Time,
		ScrapedAt:         row.ScrapedAt.Time,
		FirstDiscoveredAt: row.FirstDiscoveredAt.Time,
		Description:       row.Description,
		SalaryRaw:         row.SalaryRaw,
		WorkArrangement:   row.WorkArrangement,
	}
	if err := unmarshalBreakdown(row.Breakdown, &j.Breakdown); err != nil {
		return dto.Job{}, err
	}
	j.SuitabilityScore = optionalInt32(row.SuitabilityScore)
	j.Band = row.Band.String
	j.Seen = row.Seen
	j.CompanyFavourite = row.CompanyFavourite
	j.CompanyID = row.CompanyID.String()
	j.BoardID = row.PrimaryBoardID.String()
	j.ProviderPostingID = row.ProviderPostingID.String
	j.ContentFingerprint = row.ContentFingerprint.String
	return j, nil
}

func toCompanyDTO(row sqlc.Company) dto.Company {
	return dto.Company{
		ID:                row.ID.String(),
		Slug:              row.Slug,
		Name:              row.Name,
		ATSSource:         row.AtsSource.String,
		ATSToken:          row.AtsToken.String,
		Domain:            row.Domain.String,
		LinkedInCompanyID: row.LinkedinCompanyID.String,
		FirstSeenAt:       row.FirstSeenAt.Time,
		LastCrawledAt:     data.TimePtr(row.LastCrawledAt),
	}
}

func toCompanyForUserDTO(row sqlc.GetCompanyForUserRow) dto.Company {
	return dto.Company{
		ID:                   row.ID.String(),
		Slug:                 row.Slug,
		Name:                 row.Name,
		ATSSource:            row.AtsSource.String,
		ATSToken:             row.AtsToken.String,
		Domain:               row.Domain.String,
		LinkedInCompanyID:    row.LinkedinCompanyID.String,
		FirstSeenAt:          row.FirstSeenAt.Time,
		JobCount:             int(row.JobCount),
		Tracked:              row.Tracked,
		Favourite:            row.Favourite,
		ReviewState:          row.ReviewState,
		CheckIntervalMinutes: int(row.CheckIntervalMinutes.Int32),
		LastCheckedAt:        data.TimePtr(row.LastCheckedAt),
		LastCrawledAt:        data.TimePtr(row.LastCrawledAt),
	}
}

func toCompanyBoardDTO(row sqlc.CompanyBoard) dto.CompanyBoard {
	board := dto.CompanyBoard{
		ID:                 row.ID.String(),
		CompanyID:          row.CompanyID.String(),
		Source:             row.Source,
		BoardToken:         row.BoardToken,
		Status:             dto.BoardStatus(row.Status),
		VerificationMethod: row.VerificationMethod.String,
		DiscoveredVia:      row.DiscoveredVia.String,
		CreatedAt:          row.CreatedAt.Time,
	}
	board.VerifiedAt = data.TimePtr(row.VerifiedAt)
	board.LastLinkedAt = data.TimePtr(row.LastLinkedAt)
	board.RetiredAt = data.TimePtr(row.RetiredAt)
	return board
}

type windowColumns struct {
	interval  pgtype.Int4
	weekdays  pgtype.Int2
	start     pgtype.Time
	end       pgtype.Time
	timezone  pgtype.Text
	nextRunAt pgtype.Timestamptz
}

// windowArgs leaves the column arguments NULL for an unset part of the window
// so the insert falls back to the table defaults.
func windowArgs(w dto.RunWindow, nextRunAt *time.Time) windowColumns {
	cols := windowColumns{
		timezone:  data.Text(w.Timezone),
		start:     clockTime(w.Start),
		end:       clockTime(w.End),
		nextRunAt: timestamptz(nextRunAt),
	}
	if w.IntervalMinutes != nil {
		cols.interval = pgtype.Int4{Int32: int32(*w.IntervalMinutes), Valid: true}
	}
	if len(w.Weekdays) > 0 {
		cols.weekdays = pgtype.Int2{Int16: runwindow.WeekdayMask(w.Weekdays), Valid: true}
	}
	return cols
}

func timestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func clockTime(s string) pgtype.Time {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return pgtype.Time{}
	}
	micros := time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute
	return pgtype.Time{Microseconds: micros.Microseconds(), Valid: true}
}

func clockString(t pgtype.Time) string {
	d := time.Duration(t.Microseconds) * time.Microsecond
	return fmt.Sprintf("%02d:%02d", int(d.Hours()), int(d.Minutes())%60)
}

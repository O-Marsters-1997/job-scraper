package store

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/store/sqlc"
)

func fromOptionalTimestamptz(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}

func toScoringOptionDTO(row sqlc.ScoringOption) dto.ScoringOption {
	return dto.ScoringOption{
		ID:        row.ID,
		Dimension: dto.Dimension(row.Dimension),
		Label:     row.Label,
		Question:  row.Question,
		RetiredAt: fromOptionalTimestamptz(row.RetiredAt),
	}
}

func toSearchConfigDTO(row sqlc.SearchConfig) (dto.SearchConfig, error) {
	var prefs dto.Preferences
	if len(row.Preferences) > 0 {
		if err := json.Unmarshal(row.Preferences, &prefs); err != nil {
			return dto.SearchConfig{}, err
		}
	}
	return dto.SearchConfig{
		ID:                    row.ID.String(),
		UserID:                row.UserID.String(),
		ExcludedTitleKeywords: row.ExcludedTitleKeywords,
		ExcludedCompanies:     row.ExcludedCompanies,
		ExcludedLocations:     row.ExcludedLocations,
		NotifyThreshold:       int(row.NotifyThreshold),
		Preferences:           prefs,
		UpdatedAt:             row.UpdatedAt.Time,
	}, nil
}

func toJobDTO(row sqlc.GetJobForScoringRow) dto.Job {
	return dto.Job{
		ID: row.ID.String(), Title: row.Title, Location: row.Location, URL: row.Url,
		CompanySlug: row.CompanySlug, Source: row.Source, UpdatedAt: row.UpdatedAt.Time,
		ScrapedAt: row.ScrapedAt.Time, Description: row.Description, SalaryRaw: row.SalaryRaw,
		WorkArrangement: row.WorkArrangement, CompanyID: uuidString(row.CompanyID),
		BoardID: uuidString(row.PrimaryBoardID), ProviderPostingID: row.ProviderPostingID.String,
		ContentFingerprint: row.ContentFingerprint.String,
	}
}

func toScoringInputJobDTO(row sqlc.ListScoringInputJobsRow) dto.Job {
	return dto.Job{
		ID: row.ID.String(), Title: row.Title, Location: row.Location, URL: row.Url,
		CompanySlug: row.CompanySlug, Source: row.Source, UpdatedAt: row.UpdatedAt.Time,
		ScrapedAt: row.ScrapedAt.Time, Description: row.Description, SalaryRaw: row.SalaryRaw,
		WorkArrangement: row.WorkArrangement, CompanyID: uuidString(row.CompanyID),
		BoardID: uuidString(row.PrimaryBoardID), ProviderPostingID: row.ProviderPostingID.String,
		ContentFingerprint: row.ContentFingerprint.String,
	}
}

func uuidString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return id.String()
}

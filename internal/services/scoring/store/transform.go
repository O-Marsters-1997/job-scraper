package store

import (
	"encoding/json"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/store/sqlc"
)

func toScoringOptionDTO(row sqlc.ScoringOption) dto.ScoringOption {
	return dto.ScoringOption{
		ID:        row.ID,
		Dimension: dto.Dimension(row.Dimension),
		Label:     row.Label,
		Question:  row.Question,
		RetiredAt: data.TimePtr(row.RetiredAt),
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
		RequiredLocations:     row.RequiredLocations,
		RequiredTitleKeywords: row.RequiredTitleKeywords,
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
		WorkArrangement: row.WorkArrangement, CompanyID: row.CompanyID.String(),
		BoardID: row.PrimaryBoardID.String(), ProviderPostingID: row.ProviderPostingID.String,
		ContentFingerprint: row.ContentFingerprint.String,
	}
}

func toScoreFeedbackDTO(row sqlc.ScoreFeedback) (dto.ScoreFeedback, error) {
	picks := []dto.Pick{}
	if err := json.Unmarshal(row.Picks, &picks); err != nil {
		return dto.ScoreFeedback{}, err
	}
	var snapshot dto.ScoreFeedbackSnapshot
	if err := json.Unmarshal(row.Snapshot, &snapshot); err != nil {
		return dto.ScoreFeedback{}, err
	}
	return dto.ScoreFeedback{
		ID: row.ID.String(), Kind: row.Kind, Reason: row.Reason, Picks: picks,
		Model: row.Model, Snapshot: snapshot, CreatedAt: row.CreatedAt.Time,
	}, nil
}

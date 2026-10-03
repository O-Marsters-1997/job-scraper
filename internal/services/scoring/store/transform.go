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

func toListedScoreFeedbackDTO(row sqlc.ListScoreFeedbackRow) (dto.ScoreFeedback, error) {
	out, err := toScoreFeedbackDTO(sqlc.ScoreFeedback{
		ID: row.ID, UserID: row.UserID, JobID: row.JobID, Kind: row.Kind, Direction: row.Direction,
		Reason: row.Reason, Picks: row.Picks, Model: row.Model, Snapshot: row.Snapshot, CreatedAt: row.CreatedAt,
	})
	out.PicksChanged, out.ModelChanged = row.PicksChanged, row.ModelChanged
	return out, err
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
	out := dto.ScoreFeedback{
		ID: row.ID.String(), Kind: row.Kind, Reason: row.Reason, Picks: picks,
		Model: row.Model, Snapshot: snapshot, CreatedAt: row.CreatedAt.Time,
	}
	if row.Direction.Valid {
		out.Direction = &row.Direction.String
	}
	if row.JobID.Valid {
		id := row.JobID.String()
		out.JobID = &id
	}
	return out, nil
}

func toGradeDTO(row sqlc.JobGrade) dto.Grade {
	g := dto.Grade{
		JobID: row.JobID.String(), Grade: row.Grade, Reasons: nonNilStrings(row.Reasons),
		ScoreModel: row.ScoreModel.String, UpdatedAt: row.UpdatedAt.Time,
	}
	if row.ScoreAtGrade.Valid {
		score := int(row.ScoreAtGrade.Int32)
		g.ScoreAtGrade = &score
	}
	return g
}

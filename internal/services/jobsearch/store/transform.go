package store

import (
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
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
		ID:              row.ID.String(),
		Title:           row.Title,
		Location:        row.Location,
		URL:             row.Url,
		CompanySlug:     row.CompanySlug,
		Source:          row.Source,
		UpdatedAt:       row.UpdatedAt.Time,
		ScrapedAt:       row.ScrapedAt.Time,
		SalaryRaw:       row.SalaryRaw,
		WorkArrangement: row.WorkArrangement,
	}
	if err := unmarshalBreakdown(row.Breakdown, &j.Breakdown); err != nil {
		return dto.Job{}, err
	}
	j.SuitabilityScore = optionalInt32(row.SuitabilityScore)
	j.CompanyID = row.CompanyID.String()
	j.BoardID = row.PrimaryBoardID.String()
	j.ProviderPostingID = row.ProviderPostingID.String
	j.ContentFingerprint = row.ContentFingerprint.String
	return j, nil
}

func toGetJobDTO(row sqlc.GetJobRow) (dto.Job, error) {
	j := dto.Job{
		ID:              row.ID.String(),
		Title:           row.Title,
		Location:        row.Location,
		URL:             row.Url,
		CompanySlug:     row.CompanySlug,
		Source:          row.Source,
		UpdatedAt:       row.UpdatedAt.Time,
		ScrapedAt:       row.ScrapedAt.Time,
		Description:     row.Description,
		SalaryRaw:       row.SalaryRaw,
		WorkArrangement: row.WorkArrangement,
	}
	if err := unmarshalBreakdown(row.Breakdown, &j.Breakdown); err != nil {
		return dto.Job{}, err
	}
	j.SuitabilityScore = optionalInt32(row.SuitabilityScore)
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

func toCompanyForUserDTO(row sqlc.ListCompaniesForUserRow) dto.Company {
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
		CreatedAt:          row.CreatedAt.Time,
	}
	board.VerifiedAt = data.TimePtr(row.VerifiedAt)
	board.LastLinkedAt = data.TimePtr(row.LastLinkedAt)
	board.RetiredAt = data.TimePtr(row.RetiredAt)
	return board
}

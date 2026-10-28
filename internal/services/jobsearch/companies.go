package jobsearch

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
	"github.com/ollymarsters/job-scraper/internal/slug"
)

const defaultCheckIntervalMinutes = 360

func (s *Service) CreateCompany(ctx context.Context, userID string, in dto.CreateCompanyInput) (dto.Company, error) {
	if in.URL == "" {
		return dto.Company{}, apperr.Invalid("url is required")
	}
	source, token, ok := detect.ResolveBoard(in.URL)
	if !ok {
		return dto.Company{}, apperr.Unprocessable("could not resolve an ATS board from that URL")
	}

	company, err := s.store.UpsertCompany(ctx, dto.CompanyUpsert{
		Slug: token,
		Name: slug.Humanize(token),
		Kind: dto.KindDirect,
	})
	if err != nil {
		return dto.Company{}, err
	}
	if _, err := s.store.UpsertCandidateBoard(ctx, company.ID, source, token); err != nil {
		return dto.Company{}, err
	}

	if in.Track == nil || *in.Track {
		if _, err := s.store.SetCompanyTracking(ctx, userID, company.ID, true, defaultCheckIntervalMinutes); err != nil {
			return dto.Company{}, err
		}
	}
	return company, nil
}

func (s *Service) ListCompanies(ctx context.Context, userID string, q dto.CompaniesQuery) (dto.CompanyPage, error) {
	limit, err := parsePageLimit(q.Limit)
	if err != nil {
		return dto.CompanyPage{}, err
	}
	offset, err := parsePageOffset(q.Offset)
	if err != nil {
		return dto.CompanyPage{}, err
	}
	sort, err := parseCompanySort(q.Sort)
	if err != nil {
		return dto.CompanyPage{}, err
	}
	poster, err := parseCompanyPoster(q.Poster)
	if err != nil {
		return dto.CompanyPage{}, err
	}
	options := dto.CompanyPageOptions{
		Limit:         int32(limit),
		Offset:        int32(offset),
		Search:        strings.TrimSpace(q.Q),
		TrackedOnly:   q.Tracked == "1",
		FavouriteOnly: q.Favourite == "1",
		NoBoardOnly:   q.NoBoard == "1",
		Sort:          sort,
		Poster:        poster,
		ATSSources:    splitCSV(q.ATS),
	}

	page, err := s.store.PageCompaniesForUser(ctx, userID, options)
	if err != nil {
		if errors.Is(err, store.ErrInvalidID) {
			return dto.CompanyPage{}, apperr.Invalid("invalid user ID")
		}
		return dto.CompanyPage{}, err
	}
	if page.Items == nil {
		page.Items = []dto.Company{}
	}
	return page, nil
}

func (s *Service) CompanyATSCounts(ctx context.Context, _ string) ([]dto.ATSCount, error) {
	return s.store.CountCompaniesByATS(ctx)
}

func splitCSV(raw string) []string {
	var out []string
	for part := range strings.SplitSeq(raw, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parsePageOffset(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	offset, err := strconv.Atoi(raw)
	if err != nil || offset < 0 || offset > math.MaxInt32 {
		return 0, apperr.Invalid("offset must be a non-negative integer")
	}
	return offset, nil
}

func parseCompanyPoster(raw string) (dto.Poster, error) {
	switch poster := dto.Poster(raw); poster {
	case "", dto.PosterAgency, dto.PosterDirect:
		return poster, nil
	default:
		return "", apperr.Invalid("poster must be agency or direct")
	}
}

func parseCompanySort(raw string) (dto.CompanySort, error) {
	switch sort := dto.CompanySort(raw); sort {
	case "":
		return dto.CompanySortRelevance, nil
	case dto.CompanySortRelevance, dto.CompanySortAlphabetical, dto.CompanySortOpenJobs:
		return sort, nil
	default:
		return "", apperr.Invalid("sort must be relevance, alphabetical or open_jobs")
	}
}

func (s *Service) GetCompany(ctx context.Context, userID, id string) (dto.Company, error) {
	company, err := s.store.GetCompanyForUser(ctx, userID, id)
	switch {
	case errors.Is(err, data.ErrNotFound):
		return dto.Company{}, apperr.NotFound("company not found")
	case errors.Is(err, store.ErrInvalidID):
		return dto.Company{}, apperr.Invalid("invalid company ID")
	default:
		return company, err
	}
}

func (s *Service) FavouriteCompany(ctx context.Context, userID, companyID string) (dto.Company, error) {
	return s.setCompanyFavourite(ctx, userID, companyID, true)
}

func (s *Service) UnfavouriteCompany(ctx context.Context, userID, companyID string) (dto.Company, error) {
	return s.setCompanyFavourite(ctx, userID, companyID, false)
}

func (s *Service) setCompanyFavourite(ctx context.Context, userID, companyID string, favourite bool) (dto.Company, error) {
	if _, err := s.GetCompany(ctx, userID, companyID); err != nil {
		return dto.Company{}, err
	}
	if err := s.store.SetCompanyFavourite(ctx, userID, companyID, favourite); err != nil {
		return dto.Company{}, err
	}
	return s.GetCompany(ctx, userID, companyID)
}

func (s *Service) SetCompanyKind(ctx context.Context, userID string, in dto.SetCompanyKindInput) (dto.Company, error) {
	if in.Kind != dto.KindAgency && in.Kind != dto.KindDirect {
		return dto.Company{}, apperr.Invalid("kind must be agency or direct")
	}
	return s.applyCompanyKind(ctx, userID, in.CompanyID, in.Kind)
}

func (s *Service) UndoCompanyKind(ctx context.Context, userID string, in dto.UndoCompanyKindInput) (dto.Company, error) {
	switch in.Kind {
	case dto.KindUnclassified, dto.KindLikelyAgency, dto.KindLikelyDirect, dto.KindAgency, dto.KindDirect:
	default:
		return dto.Company{}, apperr.Invalid("unknown company kind")
	}
	return s.applyCompanyKind(ctx, userID, in.CompanyID, in.Kind)
}

func (s *Service) applyCompanyKind(ctx context.Context, userID, companyID string, kind dto.CompanyKind) (dto.Company, error) {
	err := s.store.SetCompanyKind(ctx, companyID, kind)
	switch {
	case errors.Is(err, data.ErrNotFound):
		return dto.Company{}, apperr.NotFound("company not found")
	case errors.Is(err, store.ErrInvalidID):
		return dto.Company{}, apperr.Invalid("invalid company ID")
	case err != nil:
		return dto.Company{}, err
	}
	return s.GetCompany(ctx, userID, companyID)
}

func (s *Service) SetCompaniesKind(ctx context.Context, _ string, in dto.SetCompaniesKindInput) (dto.CompaniesKindResult, error) {
	if in.Kind != dto.KindAgency && in.Kind != dto.KindDirect {
		return dto.CompaniesKindResult{}, apperr.Invalid("kind must be agency or direct")
	}
	ids, err := distinctCompanyIDs(in.CompanyIDs)
	if err != nil {
		return dto.CompaniesKindResult{}, err
	}
	for _, id := range ids {
		err := s.store.SetCompanyKind(ctx, id, in.Kind)
		switch {
		case errors.Is(err, data.ErrNotFound):
			return dto.CompaniesKindResult{}, apperr.NotFound("company not found")
		case errors.Is(err, store.ErrInvalidID):
			return dto.CompaniesKindResult{}, apperr.Invalid("invalid company ID")
		case err != nil:
			return dto.CompaniesKindResult{}, err
		}
	}
	return dto.CompaniesKindResult{Updated: len(ids)}, nil
}

func (s *Service) SetCompanyTracking(ctx context.Context, userID string, in dto.SetCompanyTrackingInput) (dto.CompanyTracking, error) {
	if in.Enabled == nil {
		return dto.CompanyTracking{}, apperr.Invalid("enabled is required")
	}
	interval := 0
	if in.CheckIntervalMinutes != nil {
		interval = *in.CheckIntervalMinutes
		if interval < 60 {
			return dto.CompanyTracking{}, apperr.Invalid("check_interval_minutes must be at least 60")
		}
	}

	company, err := s.store.GetCompany(ctx, in.CompanyID)
	if err != nil {
		return dto.CompanyTracking{}, err
	}
	return s.store.SetCompanyTracking(ctx, userID, company.ID, *in.Enabled, interval)
}

func (s *Service) ListTrackedCompanies(ctx context.Context, userID string) ([]dto.TrackedCompany, error) {
	companies, err := s.store.ListTrackedCompaniesForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range companies {
		for j := range companies[i].Boards {
			b := &companies[i].Boards[j]
			b.URL = detect.BoardURL(b.Source, b.BoardToken)
		}
	}
	return companies, nil
}

func (s *Service) SetCompanyReview(ctx context.Context, userID string, in dto.SetCompanyReviewInput) (dto.CompanyTracking, error) {
	switch in.State {
	case "new", "kept", "dismissed":
	default:
		return dto.CompanyTracking{}, apperr.Invalid("state must be new, kept or dismissed")
	}
	return s.store.SetCompanyReviewState(ctx, userID, in.CompanyID, in.State)
}

func (s *Service) UntrackCompany(ctx context.Context, userID, companyID string) error {
	return s.store.DeleteCompanyTracking(ctx, userID, companyID)
}

func (s *Service) ListCompanyBoards(ctx context.Context, _, companyID string) ([]dto.CompanyBoard, error) {
	if _, err := s.store.GetCompany(ctx, companyID); err != nil {
		return nil, err
	}
	return s.store.ListCompanyBoards(ctx, companyID)
}

func (s *Service) AddCompanyBoard(ctx context.Context, _ string, in dto.AddCompanyBoardInput) (dto.CompanyBoard, error) {
	companyID := in.CompanyID
	if _, err := s.store.GetCompany(ctx, companyID); err != nil {
		return dto.CompanyBoard{}, err
	}
	source, token, ok := detect.ResolveBoard(in.URL)
	if !ok {
		return dto.CompanyBoard{}, apperr.Unprocessable("could not resolve an ATS board from that URL")
	}
	board, err := s.store.UpsertCandidateBoard(ctx, companyID, source, token)
	if err != nil {
		return dto.CompanyBoard{}, err
	}
	if in.Confirm && board.Status == dto.BoardCandidate {
		task := queue.Task{Version: 1, ID: uuid.NewString(), Source: source, Kind: queue.BoardVerifyTask, CompanyID: companyID, BoardToken: token}
		if err := s.queue.Publish(ctx, task); err != nil {
			return dto.CompanyBoard{}, err
		}
	}
	return board, nil
}

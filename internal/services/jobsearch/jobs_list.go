package jobsearch

import (
	"context"
	"errors"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
)

const (
	defaultJobPageSize = 25
	maxJobPageSize     = 100
)

func (s *Service) List(ctx context.Context, userID string, q dto.JobsQuery) (dto.JobPage, error) {
	options, err := parseJobsQuery(q)
	if err != nil {
		return dto.JobPage{}, err
	}
	options.Now = time.Now()
	options.ExcludedCompanySlugs, err = s.excludedCompanySlugs(ctx, userID)
	if err != nil {
		return dto.JobPage{}, err
	}

	page, err := s.store.Page(ctx, userID, options)
	if err != nil {
		if errors.Is(err, store.ErrInvalidID) {
			return dto.JobPage{}, apperr.Invalid("invalid company ID")
		}
		return dto.JobPage{}, err
	}
	if page.Items == nil {
		page.Items = []dto.Job{}
	}
	return page, nil
}

func parseJobsQuery(q dto.JobsQuery) (dto.JobPageOptions, error) {
	pageNumber, err := parsePositiveInt(q.Page, 1, "page")
	if err != nil {
		return dto.JobPageOptions{}, err
	}
	pageSize, err := parsePositiveInt(q.PageSize, defaultJobPageSize, "page_size")
	if err != nil {
		return dto.JobPageOptions{}, err
	}
	pageSize = min(pageSize, maxJobPageSize)
	if pageNumber-1 > math.MaxInt32/pageSize {
		return dto.JobPageOptions{}, apperr.Invalid("page is too large")
	}

	graded, err := oneOf(q.Graded, "graded", "all", "graded", "ungraded")
	if err != nil {
		return dto.JobPageOptions{}, err
	}
	seen, err := oneOf(q.Seen, "seen", "all", "seen", "unseen")
	if err != nil {
		return dto.JobPageOptions{}, err
	}
	poster, err := oneOf(q.Poster, "poster", "all", string(dto.PosterAgency), string(dto.PosterDirect))
	if err != nil {
		return dto.JobPageOptions{}, err
	}
	if poster == "all" {
		poster = ""
	}

	options := dto.JobPageOptions{
		Limit:         pageSize,
		Offset:        (pageNumber - 1) * pageSize,
		Sort:          parseJobSort(q.Sort),
		CompanyID:     q.CompanyID,
		ScoredOnly:    q.Scored == "1",
		Graded:        graded,
		Seen:          seen,
		Poster:        dto.Poster(poster),
		FavouriteOnly: q.Favourite == "1",
		Sources:       splitList(q.Source),
		Arrangements:  splitList(strings.ToLower(q.Work)),
		SalaryOnly:    q.Salary == "1",
		Search:        q.Q,
		SkipTotal:     q.SkipTotal == "1",
	}
	if q.Suit != "" {
		suit, err := strconv.ParseInt(q.Suit, 10, 32)
		if err != nil {
			return dto.JobPageOptions{}, apperr.Invalid("suit must be an integer")
		}
		options.MinSuitability = new(int32(suit))
	}
	if options.SalaryMin, err = parseOptionalFloat(q.SalaryMin, "salary_min"); err != nil {
		return dto.JobPageOptions{}, err
	}
	if options.SalaryMax, err = parseOptionalFloat(q.SalaryMax, "salary_max"); err != nil {
		return dto.JobPageOptions{}, err
	}
	return options, nil
}

func parseJobSort(raw string) dto.JobSort {
	if sort := dto.JobSort(raw); sort == dto.JobSortNewest || sort == dto.JobSortBest {
		return sort
	}
	return dto.JobSortRelevant
}

func parsePositiveInt(raw string, fallback int32, name string) (int32, error) {
	if raw == "" {
		return fallback, nil
	}
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || n < 1 {
		return 0, apperr.Invalid(name + " must be a positive integer")
	}
	return int32(n), nil
}

func parseOptionalFloat(raw, name string) (*float64, error) {
	if raw == "" {
		return nil, nil
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return nil, apperr.Invalid(name + " must be a number")
	}
	return &f, nil
}

func oneOf(raw, name, fallback string, allowed ...string) (string, error) {
	if raw == "" {
		return fallback, nil
	}
	if slices.Contains(allowed, raw) {
		return raw, nil
	}
	return "", apperr.Invalid(name + " must be one of " + strings.Join(allowed, ", "))
}

func splitList(raw string) []string {
	var out []string
	for item := range strings.SplitSeq(raw, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

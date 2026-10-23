// Package store is the jobsearch context's Postgres store: jobs, companies,
// company boards, source targets and candidates, plus a score-sorted read-join
// onto job_scores for the job page.
package store

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/jobmatch"
	"github.com/ollymarsters/job-scraper/internal/runwindow"
	"github.com/ollymarsters/job-scraper/internal/salary"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store/sqlc"
	"github.com/ollymarsters/job-scraper/internal/services/sourcetargets"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
)

var (
	ErrInvalidID             = errors.New("invalid ID")
	ErrCanonicalConflict     = errors.New("canonical job identity conflict")
	ErrBoardConflict         = apperr.Conflict("board belongs to another company")
	ErrSourceTargetExists    = apperr.Conflict("source target already exists")
	ErrBoardClaimUnavailable = errors.New("board claim unavailable")
)

// ScoringWriter is scoring's tx-scoped facade, called from within
// jobsearch's own transactions instead of writing scoring's tables
// directly (ADR 0011).
type ScoringWriter interface {
	JobsChanged(ctx context.Context, tx pgx.Tx, jobIDs []string, firstDiscovery bool) error
	CompanyFavouriteChanged(ctx context.Context, tx pgx.Tx, userID, companyID string) error
	JobsClosed(ctx context.Context, tx pgx.Tx, jobIDs []string) error
	CompanyTracked(ctx context.Context, tx pgx.Tx, userID, companyID string) error
}

type Store struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
	scoring ScoringWriter
}

func New(pool *pgxpool.Pool, scoring ScoringWriter) *Store {
	return &Store{pool: pool, queries: sqlc.New(pool), scoring: scoring}
}

func optionalUUID(id string) (pgtype.UUID, error) {
	if id == "" {
		return pgtype.UUID{}, nil
	}
	return data.UUID(id)
}

// nonNil keeps an empty list from reaching Postgres as NULL, which would make
// `<> ALL(NULL)` hide every job.
func nonNil(items []string) []string {
	if items == nil {
		return []string{}
	}
	return items
}

func (s *Store) Page(ctx context.Context, userID string, options dto.JobPageOptions) (dto.JobPage, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.JobPage{}, ErrInvalidID
	}
	var companyID pgtype.UUID
	if options.CompanyID != "" {
		if companyID, err = data.UUID(options.CompanyID); err != nil {
			return dto.JobPage{}, ErrInvalidID
		}
	}
	now := options.Now
	if now.IsZero() {
		now = time.Now()
	}
	filters := sqlc.CountJobsParams{
		UserID:               uid,
		Now:                  pgtype.Timestamptz{Time: now, Valid: true},
		ExcludedCompanySlugs: nonNil(options.ExcludedCompanySlugs),
		CompanyID:            companyID,
		ScoredOnly:           options.ScoredOnly,
		Graded:               cmp.Or(options.Graded, "all"),
		Seen:                 cmp.Or(options.Seen, "all"),
		Poster:               cmp.Or(string(options.Poster), "all"),
		FavouriteOnly:        options.FavouriteOnly,
		Sources:              nonNil(options.Sources),
		Arrangements:         nonNil(options.Arrangements),
		SalaryOnly:           options.SalaryOnly,
		SearchPattern:        likePattern(options.Search),
		Bands:                nonNil(options.Bands),
	}
	if options.ScrapedSince != nil {
		filters.ScrapedSince = pgtype.Timestamptz{Time: *options.ScrapedSince, Valid: true}
	}
	if options.MinSuitability != nil {
		filters.MinSuitability = pgtype.Int4{Int32: *options.MinSuitability, Valid: true}
	}
	if options.SalaryMin != nil {
		filters.SalaryFloor = pgtype.Float8{Float64: *options.SalaryMin, Valid: true}
	}
	if options.SalaryMax != nil {
		filters.SalaryCeiling = pgtype.Float8{Float64: *options.SalaryMax, Valid: true}
	}

	rows, err := s.queries.PageJobs(ctx, sqlc.PageJobsParams{
		UserID: filters.UserID, Now: filters.Now, ExcludedCompanySlugs: filters.ExcludedCompanySlugs,
		CompanyID: filters.CompanyID, ScoredOnly: filters.ScoredOnly, MinSuitability: filters.MinSuitability,
		Graded: filters.Graded, Seen: filters.Seen, Poster: filters.Poster, FavouriteOnly: filters.FavouriteOnly,
		Sources: filters.Sources, Arrangements: filters.Arrangements, SalaryOnly: filters.SalaryOnly,
		SalaryFloor: filters.SalaryFloor, SalaryCeiling: filters.SalaryCeiling, SearchPattern: filters.SearchPattern,
		Bands: filters.Bands, ScrapedSince: filters.ScrapedSince,
		Sort: string(options.Sort), PageOffset: options.Offset, PageLimit: options.Limit,
	})
	if err != nil {
		return dto.JobPage{}, fmt.Errorf("page jobs: %w", err)
	}
	page := dto.JobPage{Items: make([]dto.Job, len(rows))}
	for i, row := range rows {
		job, err := toPageJobDTO(row)
		if err != nil {
			return dto.JobPage{}, fmt.Errorf("page jobs: %w", err)
		}
		page.Items[i] = job
	}
	if err := s.attachListings(ctx, page.Items, rows); err != nil {
		return dto.JobPage{}, err
	}

	if len(rows) > 0 && len(rows) < int(options.Limit) {
		page.Total = int(options.Offset) + len(rows)
		return page, nil
	}
	total, err := s.queries.CountJobs(ctx, filters)
	if err != nil {
		return dto.JobPage{}, fmt.Errorf("count jobs: %w", err)
	}
	page.Total = int(total)
	return page, nil
}

func (s *Store) JobStats(ctx context.Context, userID string, options dto.JobStatsOptions) (dto.JobStats, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.JobStats{}, ErrInvalidID
	}
	row, err := s.queries.JobStats(ctx, sqlc.JobStatsParams{
		UserID:               uid,
		Now:                  pgtype.Timestamptz{Time: options.Now, Valid: true},
		DayStart:             pgtype.Timestamptz{Time: options.DayStart, Valid: true},
		ExcludedCompanySlugs: nonNil(options.ExcludedCompanySlugs),
	})
	if err != nil {
		return dto.JobStats{}, fmt.Errorf("job stats: %w", err)
	}
	return dto.JobStats{
		Total:        int(row.Total),
		Sources:      int(row.Sources),
		NewToday:     int(row.NewToday),
		TodaySources: int(row.TodaySources),
	}, nil
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func likePattern(search string) string {
	if search = strings.TrimSpace(search); search == "" {
		return ""
	}
	return "%" + likeEscaper.Replace(search) + "%"
}

func (s *Store) GetJob(ctx context.Context, jobID, userID string) (dto.Job, error) {
	jid, err := data.UUID(jobID)
	if err != nil {
		return dto.Job{}, ErrInvalidID
	}
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.Job{}, ErrInvalidID
	}
	row, err := s.queries.GetJob(ctx, sqlc.GetJobParams{ID: jid, UserID: uid})
	if err != nil {
		return dto.Job{}, data.QueryErr("GetJob", err)
	}
	job, err := toGetJobDTO(row)
	if err != nil {
		return dto.Job{}, fmt.Errorf("store.GetJob: %w", err)
	}
	return job, nil
}

func (s *Store) FindJobByPosting(ctx context.Context, rawURL, source, token, postingID string) (dto.Job, error) {
	if normalized, err := normalizeJobURL(rawURL); err == nil {
		rawURL = normalized
	}
	row, err := s.queries.FindJobByPosting(ctx, sqlc.FindJobByPostingParams{
		Url: rawURL, Source: source, Token: token, PostingID: postingID,
	})
	if err != nil {
		return dto.Job{}, data.QueryErr("FindJobByPosting", err)
	}
	return dto.Job{
		ID:                 row.ID.String(),
		Title:              row.Title,
		Location:           row.Location,
		URL:                row.Url,
		CompanySlug:        row.CompanySlug,
		CompanyID:          row.CompanyID.String(),
		BoardID:            row.PrimaryBoardID.String(),
		ProviderPostingID:  row.ProviderPostingID.String,
		ContentFingerprint: row.ContentFingerprint.String,
		Source:             row.Source,
		PostedAt:           data.TimePtr(row.PostedAt),
		PostedAfter:        data.TimePtr(row.PostedAfter),
		RepostedAt:         data.TimePtr(row.RepostedAt),
		ScrapedAt:          row.ScrapedAt.Time,
		FirstDiscoveredAt:  row.FirstDiscoveredAt.Time,
		Description:        row.Description,
		SalaryRaw:          row.SalaryRaw,
		WorkArrangement:    row.WorkArrangement,
	}, nil
}

func (s *Store) ListJobListings(ctx context.Context, jobID string) ([]dto.JobListing, error) {
	jid, err := data.UUID(jobID)
	if err != nil {
		return nil, ErrInvalidID
	}
	rows, err := s.queries.ListJobListings(ctx, jid)
	if err != nil {
		return nil, fmt.Errorf("store.ListJobListings: %w", err)
	}
	listings := make([]dto.JobListing, len(rows))
	for i, row := range rows {
		listings[i] = dto.JobListing{Source: row.Source, URL: row.NormalizedUrl, FirstSeenAt: row.FirstSeenAt.Time}
	}
	return listings, nil
}

func (s *Store) attachListings(ctx context.Context, jobs []dto.Job, rows []sqlc.PageJobsRow) error {
	if len(jobs) == 0 {
		return nil
	}
	ids := make([]pgtype.UUID, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	listings, err := s.queries.ListJobListingsForJobs(ctx, ids)
	if err != nil {
		return fmt.Errorf("list job listings: %w", err)
	}
	byJob := make(map[string][]dto.JobListing, len(jobs))
	for _, l := range listings {
		id := l.JobID.String()
		byJob[id] = append(byJob[id], dto.JobListing{Source: l.Source, URL: l.NormalizedUrl, FirstSeenAt: l.FirstSeenAt.Time})
	}
	for i := range jobs {
		jobs[i].Listings = byJob[jobs[i].ID]
	}
	return nil
}

func (s *Store) MarkJobsSeen(ctx context.Context, userID string, jobIDs []string, seen bool) error {
	uid, err := data.UUID(userID)
	if err != nil {
		return ErrInvalidID
	}
	ids := make([]pgtype.UUID, len(jobIDs))
	for i, id := range jobIDs {
		if ids[i], err = data.UUID(id); err != nil {
			return ErrInvalidID
		}
	}
	if seen {
		err = s.queries.MarkJobsSeen(ctx, sqlc.MarkJobsSeenParams{UserID: uid, JobIds: ids})
	} else {
		err = s.queries.MarkJobsUnseen(ctx, sqlc.MarkJobsUnseenParams{UserID: uid, JobIds: ids})
	}
	if err != nil {
		return fmt.Errorf("store.MarkJobsSeen: %w", err)
	}
	return nil
}

func normalizeJobURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil {
		return "", fmt.Errorf("invalid job URL")
	}
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	u.RawQuery = StripTrackingParams(u.RawQuery)
	return u.String(), nil
}

// StripTrackingParams drops utm_*, gh_src, lever-source, lever-origin and ref
// from a raw query string, keeping the remaining parameters in order.
func StripTrackingParams(rawQuery string) string {
	var kept []string
	for _, pair := range strings.Split(rawQuery, "&") {
		key, _, _ := strings.Cut(pair, "=")
		switch {
		case pair == "", strings.HasPrefix(key, "utm_"), key == "gh_src", key == "lever-source", key == "lever-origin", key == "ref":
		default:
			kept = append(kept, pair)
		}
	}
	return strings.Join(kept, "&")
}

func jobFingerprint(job dto.Job) string {
	content, _ := json.Marshal([5]string{job.Title, job.Description, job.Location, job.SalaryRaw, job.WorkArrangement})
	return fmt.Sprintf("%x", sha256.Sum256(content))
}

func salaryBounds(raw string) (lo, hi pgtype.Float8) {
	floor, ceiling, ok := salary.Parse(raw)
	if !ok {
		return lo, hi
	}
	return pgtype.Float8{Float64: floor, Valid: true}, pgtype.Float8{Float64: ceiling, Valid: true}
}

func (s *Store) SaveCanonical(ctx context.Context, job dto.Job) (dto.Job, string, error) {
	normalizedURL, err := normalizeJobURL(job.URL)
	if err != nil {
		return dto.Job{}, "", err
	}
	job.URL = normalizedURL
	job.ContentFingerprint = jobFingerprint(job)
	companyID, err := optionalUUID(job.CompanyID)
	if err != nil {
		return dto.Job{}, "", fmt.Errorf("company ID: %w", err)
	}
	boardID, err := optionalUUID(job.BoardID)
	if err != nil {
		return dto.Job{}, "", fmt.Errorf("board ID: %w", err)
	}
	postingID := data.Text(job.ProviderPostingID)
	fingerprint := pgtype.Text{String: job.ContentFingerprint, Valid: true}
	postedAt := data.Timestamptz(job.PostedAt)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dto.Job{}, "", fmt.Errorf("begin canonical job: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := s.queries.WithTx(tx)

	identity := normalizedURL
	if boardID.Valid && postingID.Valid {
		identity = job.BoardID + ":" + job.ProviderPostingID
	}
	if err := queries.LockCanonicalJob(ctx, identity); err != nil {
		return dto.Job{}, "", fmt.Errorf("lock canonical job: %w", err)
	}

	previous, err := queries.FindCanonicalJob(ctx, sqlc.FindCanonicalJobParams{
		Url: normalizedURL, BoardID: boardID, PostingID: postingID,
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return dto.Job{}, "", fmt.Errorf("find canonical job: %w", err)
	}
	if previous.PrimaryBoardID.Valid && boardID.Valid &&
		(previous.PrimaryBoardID != boardID || previous.ProviderPostingID != postingID) {
		return dto.Job{}, "", fmt.Errorf("%w: URL belongs to another trusted posting", ErrCanonicalConflict)
	}

	salaryMin, salaryMax := salaryBounds(job.SalaryRaw)
	matchTitle := jobmatch.Title(job.Title)
	matchLocation := jobmatch.Location(job.Location)
	trusted := boardID.Valid && postingID.Valid
	isSecondaryListing := err == nil && previous.Url != normalizedURL &&
		(!trusted || previous.PrimaryBoardID != boardID || previous.ProviderPostingID != postingID)

	status := "new"
	var jobID pgtype.UUID
	switch {
	case isSecondaryListing:
		status = "unchanged"
		jobID = previous.ID
		job.URL = previous.Url
	case errors.Is(err, pgx.ErrNoRows):
		match, found, err := findMatch(ctx, queries, job, companyID, matchTitle, matchLocation, trusted)
		if err != nil {
			return dto.Job{}, "", err
		}
		if found && !trusted {
			status = "merged"
			jobID = match.ID
			job.URL = match.Url
			break
		}
		if found {
			status = "upgraded"
			jobID = match.ID
			err = queries.UpgradeCanonicalJob(ctx, sqlc.UpgradeCanonicalJobParams{
				Title: job.Title, Location: job.Location, Url: job.URL, Source: job.Source,
				PostedAt: postedAt, Description: job.Description, SalaryRaw: job.SalaryRaw,
				SalaryMin: salaryMin, SalaryMax: salaryMax,
				WorkArrangement: job.WorkArrangement, Fingerprint: fingerprint,
				MatchTitle:    pgtype.Text{String: matchTitle, Valid: true},
				MatchLocation: pgtype.Text{String: matchLocation, Valid: true},
				CompanyID:     companyID, CompanySlug: job.CompanySlug, BoardID: boardID, PostingID: postingID, ID: jobID,
			})
			if err != nil {
				return dto.Job{}, "", fmt.Errorf("upgrade canonical job: %w", err)
			}
			break
		}
		cardDate, err := candidatePostedAt(ctx, queries, job, matchTitle, matchLocation)
		if err != nil {
			return dto.Job{}, "", err
		}
		if cardDate != nil && (job.PostedAt == nil || cardDate.Before(*job.PostedAt)) {
			job.PostedAt = cardDate
			postedAt = data.Timestamptz(cardDate)
		}
		jobID, err = queries.InsertCanonicalJob(ctx, sqlc.InsertCanonicalJobParams{
			Title: job.Title, Location: job.Location, Url: job.URL, CompanySlug: job.CompanySlug,
			Source: job.Source, PostedAt: postedAt, Description: job.Description,
			SalaryRaw: job.SalaryRaw, SalaryMin: salaryMin, SalaryMax: salaryMax,
			WorkArrangement: job.WorkArrangement, DiscoveredVia: job.DiscoveredVia,
			CompanyID: companyID, BoardID: boardID, PostingID: postingID, Fingerprint: fingerprint,
			MatchTitle:    pgtype.Text{String: matchTitle, Valid: true},
			MatchLocation: pgtype.Text{String: matchLocation, Valid: true},
		})
		if err != nil {
			return dto.Job{}, "", fmt.Errorf("insert canonical job: %w", err)
		}
	default:
		jobID = previous.ID
		oldFingerprint := previous.ContentFingerprint
		if oldFingerprint == "" {
			oldFingerprint = jobFingerprint(dto.Job{
				Title: previous.Title, Description: previous.Description, Location: previous.Location,
				SalaryRaw: previous.SalaryRaw, WorkArrangement: previous.WorkArrangement,
			})
		}
		if oldFingerprint == job.ContentFingerprint {
			status = "unchanged"
			err = queries.UpdateUnchangedCanonicalJob(ctx, sqlc.UpdateUnchangedCanonicalJobParams{
				CompanyID: companyID, CompanySlug: job.CompanySlug, BoardID: boardID, PostingID: postingID,
				Fingerprint: fingerprint, PostedAt: postedAt, ID: jobID,
			})
		} else {
			status = "changed"
			err = queries.UpdateChangedCanonicalJob(ctx, sqlc.UpdateChangedCanonicalJobParams{
				Title: job.Title, Location: job.Location, PostedAt: postedAt,
				Description: job.Description, SalaryRaw: job.SalaryRaw,
				SalaryMin: salaryMin, SalaryMax: salaryMax,
				WorkArrangement: job.WorkArrangement, Fingerprint: fingerprint,
				CompanyID: companyID, CompanySlug: job.CompanySlug, BoardID: boardID, PostingID: postingID, ID: jobID,
			})
		}
		if err != nil {
			return dto.Job{}, "", fmt.Errorf("update canonical job: %w", err)
		}
		job.URL = previous.Url
	}

	id := jobID.String()
	rows, err := queries.SaveCanonicalJobAlias(ctx, sqlc.SaveCanonicalJobAliasParams{
		Column1: jobID, NormalizedUrl: normalizedURL, Source: job.Source,
	})
	if err != nil {
		return dto.Job{}, "", fmt.Errorf("save job URL: %w", err)
	}
	if rows != 1 {
		return dto.Job{}, "", fmt.Errorf("%w: URL belongs to another canonical job", ErrCanonicalConflict)
	}
	if err := saveDiscoveryAlias(ctx, queries, jobID, job); err != nil {
		return dto.Job{}, "", err
	}
	if status == "new" || status == "changed" || status == "upgraded" {
		if err := s.scoring.JobsChanged(ctx, tx, []string{id}, status == "new"); err != nil {
			return dto.Job{}, "", fmt.Errorf("scoring.JobsChanged: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return dto.Job{}, "", fmt.Errorf("commit canonical job: %w", err)
	}
	job.ID = id
	return job, status, nil
}

func saveDiscoveryAlias(ctx context.Context, queries *sqlc.Queries, jobID pgtype.UUID, job dto.Job) error {
	if job.DiscoveredVia == "" || job.DiscoveryURL == "" {
		return nil
	}
	listingURL, err := normalizeJobURL(job.DiscoveryURL)
	if err == nil {
		_, err = queries.SaveCanonicalJobAlias(ctx, sqlc.SaveCanonicalJobAliasParams{
			Column1: jobID, NormalizedUrl: listingURL, Source: job.DiscoveredVia,
		})
	}
	if err != nil {
		return fmt.Errorf("save discovery listing URL: %w", err)
	}
	return nil
}

func findMatch(ctx context.Context, queries *sqlc.Queries, job dto.Job, companyID pgtype.UUID, matchTitle, matchLocation string, trusted bool) (sqlc.FindMatchCandidatesRow, bool, error) {
	candidates, err := matchingJobs(ctx, queries, job, companyID, matchTitle, matchLocation, trusted)
	if err != nil || len(candidates) != 1 {
		return sqlc.FindMatchCandidatesRow{}, false, err
	}
	return candidates[0], true, nil
}

func matchingJobs(ctx context.Context, queries *sqlc.Queries, job dto.Job, companyID pgtype.UUID, matchTitle, matchLocation string, trusted bool) ([]sqlc.FindMatchCandidatesRow, error) {
	if err := queries.LockCanonicalJob(ctx, "match:"+job.CompanySlug+"|"+matchTitle); err != nil {
		return nil, fmt.Errorf("lock match: %w", err)
	}
	candidates, err := queries.FindMatchCandidates(ctx, sqlc.FindMatchCandidatesParams{
		MatchTitle: pgtype.Text{String: matchTitle, Valid: true}, CompanySlug: job.CompanySlug, CompanyID: companyID,
	})
	if err != nil {
		return nil, fmt.Errorf("find match candidates: %w", err)
	}
	return slices.DeleteFunc(candidates, func(c sqlc.FindMatchCandidatesRow) bool {
		return (trusted && c.PrimaryBoardID.Valid) || !jobmatch.LocationsCompatible(matchLocation, c.MatchLocation)
	}), nil
}

func (s *Store) ApplyBoardCards(ctx context.Context, source string, cards []dto.Job) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin board card dates: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := s.queries.WithTx(tx)

	for _, card := range cards {
		if card.PostedAt == nil || card.URL == "" {
			continue
		}
		matches, err := matchingJobs(ctx, queries, card, pgtype.UUID{}, jobmatch.Title(card.Title), jobmatch.Location(card.Location), false)
		if err != nil {
			return err
		}
		switch len(matches) {
		case 0:
			normalized, err := NormalizeCandidateURL(card.URL)
			if err != nil {
				return err
			}
			if err := queries.UpsertBoardCardDate(ctx, sqlc.UpsertBoardCardDateParams{
				NormalizedUrl: normalized, Source: source, CardTitle: card.Title, CardCompany: card.CompanySlug,
				CardLocation: card.Location, PostedAt: data.Timestamptz(card.PostedAt),
			}); err != nil {
				return fmt.Errorf("upsert board card candidate: %w", err)
			}
		case 1:
			if err := queries.ApplyPostedDate(ctx, sqlc.ApplyPostedDateParams{PostedAt: data.Timestamptz(card.PostedAt), ID: matches[0].ID}); err != nil {
				return fmt.Errorf("apply board card date: %w", err)
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit board card dates: %w", err)
	}
	return nil
}

func candidatePostedAt(ctx context.Context, queries *sqlc.Queries, job dto.Job, matchTitle, matchLocation string) (*time.Time, error) {
	rows, err := queries.ListDatedCandidatesForCompany(ctx, job.CompanySlug)
	if err != nil {
		return nil, fmt.Errorf("list dated candidates: %w", err)
	}
	var earliest *time.Time
	for _, row := range rows {
		if jobmatch.Title(row.CardTitle) != matchTitle || !jobmatch.LocationsCompatible(matchLocation, jobmatch.Location(row.CardLocation)) {
			continue
		}
		if earliest == nil || row.PostedAt.Time.Before(*earliest) {
			earliest = &row.PostedAt.Time
		}
	}
	return earliest, nil
}

func (s *Store) UpsertCompany(ctx context.Context, c dto.CompanyUpsert) (dto.Company, error) {
	row, err := s.queries.UpsertCompany(ctx, sqlc.UpsertCompanyParams{
		Slug:              c.Slug,
		Name:              c.Name,
		AtsSource:         data.Text(c.ATSSource),
		AtsToken:          data.Text(c.ATSToken),
		Domain:            data.Text(c.Domain),
		LinkedinCompanyID: data.Text(c.LinkedInCompanyID),
		Kind:              string(c.Kind),
	})
	if err != nil {
		return dto.Company{}, fmt.Errorf("store.UpsertCompany: %w", err)
	}
	return toCompanyDTO(row), nil
}

func (s *Store) GetCompany(ctx context.Context, id string) (dto.Company, error) {
	cid, err := data.UUID(id)
	if err != nil {
		return dto.Company{}, err
	}
	row, err := s.queries.GetCompany(ctx, cid)
	if err != nil {
		return dto.Company{}, data.QueryErr("GetCompany", err)
	}
	return toCompanyDTO(row), nil
}

func (s *Store) PageCompaniesForUser(ctx context.Context, userID string, options dto.CompanyPageOptions) (dto.CompanyPage, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.CompanyPage{}, ErrInvalidID
	}
	rows, err := s.queries.PageCompaniesForUser(ctx, sqlc.PageCompaniesForUserParams{
		UserID:        uid,
		Search:        options.Search,
		TrackedOnly:   options.TrackedOnly,
		FavouriteOnly: options.FavouriteOnly,
		NoBoardOnly:   options.NoBoardOnly,
		Sort:          string(options.Sort),
		Poster:        string(options.Poster),
		AtsSources:    options.ATSSources,
		PageLimit:     options.Limit,
		PageOffset:    options.Offset,
	})
	if err != nil {
		return dto.CompanyPage{}, fmt.Errorf("store.PageCompaniesForUser: %w", err)
	}
	page := dto.CompanyPage{Items: make([]dto.Company, len(rows))}
	for i, r := range rows {
		page.Total = int(r.Total)
		page.Items[i] = toCompanyForUserDTO(sqlc.GetCompanyForUserRow{
			ID: r.ID, Slug: r.Slug, Name: r.Name, AtsSource: r.AtsSource, AtsToken: r.AtsToken,
			Domain: r.Domain, LinkedinCompanyID: r.LinkedinCompanyID, LastCrawledAt: r.LastCrawledAt,
			FirstSeenAt: r.FirstSeenAt, Kind: r.Kind, JobCount: r.JobCount, Tracked: r.Tracked, Favourite: r.Favourite,
			ReviewState: r.ReviewState, CheckIntervalMinutes: r.CheckIntervalMinutes, LastCheckedAt: r.LastCheckedAt,
		})
	}
	coverage, err := s.queries.CountCompanyBoardCoverage(ctx)
	if err != nil {
		return dto.CompanyPage{}, fmt.Errorf("store.PageCompaniesForUser coverage: %w", err)
	}
	page.AllCompanies, page.VerifiedBoardCompanies = int(coverage.AllCompanies), int(coverage.VerifiedCompanies)
	return page, nil
}

func (s *Store) CountCompaniesByATS(ctx context.Context) ([]dto.ATSCount, error) {
	rows, err := s.queries.CountCompaniesByATS(ctx)
	if err != nil {
		return nil, fmt.Errorf("store.CountCompaniesByATS: %w", err)
	}
	counts := make([]dto.ATSCount, len(rows))
	for i, r := range rows {
		counts[i] = dto.ATSCount{Source: r.Source, Companies: int(r.Companies)}
	}
	return counts, nil
}

func (s *Store) GetCompanyForUser(ctx context.Context, userID, id string) (dto.Company, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.Company{}, ErrInvalidID
	}
	cid, err := data.UUID(id)
	if err != nil {
		return dto.Company{}, ErrInvalidID
	}
	row, err := s.queries.GetCompanyForUser(ctx, sqlc.GetCompanyForUserParams{UserID: uid, ID: cid})
	if err != nil {
		return dto.Company{}, data.QueryErr("GetCompanyForUser", err)
	}
	return toCompanyForUserDTO(row), nil
}

func (s *Store) ListTrackedCompaniesForUser(ctx context.Context, userID string) ([]dto.TrackedCompany, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListTrackedCompaniesForUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListTrackedCompaniesForUser: %w", err)
	}
	boardRows, err := s.queries.ListTrackedCompanyBoards(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListTrackedCompaniesForUser boards: %w", err)
	}
	boards := make(map[string][]dto.TrackedBoard, len(rows))
	for _, b := range boardRows {
		companyID := b.CompanyID.String()
		boards[companyID] = append(boards[companyID], dto.TrackedBoard{
			ID: b.ID.String(), Source: b.Source, BoardToken: b.BoardToken, Status: dto.BoardStatus(b.Status),
		})
	}
	out := make([]dto.TrackedCompany, len(rows))
	for i, r := range rows {
		id := r.ID.String()
		out[i] = dto.TrackedCompany{
			ID: id, Name: r.Name, Slug: r.Slug, Enabled: r.Enabled, ReviewState: r.ReviewState,
			CheckIntervalMinutes: int(r.CheckIntervalMinutes),
			Boards:               []dto.TrackedBoard{},
			OpenJobs:             int(r.OpenJobs),
			RelevantJobs:         int(r.RelevantJobs),
		}
		if b, ok := boards[id]; ok {
			out[i].Boards = b
		}
		out[i].LastCheckedAt = data.TimePtr(r.LastCheckedAt)
	}
	return out, nil
}

func (s *Store) ListNewCompanies(ctx context.Context, userID string) ([]dto.NewCompany, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListNewCompaniesForUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListNewCompanies: %w", err)
	}
	boardRows, err := s.queries.ListTrackedCompanyBoards(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListNewCompanies boards: %w", err)
	}
	boards := make(map[string][]dto.TrackedBoard, len(rows))
	for _, b := range boardRows {
		companyID := b.CompanyID.String()
		boards[companyID] = append(boards[companyID], dto.TrackedBoard{
			ID: b.ID.String(), Source: b.Source, BoardToken: b.BoardToken, Status: dto.BoardStatus(b.Status),
		})
	}
	profileRows, err := s.queries.ListNewCompanyProfiles(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListNewCompanies profiles: %w", err)
	}
	profiles := make(map[string]*dto.CompanyProfile, len(profileRows))
	for _, p := range profileRows {
		var profile dto.CompanyProfile
		if err := json.Unmarshal(p.Data, &profile); err != nil {
			return nil, fmt.Errorf("store.ListNewCompanies profile: %w", err)
		}
		profiles[p.CompanyID.String()] = &profile
	}
	out := make([]dto.NewCompany, len(rows))
	for i, r := range rows {
		id := r.ID.String()
		out[i] = dto.NewCompany{ID: id, Name: r.Name, Slug: r.Slug, Boards: []dto.TrackedBoard{}, Profile: profiles[id]}
		if b, ok := boards[id]; ok {
			out[i].Boards = b
		}
	}
	return out, nil
}

func (s *Store) SaveCompanyProfile(ctx context.Context, companyID, source string, profile dto.CompanyProfile) error {
	cid, err := data.UUID(companyID)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("store.SaveCompanyProfile: %w", err)
	}
	if err := s.queries.UpsertCompanyProfile(ctx, sqlc.UpsertCompanyProfileParams{CompanyID: cid, Source: source, Data: raw}); err != nil {
		return fmt.Errorf("store.SaveCompanyProfile: %w", err)
	}
	return nil
}

func (s *Store) ListNewCompanyJobs(ctx context.Context, userID string) ([]dto.Job, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListNewCompanyJobs(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListNewCompanyJobs: %w", err)
	}
	out := make([]dto.Job, len(rows))
	for i, r := range rows {
		out[i] = dto.Job{
			CompanyID: r.CompanyID.String(), CompanySlug: r.CompanySlug, Title: r.Title, Location: r.Location,
			SuitabilityScore: optionalInt32(r.SuitabilityScore),
		}
	}
	return out, nil
}

func (s *Store) DeleteCompanyTracking(ctx context.Context, userID, companyID string) error {
	uid, err := data.UUID(userID)
	if err != nil {
		return err
	}
	cid, err := data.UUID(companyID)
	if err != nil {
		return err
	}
	n, err := s.queries.DeleteCompanyTracking(ctx, sqlc.DeleteCompanyTrackingParams{UserID: uid, CompanyID: cid})
	if err != nil {
		return fmt.Errorf("store.DeleteCompanyTracking: %w", err)
	}
	if n == 0 {
		return data.ErrNotFound
	}
	return nil
}

func (s *Store) SetCompanyTracking(ctx context.Context, userID, companyID string, enabled bool, interval int) (dto.CompanyTracking, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.CompanyTracking{}, err
	}
	cid, err := data.UUID(companyID)
	if err != nil {
		return dto.CompanyTracking{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dto.CompanyTracking{}, fmt.Errorf("begin company tracking: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := s.queries.WithTx(tx)
	row, err := queries.SetCompanyTracking(ctx, sqlc.SetCompanyTrackingParams{
		UserID: uid, CompanyID: cid, Enabled: enabled, CheckIntervalMinutes: int32(interval),
	})
	if err != nil {
		return dto.CompanyTracking{}, fmt.Errorf("store.SetCompanyTracking: %w", err)
	}
	if enabled {
		if err := queries.BackfillCompanyJobFingerprints(ctx, cid); err != nil {
			return dto.CompanyTracking{}, fmt.Errorf("backfill tracked company jobs: %w", err)
		}
		if err := s.scoring.CompanyTracked(ctx, tx, userID, companyID); err != nil {
			return dto.CompanyTracking{}, fmt.Errorf("scoring.CompanyTracked: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return dto.CompanyTracking{}, fmt.Errorf("commit company tracking: %w", err)
	}
	return dto.CompanyTracking{
		UserID: row.UserID.String(), CompanyID: row.CompanyID.String(),
		Enabled: row.Enabled, ReviewState: row.ReviewState, CheckIntervalMinutes: int(row.CheckIntervalMinutes),
	}, nil
}

func (s *Store) TrackDiscoveredCompany(ctx context.Context, userID, companyID string) (bool, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return false, err
	}
	cid, err := data.UUID(companyID)
	if err != nil {
		return false, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin discovered tracking: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := s.queries.WithTx(tx)
	n, err := queries.TrackDiscoveredCompany(ctx, sqlc.TrackDiscoveredCompanyParams{UserID: uid, CompanyID: cid})
	if err != nil {
		return false, fmt.Errorf("store.TrackDiscoveredCompany: %w", err)
	}
	if n == 0 {
		return false, nil
	}
	if err := queries.BackfillCompanyJobFingerprints(ctx, cid); err != nil {
		return false, fmt.Errorf("backfill tracked company jobs: %w", err)
	}
	if err := s.scoring.CompanyTracked(ctx, tx, userID, companyID); err != nil {
		return false, fmt.Errorf("scoring.CompanyTracked: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit discovered tracking: %w", err)
	}
	return true, nil
}

func (s *Store) SetCompanyReviewState(ctx context.Context, userID, companyID, state string) (dto.CompanyTracking, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.CompanyTracking{}, err
	}
	cid, err := data.UUID(companyID)
	if err != nil {
		return dto.CompanyTracking{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dto.CompanyTracking{}, fmt.Errorf("begin company review: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := s.queries.WithTx(tx)
	row, err := queries.SetCompanyReviewState(ctx, sqlc.SetCompanyReviewStateParams{UserID: uid, CompanyID: cid, ReviewState: state})
	if err != nil {
		return dto.CompanyTracking{}, data.QueryErr("SetCompanyReviewState", err)
	}
	if state == "dismissed" {
		if err := queries.RemoveCompanyFavourite(ctx, sqlc.RemoveCompanyFavouriteParams{UserID: uid, CompanyID: cid}); err != nil {
			return dto.CompanyTracking{}, fmt.Errorf("store.SetCompanyReviewState favourite: %w", err)
		}
		if err := s.scoring.CompanyFavouriteChanged(ctx, tx, userID, companyID); err != nil {
			return dto.CompanyTracking{}, fmt.Errorf("scoring.CompanyFavouriteChanged: %w", err)
		}
	}
	if row.Enabled {
		if err := queries.BackfillCompanyJobFingerprints(ctx, cid); err != nil {
			return dto.CompanyTracking{}, fmt.Errorf("backfill tracked company jobs: %w", err)
		}
		if err := s.scoring.CompanyTracked(ctx, tx, userID, companyID); err != nil {
			return dto.CompanyTracking{}, fmt.Errorf("scoring.CompanyTracked: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return dto.CompanyTracking{}, fmt.Errorf("commit company review: %w", err)
	}
	return dto.CompanyTracking{
		UserID: row.UserID.String(), CompanyID: row.CompanyID.String(),
		Enabled: row.Enabled, ReviewState: row.ReviewState, CheckIntervalMinutes: int(row.CheckIntervalMinutes),
	}, nil
}

func (s *Store) SetCompanyKind(ctx context.Context, companyID string, kind dto.CompanyKind) error {
	cid, err := data.UUID(companyID)
	if err != nil {
		return ErrInvalidID
	}
	n, err := s.queries.SetCompanyKind(ctx, sqlc.SetCompanyKindParams{ID: cid, Kind: string(kind)})
	if err != nil {
		return fmt.Errorf("store.SetCompanyKind: %w", err)
	}
	if n == 0 {
		return data.ErrNotFound
	}
	return nil
}

func (s *Store) SuggestCompanyKind(ctx context.Context, companyID string, kind dto.CompanyKind) (bool, error) {
	cid, err := data.UUID(companyID)
	if err != nil {
		return false, ErrInvalidID
	}
	n, err := s.queries.SuggestCompanyKind(ctx, sqlc.SuggestCompanyKindParams{ID: cid, Kind: string(kind)})
	if err != nil {
		return false, fmt.Errorf("store.SuggestCompanyKind: %w", err)
	}
	return n > 0, nil
}

const favouriteCheckIntervalMinutes = 60

func (s *Store) SetCompanyFavourite(ctx context.Context, userID, companyID string, favourite bool) error {
	uid, err := data.UUID(userID)
	if err != nil {
		return ErrInvalidID
	}
	cid, err := data.UUID(companyID)
	if err != nil {
		return ErrInvalidID
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin company favourite: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := s.queries.WithTx(tx)
	if favourite {
		if err := queries.AddCompanyFavourite(ctx, sqlc.AddCompanyFavouriteParams{UserID: uid, CompanyID: cid}); err != nil {
			return fmt.Errorf("store.SetCompanyFavourite: %w", err)
		}
		boards, err := queries.ListCompanyBoards(ctx, cid)
		if err != nil {
			return fmt.Errorf("store.SetCompanyFavourite boards: %w", err)
		}
		if slices.ContainsFunc(boards, isVerifiedATSBoard) {
			if err := s.trackFavourite(ctx, tx, queries, userID, companyID, uid, cid); err != nil {
				return err
			}
		}
	} else {
		if err := queries.RemoveCompanyFavourite(ctx, sqlc.RemoveCompanyFavouriteParams{UserID: uid, CompanyID: cid}); err != nil {
			return fmt.Errorf("store.SetCompanyFavourite: %w", err)
		}
		if err := queries.ResetTrackedCompanyInterval(ctx, sqlc.ResetTrackedCompanyIntervalParams{UserID: uid, CompanyID: cid}); err != nil {
			return fmt.Errorf("store.SetCompanyFavourite reset interval: %w", err)
		}
	}
	if err := s.scoring.CompanyFavouriteChanged(ctx, tx, userID, companyID); err != nil {
		return fmt.Errorf("scoring.CompanyFavouriteChanged: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit company favourite: %w", err)
	}
	return nil
}

func isVerifiedATSBoard(b sqlc.CompanyBoard) bool {
	role, ok := sourcespec.SourceRole(b.Source)
	return ok && role == sourcespec.RoleATS && b.Status == string(dto.BoardVerified)
}

func (s *Store) trackFavourite(ctx context.Context, tx pgx.Tx, queries *sqlc.Queries, userID, companyID string, uid, cid pgtype.UUID) error {
	if _, err := queries.SetCompanyTracking(ctx, sqlc.SetCompanyTrackingParams{
		UserID: uid, CompanyID: cid, Enabled: true, CheckIntervalMinutes: favouriteCheckIntervalMinutes,
	}); err != nil {
		return fmt.Errorf("store.SetCompanyFavourite track: %w", err)
	}
	if err := queries.KeepNewTrackedCompany(ctx, sqlc.KeepNewTrackedCompanyParams{UserID: uid, CompanyID: cid}); err != nil {
		return fmt.Errorf("store.SetCompanyFavourite keep: %w", err)
	}
	if err := queries.BackfillCompanyJobFingerprints(ctx, cid); err != nil {
		return fmt.Errorf("backfill tracked company jobs: %w", err)
	}
	if err := s.scoring.CompanyTracked(ctx, tx, userID, companyID); err != nil {
		return fmt.Errorf("scoring.CompanyTracked: %w", err)
	}
	return nil
}

func (s *Store) ListCompanyBoards(ctx context.Context, companyID string) ([]dto.CompanyBoard, error) {
	id, err := data.UUID(companyID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListCompanyBoards(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("store.ListCompanyBoards: %w", err)
	}
	boards := make([]dto.CompanyBoard, len(rows))
	checks, err := s.queries.ListBoardChecks(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("store.ListCompanyBoards checks: %w", err)
	}
	checkByID := make(map[string]sqlc.ListBoardChecksRow, len(checks))
	for _, check := range checks {
		checkByID[check.ID.String()] = check
	}
	for i, row := range rows {
		boards[i] = toCompanyBoardDTO(row)
		check := checkByID[boards[i].ID]
		boards[i].LastCompletedAt = data.TimePtr(check.LastCompletedAt)
		boards[i].LastReportedTotal = int(check.LastReportedTotal)
		boards[i].LastParsed = int(check.LastParsed)
	}
	return boards, nil
}

func (s *Store) UpsertCandidateBoard(ctx context.Context, companyID, source, token string) (dto.CompanyBoard, error) {
	id, err := data.UUID(companyID)
	if err != nil {
		return dto.CompanyBoard{}, err
	}
	row, err := s.queries.UpsertCandidateBoard(ctx, sqlc.UpsertCandidateBoardParams{
		CompanyID: id, Source: source, BoardToken: token,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.CompanyBoard{}, ErrBoardConflict
	}
	if err != nil {
		return dto.CompanyBoard{}, fmt.Errorf("store.UpsertCandidateBoard: %w", err)
	}
	return toCompanyBoardDTO(row), nil
}

func (s *Store) GetVerifiedBoardID(ctx context.Context, source, token string) (string, error) {
	id, err := s.queries.GetVerifiedBoardID(ctx, sqlc.GetVerifiedBoardIDParams{Source: source, BoardToken: token})
	if err != nil {
		return "", data.QueryErr("GetVerifiedBoardID", err)
	}
	return id.String(), nil
}

func (s *Store) GetBoardCompanyID(ctx context.Context, source, token string) (string, error) {
	id, err := s.queries.GetBoardCompanyID(ctx, sqlc.GetBoardCompanyIDParams{Source: source, BoardToken: token})
	if err != nil {
		return "", data.QueryErr("GetBoardCompanyID", err)
	}
	return id.String(), nil
}

func (s *Store) ListUntrackedDiscoveredBoards(ctx context.Context, uk bool, ukSince time.Time) ([]dto.CompanyBoard, error) {
	rows, err := s.queries.ListUntrackedDiscoveredBoards(ctx, sqlc.ListUntrackedDiscoveredBoardsParams{Uk: uk, UkSince: pgtype.Timestamptz{Time: ukSince, Valid: true}})
	if err != nil {
		return nil, fmt.Errorf("store.ListUntrackedDiscoveredBoards: %w", err)
	}
	boards := make([]dto.CompanyBoard, len(rows))
	for i, row := range rows {
		boards[i] = toCompanyBoardDTO(row)
	}
	return boards, nil
}

func (s *Store) ListVerifiedBoardsBySlug(ctx context.Context, slugs []string) ([]dto.CardBoard, error) {
	rows, err := s.queries.ListVerifiedBoardsBySlug(ctx, slugs)
	if err != nil {
		return nil, data.QueryErr("ListVerifiedBoardsBySlug", err)
	}
	boards := make([]dto.CardBoard, len(rows))
	for i, row := range rows {
		boards[i] = dto.CardBoard{CompanySlug: row.Slug, Source: row.Source, BoardToken: row.BoardToken, Tracked: row.Tracked, Polled: row.Polled}
	}
	return boards, nil
}

func (s *Store) ListVerifiedCompanySlugs(ctx context.Context, slugs []string) ([]string, error) {
	out, err := s.queries.ListVerifiedCompanySlugs(ctx, slugs)
	if err != nil {
		return nil, data.QueryErr("ListVerifiedCompanySlugs", err)
	}
	return out, nil
}

func toSourceTargetDTO(row sqlc.SourceTarget) dto.SourceTarget {
	filters := map[string]string{}
	if len(row.Filters) > 0 {
		_ = json.Unmarshal(row.Filters, &filters)
	}
	t := dto.SourceTarget{
		ID:      row.ID.String(),
		UserID:  row.UserID.String(),
		Source:  row.Source,
		Value:   row.Value,
		Enabled: row.Enabled,
		Filters: filters,
		RunWindow: dto.RunWindow{
			IntervalMinutes: optionalInt32(row.IntervalMinutes),
			Weekdays:        runwindow.Weekdays(row.Weekdays),
			Start:           clockString(row.WindowStart),
			End:             clockString(row.WindowEnd),
			Timezone:        row.Timezone,
		},
		RunStatus:      row.RunStatus,
		LastRunError:   row.LastRunError,
		DisabledReason: row.DisabledReason,
		UpdatedAt:      row.UpdatedAt.Time,
	}
	if row.RunID.Valid {
		t.RunID = row.RunID.String()
	}
	if row.CompanyID.Valid {
		t.CompanyID = row.CompanyID.String()
	}
	t.NextRunAt = data.TimePtr(row.NextRunAt)
	t.LastRunAt = data.TimePtr(row.LastRunAt)
	t.LastSucceededAt = data.TimePtr(row.LastSucceededAt)
	return t
}

func (s *Store) ListSourceTargetsByUser(ctx context.Context, userID string) ([]dto.SourceTarget, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListSourceTargetsByUser(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("store.ListSourceTargetsByUser: %w", err)
	}
	out := make([]dto.SourceTarget, len(rows))
	for i, r := range rows {
		out[i] = toSourceTargetDTO(r)
	}
	return out, nil
}

func (s *Store) CreateSourceTarget(ctx context.Context, userID, source, value string, enabled bool, filters map[string]string, window dto.RunWindow, nextRunAt *time.Time) (dto.SourceTarget, error) {
	return s.createSourceTarget(ctx, "CreateSourceTarget", userID, source, value, enabled, filters, window, nextRunAt, s.queries.CreateSourceTarget)
}

func (s *Store) CreateSourceTargetWithRun(ctx context.Context, userID, source, value string, enabled bool, filters map[string]string, window dto.RunWindow, nextRunAt *time.Time) (dto.SourceTarget, error) {
	return s.createSourceTarget(ctx, "CreateSourceTargetWithRun", userID, source, value, enabled, filters, window, nextRunAt,
		func(ctx context.Context, p sqlc.CreateSourceTargetParams) (sqlc.SourceTarget, error) {
			var row sqlc.SourceTarget
			err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
				queries := s.queries.WithTx(tx)
				var err error
				if row, err = queries.CreateSourceTargetWithRun(ctx, sqlc.CreateSourceTargetWithRunParams(p)); err != nil {
					return err
				}
				return insertScrapeRun(ctx, queries, row, dto.TriggerManual)
			})
			return row, err
		})
}

func (s *Store) createSourceTarget(
	ctx context.Context, op, userID, source, value string, enabled bool, filters map[string]string,
	window dto.RunWindow, nextRunAt *time.Time,
	insert func(context.Context, sqlc.CreateSourceTargetParams) (sqlc.SourceTarget, error),
) (dto.SourceTarget, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	filtersJSON, err := json.Marshal(filters)
	if err != nil {
		return dto.SourceTarget{}, fmt.Errorf("store.%s: marshal filters: %w", op, err)
	}
	args := windowArgs(window, nextRunAt)
	row, err := insert(ctx, sqlc.CreateSourceTargetParams{
		UserID: uid, Source: source, Value: value, Enabled: enabled, Filters: filtersJSON,
		IntervalMinutes: args.interval, Weekdays: args.weekdays, WindowStart: args.start,
		WindowEnd: args.end, Timezone: args.timezone, NextRunAt: args.nextRunAt,
	})
	if err != nil {
		if data.IsUniqueViolation(err) {
			return dto.SourceTarget{}, ErrSourceTargetExists
		}
		return dto.SourceTarget{}, fmt.Errorf("store.%s: %w", op, err)
	}
	return toSourceTargetDTO(row), nil
}

func (s *Store) UpdateSourceTarget(ctx context.Context, id, userID string, enabled *bool, window *dto.RunWindow, nextRunAt *time.Time) (dto.SourceTarget, error) {
	tid, err := data.UUID(id)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	uid, err := data.UUID(userID)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	params := sqlc.UpdateSourceTargetParams{ID: tid, UserID: uid}
	if enabled != nil {
		params.Enabled = pgtype.Bool{Bool: *enabled, Valid: true}
	}
	if window != nil {
		args := windowArgs(*window, nextRunAt)
		params.SetWindow = true
		params.IntervalMinutes = args.interval
		params.Weekdays = args.weekdays
		params.WindowStart = args.start
		params.WindowEnd = args.end
		params.Timezone = args.timezone
		params.NextRunAt = args.nextRunAt
	}
	row, err := s.queries.UpdateSourceTarget(ctx, params)
	if err != nil {
		return dto.SourceTarget{}, data.QueryErr("UpdateSourceTarget", err)
	}
	return toSourceTargetDTO(row), nil
}

func (s *Store) DeleteSourceTarget(ctx context.Context, id, userID string) error {
	tid, err := data.UUID(id)
	if err != nil {
		return err
	}
	uid, err := data.UUID(userID)
	if err != nil {
		return err
	}
	if err := s.queries.DeleteSourceTarget(ctx, sqlc.DeleteSourceTargetParams{ID: tid, UserID: uid}); err != nil {
		return fmt.Errorf("store.DeleteSourceTarget: %w", err)
	}
	return nil
}

func (s *Store) StartSourceTargetRun(ctx context.Context, id string, nextRunAt *time.Time) (dto.SourceTarget, error) {
	tid, err := data.UUID(id)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	var row sqlc.SourceTarget
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		queries := s.queries.WithTx(tx)
		var err error
		if row, err = queries.StartSourceTargetRun(ctx, sqlc.StartSourceTargetRunParams{ID: tid, NextRunAt: timestamptz(nextRunAt)}); err != nil {
			return err
		}
		return insertScrapeRun(ctx, queries, row, dto.TriggerManual)
	})
	if err != nil {
		return dto.SourceTarget{}, data.QueryErr("StartSourceTargetRun", err)
	}
	return toSourceTargetDTO(row), nil
}

// ClaimDueSourceTargets starts a queued run on up to limit due Source Targets,
// moving each to the next_run_at that nextRunAt returns for its window. Rows
// another claim holds are skipped.
func (s *Store) ClaimDueSourceTargets(ctx context.Context, limit int, nextRunAt func(dto.RunWindow) *time.Time) ([]dto.SourceTarget, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin claim due source targets: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := s.queries.WithTx(tx)

	due, err := queries.LockDueSourceTargets(ctx, int32(limit))
	if err != nil {
		return nil, data.QueryErr("LockDueSourceTargets", err)
	}
	claimed := make([]dto.SourceTarget, 0, len(due))
	for _, row := range due {
		next := nextRunAt(toSourceTargetDTO(row).RunWindow)
		started, err := queries.StartSourceTargetRun(ctx, sqlc.StartSourceTargetRunParams{ID: row.ID, NextRunAt: timestamptz(next)})
		if err != nil {
			return nil, data.QueryErr("StartSourceTargetRun", err)
		}
		if err := insertScrapeRun(ctx, queries, started, dto.TriggerScheduled); err != nil {
			return nil, err
		}
		claimed = append(claimed, toSourceTargetDTO(started))
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit claim due source targets: %w", err)
	}
	return claimed, nil
}

func (s *Store) GetSourceTarget(ctx context.Context, id string) (dto.SourceTarget, error) {
	tid, err := data.UUID(id)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	row, err := s.queries.GetSourceTarget(ctx, tid)
	if err != nil {
		return dto.SourceTarget{}, data.QueryErr("GetSourceTarget", err)
	}
	return toSourceTargetDTO(row), nil
}

func (s *Store) TransitionSourceTargetRun(ctx context.Context, id, runID, status, runError string) (dto.SourceTarget, error) {
	tid, err := data.UUID(id)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	rid, err := data.UUID(runID)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	row, err := s.queries.TransitionSourceTargetRun(ctx, sqlc.TransitionSourceTargetRunParams{
		ID: tid, RunID: rid, RunStatus: status, LastRunError: runError,
	})
	if err != nil {
		return dto.SourceTarget{}, data.QueryErr("TransitionSourceTargetRun", err)
	}
	return toSourceTargetDTO(row), nil
}

func (s *Store) DisableSourceTargets(ctx context.Context, source, reason string) (int64, error) {
	n, err := s.queries.DisableSourceTargets(ctx, sqlc.DisableSourceTargetsParams{Source: source, DisabledReason: reason})
	if err != nil {
		return 0, data.QueryErr("DisableSourceTargets", err)
	}
	return n, nil
}

func (s *Store) ListRecoverableSourceTargets(ctx context.Context) ([]dto.SourceTarget, error) {
	rows, err := s.queries.ListRecoverableSourceTargets(ctx)
	if err != nil {
		return nil, fmt.Errorf("store.ListRecoverableSourceTargets: %w", err)
	}
	out := make([]dto.SourceTarget, len(rows))
	for i, r := range rows {
		out[i] = toSourceTargetDTO(r)
	}
	return out, nil
}

func (s *Store) ClaimRecoverableSourceTarget(ctx context.Context, id, runID string) (dto.SourceTarget, error) {
	tid, err := data.UUID(id)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	rid, err := data.UUID(runID)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	row, err := s.queries.ClaimRecoverableSourceTarget(ctx, sqlc.ClaimRecoverableSourceTargetParams{ID: tid, RunID: rid})
	if err != nil {
		return dto.SourceTarget{}, data.QueryErr("ClaimRecoverableSourceTarget", err)
	}
	return toSourceTargetDTO(row), nil
}

// NormalizeCandidateURL is the rule behind job_candidates.normalized_url.
func NormalizeCandidateURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("invalid candidate URL %q", raw)
	}
	u.Host = strings.ToLower(u.Host)
	if strings.HasSuffix(u.Host, ".linkedin.com") {
		u.Host = "www.linkedin.com"
	}
	u.Fragment = ""
	u.RawQuery = StripTrackingParams(u.RawQuery)
	u.Path = strings.TrimSuffix(u.Path, "/")
	return u.String(), nil
}

// NormalizeOrRaw normalises raw, or returns it unchanged when it is not a valid URL.
func NormalizeOrRaw(raw string) string {
	n, err := NormalizeCandidateURL(raw)
	if err != nil {
		return raw
	}
	return n
}

func (s *Store) SaveCards(ctx context.Context, target dto.SourceTarget, cards []dto.Job) ([]sourcetargets.Candidate, error) {
	targetID, err := data.UUID(target.ID)
	if err != nil {
		return nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin candidate save: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := s.queries.WithTx(tx)

	out := make([]sourcetargets.Candidate, 0, len(cards))
	var tally cardTally
	for _, card := range cards {
		if card.URL == "" {
			continue
		}
		normalized, err := NormalizeCandidateURL(card.URL)
		if err != nil {
			return nil, err
		}
		card.URL = normalized
		card.Source = target.Source
		kind, err := queries.ClassifyScrapeCard(ctx, sqlc.ClassifyScrapeCardParams{NormalizedUrl: normalized, SourceTargetID: targetID})
		if err != nil {
			return nil, fmt.Errorf("classify scrape card: %w", err)
		}
		tally.add(kind)
		var payload []byte
		if sourcespec.CardComplete(target.Source) || card.RemoteCheckURL != "" || card.WorkArrangement != "" {
			if payload, err = json.Marshal(card); err != nil {
				return nil, fmt.Errorf("marshal candidate card: %w", err)
			}
		}
		id, err := queries.UpsertCandidate(ctx, sqlc.UpsertCandidateParams{
			NormalizedUrl: normalized, Source: target.Source,
			CardTitle: card.Title, CardCompany: card.CompanySlug, CardLocation: card.Location,
			Card: payload, PostedAt: data.Timestamptz(card.PostedAt),
		})
		if err != nil {
			return nil, fmt.Errorf("upsert candidate: %w", err)
		}
		if err := applySighting(ctx, queries, normalized, card.PostedAt); err != nil {
			return nil, err
		}
		if err := queries.RecordCandidateDiscovery(ctx, sqlc.RecordCandidateDiscoveryParams{
			CandidateID: id, SourceTargetID: targetID,
		}); err != nil {
			return nil, fmt.Errorf("record candidate discovery: %w", err)
		}
		out = append(out, sourcetargets.Candidate{ID: id.String(), URL: normalized, Card: card})
	}
	if target.RunID != "" {
		runID, err := data.UUID(target.RunID)
		if err != nil {
			return nil, err
		}
		if err := queries.AddScrapeRunCards(ctx, sqlc.AddScrapeRunCardsParams{
			ID: runID, Listings: tally.total, NewListings: tally.fresh, Elsewhere: tally.elsewhere, RepeatListings: tally.repeat,
		}); err != nil {
			return nil, fmt.Errorf("add scrape run cards: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit candidate save: %w", err)
	}
	return out, nil
}

func applySighting(ctx context.Context, queries *sqlc.Queries, normalizedURL string, postedAt *time.Time) error {
	if postedAt == nil {
		return nil
	}
	jobID, err := queries.FindJobIDByURL(ctx, normalizedURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("find job by url: %w", err)
	}
	if err := queries.ApplyPostedDate(ctx, sqlc.ApplyPostedDateParams{PostedAt: data.Timestamptz(postedAt), ID: jobID}); err != nil {
		return fmt.Errorf("apply sighting date: %w", err)
	}
	return nil
}

func (s *Store) ListForUser(ctx context.Context, userID, afterID string, limit int) ([]sourcetargets.Candidate, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		return nil, fmt.Errorf("candidate batch limit out of range: %d", limit)
	}
	if afterID == "" {
		afterID = "00000000-0000-0000-0000-000000000000"
	}
	after, err := data.UUID(afterID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListCandidatesForUser(ctx, sqlc.ListCandidatesForUserParams{
		UserID: uid, ID: after, Limit: int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("query candidates: %w", err)
	}
	out := make([]sourcetargets.Candidate, 0, len(rows))
	for _, row := range rows {
		card := dto.Job{URL: row.NormalizedUrl, Title: row.CardTitle, CompanySlug: row.CardCompany,
			Location: row.CardLocation, Source: row.Source, PostedAt: data.TimePtr(row.PostedAt)}
		if row.Card != nil {
			if err := json.Unmarshal(row.Card, &card); err != nil {
				return nil, fmt.Errorf("unmarshal candidate card %s: %w", row.ID, err)
			}
		}
		out = append(out, sourcetargets.Candidate{ID: row.ID.String(), URL: row.NormalizedUrl, Card: card})
	}
	return out, nil
}

func (s *Store) DiscoveryFilters(ctx context.Context, userID string, candidateIDs []string) (map[string][]map[string]string, error) {
	uid, err := data.UUID(userID)
	if err != nil {
		return nil, err
	}
	ids := make([]pgtype.UUID, len(candidateIDs))
	for i, id := range candidateIDs {
		if ids[i], err = data.UUID(id); err != nil {
			return nil, err
		}
	}
	rows, err := s.queries.ListCandidateDiscoveryFilters(ctx, sqlc.ListCandidateDiscoveryFiltersParams{CandidateIds: ids, UserID: uid})
	if err != nil {
		return nil, fmt.Errorf("query candidate discovery filters: %w", err)
	}
	out := make(map[string][]map[string]string, len(candidateIDs))
	for _, row := range rows {
		filters := map[string]string{}
		if len(row.Filters) > 0 {
			if err := json.Unmarshal(row.Filters, &filters); err != nil {
				return nil, fmt.Errorf("unmarshal target filters: %w", err)
			}
		}
		id := row.CandidateID.String()
		out[id] = append(out[id], filters)
	}
	return out, nil
}

func (s *Store) Assess(ctx context.Context, candidateID, userID string, version time.Time, passes bool) (bool, error) {
	cid, err := data.UUID(candidateID)
	if err != nil {
		return false, err
	}
	uid, err := data.UUID(userID)
	if err != nil {
		return false, err
	}
	claimed, err := s.queries.AssessCandidate(ctx, sqlc.AssessCandidateParams{
		CandidateID: cid, UserID: uid,
		SearchConfigVersion: pgtype.Timestamptz{Time: version, Valid: true},
		Relevance:           passes,
	})
	if err != nil {
		return false, fmt.Errorf("assess candidate: %w", err)
	}
	return claimed, nil
}

func (s *Store) MarkDetailPending(ctx context.Context, candidateID string) error {
	cid, err := data.UUID(candidateID)
	if err != nil {
		return err
	}
	return s.queries.MarkCandidateDetailPending(ctx, cid)
}

func (s *Store) DeleteExpiredCandidates(ctx context.Context) error {
	return s.queries.DeleteExpiredCandidates(ctx)
}

func (s *Store) GetLastScraped(ctx context.Context, source string) (time.Time, bool, error) {
	last, err := s.queries.GetHarvestRun(ctx, source)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, fmt.Errorf("store.GetLastScraped: %w", err)
	}
	return last.Time, true, nil
}

func (s *Store) SetLastScraped(ctx context.Context, source string) error {
	if err := s.queries.SetHarvestRun(ctx, source); err != nil {
		return fmt.Errorf("store.SetLastScraped: %w", err)
	}
	return nil
}

func (s *Store) NewURLs(ctx context.Context, urls []string) ([]string, error) {
	normalized := make([]string, len(urls))
	for i, u := range urls {
		normalized[i] = NormalizeOrRaw(u)
	}
	existing, err := s.queries.ExistingURLs(ctx, append(slices.Clone(urls), normalized...))
	if err != nil {
		return nil, fmt.Errorf("store.NewURLs: %w", err)
	}
	known := make(map[string]struct{}, len(existing))
	for _, u := range existing {
		known[u] = struct{}{}
	}
	out := make([]string, 0, len(urls))
	for i, u := range urls {
		_, rawKnown := known[u]
		_, normalizedKnown := known[normalized[i]]
		if !rawKnown && !normalizedKnown {
			out = append(out, u)
		}
	}
	return out, nil
}

func (s *Store) ListCompaniesToCrawl(ctx context.Context, limit int) ([]dto.Company, error) {
	rows, err := s.queries.ListCompaniesToCrawl(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("store.ListCompaniesToCrawl: %w", err)
	}
	out := make([]dto.Company, len(rows))
	for i, r := range rows {
		out[i] = toCompanyDTO(r)
	}
	return out, nil
}

func (s *Store) TouchCompanyCrawled(ctx context.Context, id string) error {
	cid, err := data.UUID(id)
	if err != nil {
		return err
	}
	if err := s.queries.TouchCompanyCrawled(ctx, cid); err != nil {
		return fmt.Errorf("store.TouchCompanyCrawled: %w", err)
	}
	return nil
}

func (s *Store) RenameCompany(ctx context.Context, id, name string) error {
	cid, err := data.UUID(id)
	if err != nil {
		return err
	}
	if err := s.queries.RenameCompany(ctx, sqlc.RenameCompanyParams{ID: cid, Name: name}); err != nil {
		return fmt.Errorf("store.RenameCompany: %w", err)
	}
	return nil
}

func (s *Store) MarkBoardUKJobsSeen(ctx context.Context, source, token string) error {
	if err := s.queries.MarkBoardUKJobsSeen(ctx, sqlc.MarkBoardUKJobsSeenParams{Source: source, BoardToken: token}); err != nil {
		return fmt.Errorf("store.MarkBoardUKJobsSeen: %w", err)
	}
	return nil
}

func (s *Store) VerifyCompanyBoard(ctx context.Context, companyID, source, token, method, via string) (dto.CompanyBoard, error) {
	id, err := data.UUID(companyID)
	if err != nil {
		return dto.CompanyBoard{}, err
	}
	row, err := s.queries.VerifyCompanyBoard(ctx, sqlc.VerifyCompanyBoardParams{
		CompanyID: id, Source: source, BoardToken: token,
		VerificationMethod: pgtype.Text{String: method, Valid: true}, Via: via,
	})
	if err != nil {
		return dto.CompanyBoard{}, data.QueryErr("VerifyCompanyBoard", err)
	}
	return toCompanyBoardDTO(row), nil
}

func toBoardPollDTO(id, companyID pgtype.UUID, companySlug, source, token string, intervalMinutes int32) dto.BoardPoll {
	return dto.BoardPoll{
		ID: id.String(), CompanyID: companyID.String(), CompanySlug: companySlug,
		Source: source, Token: token, IntervalMinutes: int(intervalMinutes),
	}
}

func (s *Store) ListDueBoards(ctx context.Context) ([]dto.BoardPoll, error) {
	rows, err := s.queries.ListDueBoards(ctx)
	if err != nil {
		return nil, fmt.Errorf("store.ListDueBoards: %w", err)
	}
	boards := make([]dto.BoardPoll, len(rows))
	for i, row := range rows {
		boards[i] = toBoardPollDTO(row.ID, row.CompanyID, row.CompanySlug, row.Source, row.BoardToken, row.IntervalMinutes)
	}
	return boards, nil
}

func (s *Store) MarkBoardCheckPublished(ctx context.Context, id string) error {
	boardID, err := data.UUID(id)
	if err != nil {
		return err
	}
	if err := s.queries.MarkBoardCheckPublished(ctx, boardID); err != nil {
		return fmt.Errorf("store.MarkBoardCheckPublished: %w", err)
	}
	return nil
}

func (s *Store) ClearBoardCheckPublished(ctx context.Context, id string) error {
	boardID, err := data.UUID(id)
	if err != nil {
		return err
	}
	if err := s.queries.ClearBoardCheckPublished(ctx, boardID); err != nil {
		return fmt.Errorf("store.ClearBoardCheckPublished: %w", err)
	}
	return nil
}

func (s *Store) ListActiveBoards(ctx context.Context) ([]dto.BoardPoll, error) {
	rows, err := s.queries.ListActiveBoards(ctx)
	if err != nil {
		return nil, fmt.Errorf("store.ListActiveBoards: %w", err)
	}
	boards := make([]dto.BoardPoll, len(rows))
	for i, row := range rows {
		boards[i] = toBoardPollDTO(row.ID, row.CompanyID, row.CompanySlug, row.Source, row.BoardToken, row.IntervalMinutes)
	}
	return boards, nil
}

func (s *Store) ClaimBoard(ctx context.Context, id string, manual bool) (dto.BoardPoll, error) {
	boardID, err := data.UUID(id)
	if err != nil {
		return dto.BoardPoll{}, err
	}
	board, err := s.queries.GetPollBoard(ctx, boardID)
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.BoardPoll{}, ErrBoardClaimUnavailable
	}
	if err != nil {
		return dto.BoardPoll{}, fmt.Errorf("store.ClaimBoard: get poll board: %w", err)
	}
	if !manual && board.LastScheduledAt.Valid && time.Since(board.LastScheduledAt.Time) < time.Duration(board.IntervalMinutes)*time.Minute {
		return dto.BoardPoll{}, ErrBoardClaimUnavailable
	}
	if err := s.queries.EnsureBoardPollState(ctx, boardID); err != nil {
		return dto.BoardPoll{}, fmt.Errorf("store.ClaimBoard: ensure poll state: %w", err)
	}
	owner := uuid.NewString()
	claim, err := s.queries.ClaimPollState(ctx, sqlc.ClaimPollStateParams{BoardID: boardID, LeaseOwner: pgtype.Text{String: owner, Valid: true}, Manual: manual})
	if errors.Is(err, pgx.ErrNoRows) {
		return dto.BoardPoll{}, ErrBoardClaimUnavailable
	}
	if err != nil {
		return dto.BoardPoll{}, fmt.Errorf("store.ClaimBoard: claim poll state: %w", err)
	}
	poll := toBoardPollDTO(boardID, board.CompanyID, board.CompanySlug, board.Source, board.BoardToken, board.IntervalMinutes)
	poll.LeaseOwner, poll.Version, poll.StartedAt, poll.Manual = owner, claim.LastSnapshotVersion, claim.LastStartedAt.Time, manual
	return poll, nil
}

func (s *Store) FailBoard(ctx context.Context, poll dto.BoardPoll) error {
	id, err := data.UUID(poll.ID)
	if err != nil {
		return err
	}
	n, err := s.queries.FailPollState(ctx, sqlc.FailPollStateParams{BoardID: id, LeaseOwner: pgtype.Text{String: poll.LeaseOwner, Valid: true}, LastSnapshotVersion: poll.Version})
	if err != nil {
		return fmt.Errorf("store.FailBoard: %w", err)
	}
	if n != 1 {
		return ErrBoardClaimUnavailable
	}
	return nil
}

func (s *Store) CompleteBoard(ctx context.Context, snapshot dto.BoardSnapshot) error {
	if !snapshot.Complete {
		return errors.New("incomplete board snapshot")
	}
	poll := snapshot.Poll
	id, err := data.UUID(poll.ID)
	if err != nil {
		return err
	}
	urls := make([]string, 0, len(snapshot.Jobs))
	seenURLs := make(map[string]bool, len(snapshot.Jobs))
	for _, job := range snapshot.Jobs {
		url, err := normalizeJobURL(job.URL)
		if err != nil {
			return err
		}
		if !seenURLs[url] {
			urls = append(urls, url)
			seenURLs[url] = true
		}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin board completion: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.queries.WithTx(tx)
	state, err := q.LockPollState(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrBoardClaimUnavailable
	}
	if err != nil {
		return fmt.Errorf("lock board state: %w", err)
	}
	if state.LastSnapshotVersion != poll.Version || !state.LeaseOwner.Valid || state.LeaseOwner.String != poll.LeaseOwner || !state.LeaseUntil.Valid || !state.LeaseUntil.Time.After(time.Now()) {
		return ErrBoardClaimUnavailable
	}
	aliases, err := q.FindBoardJobAliases(ctx, urls)
	if err != nil {
		return fmt.Errorf("resolve board jobs: %w", err)
	}
	if len(aliases) != len(urls) {
		return errors.New("board ingest incomplete: job URL missing")
	}
	seenJobs := make(map[pgtype.UUID]bool, len(aliases))
	for _, alias := range aliases {
		if seenJobs[alias.JobID] {
			continue
		}
		seenJobs[alias.JobID] = true
		if err := q.ObserveBoardJob(ctx, sqlc.ObserveBoardJobParams{BoardID: id, JobID: alias.JobID, LastSnapshotVersion: poll.Version}); err != nil {
			return fmt.Errorf("observe board job: %w", err)
		}
	}
	if err := q.ReopenObservedBoardJobs(ctx, sqlc.ReopenObservedBoardJobsParams{BoardID: id, LastSnapshotVersion: poll.Version}); err != nil {
		return fmt.Errorf("reopen board jobs: %w", err)
	}
	if len(urls) > 0 || state.ConsecutiveCompleteEmpty >= 1 {
		if err := q.CloseMissingBoardJobs(ctx, sqlc.CloseMissingBoardJobsParams{PrimaryBoardID: id, LastSnapshotVersion: poll.Version}); err != nil {
			return fmt.Errorf("close missing board jobs: %w", err)
		}
	}
	if len(urls) == 0 && state.ConsecutiveCompleteEmpty >= 1 {
		if err := q.RetireSupersededBoard(ctx, id); err != nil {
			return fmt.Errorf("retire board: %w", err)
		}
	}
	n, err := q.CompletePollState(ctx, sqlc.CompletePollStateParams{Manual: poll.Manual, IntervalMinutes: pollIntervalMinutes(poll, snapshot.NextPollIn), Empty: len(urls) == 0, Reported: int32(snapshot.Reported), Parsed: int32(len(urls)), BoardID: id, LeaseOwner: poll.LeaseOwner, Version: poll.Version})
	if err != nil {
		return fmt.Errorf("complete board state: %w", err)
	}
	if n != 1 {
		return ErrBoardClaimUnavailable
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit board completion: %w", err)
	}
	return nil
}

func pollIntervalMinutes(poll dto.BoardPoll, hint time.Duration) int32 {
	if hint > 0 {
		return int32(max(1, int(hint/time.Minute)))
	}
	return int32(poll.IntervalMinutes)
}

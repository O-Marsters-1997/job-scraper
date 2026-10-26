package jobsearchtest

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/jobmatch"
	"github.com/ollymarsters/job-scraper/internal/runwindow"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
	"github.com/ollymarsters/job-scraper/internal/services/sourcetargets"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
)

type boardPollState struct {
	leaseOwner       string
	leaseExpires     time.Time
	version          int64
	reported         int
	parsed           int
	completedAt      time.Time
	checkPublishedAt time.Time
}

type FakeStore struct {
	mu sync.Mutex

	seq int

	jobs     map[string]dto.Job
	seen     map[string]map[string]bool
	jobOrder []string
	byURL    map[string]string
	byBoard  map[string]string
	listings map[string][]dto.JobListing

	companies     map[string]dto.Company
	companyBySlug map[string]string
	tracking      map[string]dto.CompanyTracking
	favourites    map[string]bool

	boards    map[string]dto.CompanyBoard
	pollState map[string]*boardPollState

	lastScraped map[string]time.Time
	fetches     map[string]dto.CachedResponse
	misses      map[string]time.Time
	harvestSeen map[string]map[string]string
	profiles    map[string]dto.CompanyProfile

	sourceTargets map[string]dto.SourceTarget
	targetByKey   map[string]string

	candidates     map[string]sourcetargets.Candidate
	candidateByURL map[string]string
	discoveries    map[string]map[string]discovery
	detailPending  map[string]bool
	datedCards     map[string]dto.Job

	insightGroups []dto.JobInsightGroup
}

func (f *FakeStore) SeedInsightGroups(groups ...dto.JobInsightGroup) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.insightGroups = groups
}

func (f *FakeStore) JobInsightGroups(context.Context, string, dto.JobInsightsOptions) ([]dto.JobInsightGroup, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.insightGroups), nil
}

func NewFakeStore() *FakeStore {
	return &FakeStore{
		jobs:           make(map[string]dto.Job),
		seen:           make(map[string]map[string]bool),
		byURL:          make(map[string]string),
		byBoard:        make(map[string]string),
		listings:       make(map[string][]dto.JobListing),
		companies:      make(map[string]dto.Company),
		companyBySlug:  make(map[string]string),
		tracking:       make(map[string]dto.CompanyTracking),
		favourites:     make(map[string]bool),
		boards:         make(map[string]dto.CompanyBoard),
		pollState:      make(map[string]*boardPollState),
		lastScraped:    make(map[string]time.Time),
		fetches:        make(map[string]dto.CachedResponse),
		misses:         make(map[string]time.Time),
		harvestSeen:    make(map[string]map[string]string),
		profiles:       make(map[string]dto.CompanyProfile),
		sourceTargets:  make(map[string]dto.SourceTarget),
		targetByKey:    make(map[string]string),
		candidates:     make(map[string]sourcetargets.Candidate),
		candidateByURL: make(map[string]string),
		discoveries:    make(map[string]map[string]discovery),
		detailPending:  make(map[string]bool),
		datedCards:     make(map[string]dto.Job),
	}
}

func (f *FakeStore) nextID() string {
	f.seq++
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", f.seq)
}

func lookup[T any](m map[string]T, id string) (T, error) {
	v, ok := m[id]
	if !ok {
		var zero T
		return zero, data.ErrNotFound
	}
	return v, nil
}

func trackingKey(userID, companyID string) string { return userID + "|" + companyID }

func targetKey(userID, source, value string) string { return userID + "|" + source + "|" + value }

func jobFingerprint(job dto.Job) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s", job.Title, job.Description, job.Location, job.SalaryRaw, job.WorkArrangement)
}

func normalizeJobURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	u.RawQuery = store.StripTrackingParams(u.RawQuery)
	return u.String()
}

func (f *FakeStore) jobIdentity(job dto.Job) (string, bool) {
	if job.BoardID != "" && job.ProviderPostingID != "" {
		if id, ok := f.byBoard[job.BoardID+"|"+job.ProviderPostingID]; ok {
			return id, true
		}
	}
	if id, ok := f.byURL[job.URL]; ok {
		return id, true
	}
	return "", false
}

func (f *FakeStore) indexJob(job dto.Job) {
	f.byURL[job.URL] = job.ID
	if job.BoardID != "" && job.ProviderPostingID != "" {
		f.byBoard[job.BoardID+"|"+job.ProviderPostingID] = job.ID
	}
}

func (f *FakeStore) addListing(jobID string, job dto.Job) {
	f.byURL[job.URL] = jobID
	if slices.ContainsFunc(f.listings[jobID], func(l dto.JobListing) bool { return l.URL == job.URL }) {
		return
	}
	f.listings[jobID] = append(f.listings[jobID], dto.JobListing{Source: job.Source, URL: job.URL, FirstSeenAt: time.Now()})
}

func (f *FakeStore) addDiscoveryListing(jobID string, job dto.Job) {
	if job.DiscoveredVia == "" || job.DiscoveryURL == "" {
		return
	}
	f.addListing(jobID, dto.Job{Source: job.DiscoveredVia, URL: normalizeJobURL(job.DiscoveryURL)})
}

func (f *FakeStore) SaveCanonical(_ context.Context, job dto.Job) (dto.Job, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	job.URL = normalizeJobURL(job.URL)
	fingerprint := jobFingerprint(job)
	id, ok := f.jobIdentity(job)
	if !ok {
		id = f.nextID()
		for _, card := range f.datedCards {
			if cardMatchesJob(card, job) && (job.PostedAt == nil || card.PostedAt.Before(*job.PostedAt)) {
				job.PostedAt = card.PostedAt
			}
		}
		job.ID, job.ContentFingerprint, job.FirstDiscoveredAt = id, fingerprint, time.Now()
		if st := f.pollState[job.BoardID]; st != nil && !st.completedAt.IsZero() {
			completedAt := st.completedAt
			job.PostedAfter = &completedAt
		}
		f.jobs[id] = job
		f.jobOrder = append(f.jobOrder, id)
		f.indexJob(job)
		f.addListing(id, job)
		f.addDiscoveryListing(id, job)
		return job, "new", nil
	}

	prev := f.jobs[id]
	if prev.BoardID != "" && job.BoardID != "" && prev.BoardID != job.BoardID {
		return dto.Job{}, "", store.ErrCanonicalConflict
	}

	status := "unchanged"
	if prev.ContentFingerprint != fingerprint {
		status = "changed"
	}
	f.addListing(id, job)
	f.addDiscoveryListing(id, job)
	job.DiscoveredVia = prev.DiscoveredVia
	job.PostedAt, job.RepostedAt = applyPostedAt(prev, job.PostedAt)
	job.PostedAfter, job.FirstDiscoveredAt = prev.PostedAfter, prev.FirstDiscoveredAt
	job.ID, job.ContentFingerprint, job.URL = id, fingerprint, prev.URL
	f.jobs[id] = job
	f.indexJob(job)
	return job, status, nil
}

func applyPostedAt(prev dto.Job, incoming *time.Time) (postedAt, repostedAt *time.Time) {
	postedAt, repostedAt = prev.PostedAt, prev.RepostedAt
	switch {
	case incoming == nil:
	case postedAt == nil || incoming.Before(*postedAt):
		postedAt = incoming
	case incoming.After(postedAt.Add(2 * 24 * time.Hour)):
		if repostedAt == nil || incoming.After(*repostedAt) {
			repostedAt = incoming
		}
	}
	return postedAt, repostedAt
}

func (f *FakeStore) withUserState(userID string, job dto.Job) dto.Job {
	job.Seen = f.seen[userID][job.ID]
	companyID := cmp.Or(job.CompanyID, f.companyBySlug[job.CompanySlug])
	job.CompanyFavourite = f.favourites[trackingKey(userID, companyID)]
	job.CompanyKind = cmp.Or(f.companies[companyID].Kind, dto.KindUnclassified)
	return job
}

func (f *FakeStore) JobStats(_ context.Context, _ string, options dto.JobStatsOptions) (dto.JobStats, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var stats dto.JobStats
	sources, todaySources := map[string]bool{}, map[string]bool{}
	for _, job := range f.jobs {
		if job.SuitabilityScore == nil || slices.Contains(options.ExcludedCompanySlugs, job.CompanySlug) {
			continue
		}
		stats.Total++
		sources[job.Source] = true
		if !job.ScrapedAt.Before(options.DayStart) {
			stats.NewToday++
			todaySources[job.Source] = true
		}
	}
	stats.Sources, stats.TodaySources = len(sources), len(todaySources)
	return stats, nil
}

func (f *FakeStore) Page(_ context.Context, userID string, options dto.JobPageOptions) (dto.JobPage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	items := make([]dto.Job, 0, len(f.jobOrder))
	for _, id := range f.jobOrder {
		job := f.jobs[id]
		if slices.Contains(options.ExcludedCompanySlugs, job.CompanySlug) {
			continue
		}
		items = append(items, f.withUserState(userID, job))
	}
	total := len(items)
	items = items[min(int(options.Offset), total):]
	if limit := int(options.Limit); limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	page := dto.JobPage{Items: items}
	if !options.SkipTotal {
		page.Total = &total
	}
	return page, nil
}

func (f *FakeStore) GetJob(_ context.Context, jobID, userID string) (dto.Job, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	job, err := lookup(f.jobs, jobID)
	if err != nil {
		return dto.Job{}, err
	}
	return f.withUserState(userID, job), nil
}

func (f *FakeStore) FindJobByPosting(_ context.Context, rawURL, source, token, postingID string) (dto.Job, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if postingID != "" {
		for _, b := range f.boards {
			if b.Source == source && b.BoardToken == token {
				if id, ok := f.byBoard[b.ID+"|"+postingID]; ok {
					return f.jobs[id], nil
				}
			}
		}
	}
	if id, ok := f.byURL[normalizeJobURL(rawURL)]; ok {
		return f.jobs[id], nil
	}
	return dto.Job{}, data.ErrNotFound
}

func (f *FakeStore) ListJobListings(_ context.Context, jobID string) ([]dto.JobListing, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.listings[jobID]), nil
}

func (f *FakeStore) MarkJobsSeen(_ context.Context, userID string, jobIDs []string, seen bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.seen[userID] == nil {
		f.seen[userID] = make(map[string]bool)
	}
	for _, id := range jobIDs {
		if _, ok := f.jobs[id]; !ok {
			continue
		}
		if seen {
			f.seen[userID][id] = true
		} else {
			delete(f.seen[userID], id)
		}
	}
	return nil
}

func (f *FakeStore) NewURLs(_ context.Context, urls []string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(urls))
	for _, u := range urls {
		n := store.NormalizeOrRaw(u)
		known := false
		for _, k := range []string{u, n} {
			_, job := f.byURL[k]
			_, candidate := f.candidateByURL[k]
			known = known || job || candidate
		}
		if !known {
			out = append(out, u)
		}
	}
	return out, nil
}

func (f *FakeStore) UpsertCompany(_ context.Context, c dto.CompanyUpsert) (dto.Company, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if id, ok := f.companyBySlug[c.Slug]; ok {
		existing := f.companies[id]
		if c.Kind != "" && existing.Kind != dto.KindAgency {
			existing.Kind = c.Kind
		}
		if existing.ATSSource == "" {
			existing.ATSSource = c.ATSSource
		}
		if existing.ATSToken == "" {
			existing.ATSToken = c.ATSToken
		}
		if existing.Domain == "" {
			existing.Domain = c.Domain
		}
		if existing.LinkedInCompanyID == "" {
			existing.LinkedInCompanyID = c.LinkedInCompanyID
		}
		f.companies[id] = existing
		return existing, nil
	}
	id := f.nextID()
	company := dto.Company{
		ID: id, Slug: c.Slug, Name: c.Name, ATSSource: c.ATSSource, ATSToken: c.ATSToken,
		Domain: c.Domain, LinkedInCompanyID: c.LinkedInCompanyID, FirstSeenAt: time.Now(),
		Kind: dto.KindUnclassified,
	}
	if c.Kind != "" {
		company.Kind = c.Kind
	}
	f.companies[id] = company
	f.companyBySlug[c.Slug] = id
	return company, nil
}

func (f *FakeStore) GetCompany(_ context.Context, id string) (dto.Company, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return lookup(f.companies, id)
}

func (f *FakeStore) PageCompaniesForUser(_ context.Context, userID string, options dto.CompanyPageOptions) (dto.CompanyPage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	search := strings.ToLower(options.Search)
	var items []dto.Company
	for _, c := range f.companies {
		if tr, ok := f.tracking[trackingKey(userID, c.ID)]; ok {
			c.Tracked, c.ReviewState, c.CheckIntervalMinutes = tr.Enabled, tr.ReviewState, tr.CheckIntervalMinutes
		}
		c.Favourite = f.favourites[trackingKey(userID, c.ID)]
		if options.TrackedOnly && !c.Tracked {
			continue
		}
		if options.FavouriteOnly && !c.Favourite {
			continue
		}
		if options.NoBoardOnly && f.hasBoard(c.ID) {
			continue
		}
		if len(options.ATSSources) > 0 && !f.hasVerifiedBoardOn(c.ID, options.ATSSources) {
			continue
		}
		if options.Poster != "" && (options.Poster == dto.PosterAgency) != (c.Kind == dto.KindAgency || c.Kind == dto.KindLikelyAgency) {
			continue
		}
		if !strings.Contains(strings.ToLower(c.Name), search) && !strings.Contains(c.Slug, search) {
			continue
		}
		items = append(items, c)
	}
	slices.SortFunc(items, func(a, b dto.Company) int {
		byName := cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.ID, b.ID))
		if options.Sort != dto.CompanySortRelevance {
			return byName
		}
		return cmp.Or(cmp.Compare(rank(a.Tracked), rank(b.Tracked)), cmp.Compare(rank(a.Tracked && a.Favourite), rank(b.Tracked && b.Favourite)), byName)
	})
	total := len(items)
	items = items[min(int(options.Offset), total):]
	if limit := int(options.Limit); limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	verified := map[string]bool{}
	for _, b := range f.boards {
		if b.Status == dto.BoardVerified && b.Source != "wttj" {
			verified[b.CompanyID] = true
		}
	}
	return dto.CompanyPage{Items: items, Total: total, AllCompanies: len(f.companies), VerifiedBoardCompanies: len(verified)}, nil
}

func (f *FakeStore) hasVerifiedBoardOn(companyID string, sources []string) bool {
	for _, b := range f.boards {
		if b.CompanyID == companyID && b.Status == dto.BoardVerified && slices.Contains(sources, b.Source) {
			return true
		}
	}
	return false
}

func (f *FakeStore) CountCompaniesByATS(context.Context) ([]dto.ATSCount, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	companies := map[string]map[string]bool{}
	for _, b := range f.boards {
		if b.Status != dto.BoardVerified || b.Source == "wttj" {
			continue
		}
		if companies[b.Source] == nil {
			companies[b.Source] = map[string]bool{}
		}
		companies[b.Source][b.CompanyID] = true
	}
	counts := make([]dto.ATSCount, 0, len(companies))
	for source, ids := range companies {
		counts = append(counts, dto.ATSCount{Source: source, Companies: len(ids)})
	}
	slices.SortFunc(counts, func(a, b dto.ATSCount) int { return cmp.Compare(a.Source, b.Source) })
	return counts, nil
}

func (f *FakeStore) hasBoard(companyID string) bool {
	for _, b := range f.boards {
		if b.CompanyID == companyID {
			return true
		}
	}
	return false
}

func rank(first bool) int {
	if first {
		return 0
	}
	return 1
}

func (f *FakeStore) GetCompanyForUser(_ context.Context, userID, id string) (dto.Company, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, err := lookup(f.companies, id)
	if err != nil {
		return dto.Company{}, err
	}
	if tr, ok := f.tracking[trackingKey(userID, c.ID)]; ok {
		c.Tracked, c.ReviewState, c.CheckIntervalMinutes = tr.Enabled, tr.ReviewState, tr.CheckIntervalMinutes
	}
	c.Favourite = f.favourites[trackingKey(userID, c.ID)]
	return c, nil
}

func (f *FakeStore) SetCompanyKind(_ context.Context, companyID string, kind dto.CompanyKind) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.companies[companyID]
	if !ok {
		return data.ErrNotFound
	}
	c.Kind = kind
	f.companies[companyID] = c
	return nil
}

func (f *FakeStore) SuggestCompanyKind(_ context.Context, companyID string, kind dto.CompanyKind) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.companies[companyID]
	if !ok || c.Kind != dto.KindUnclassified {
		return false, nil
	}
	c.Kind = kind
	f.companies[companyID] = c
	return true, nil
}

func (f *FakeStore) SetCompanyFavourite(_ context.Context, userID, companyID string, favourite bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.companies[companyID]; !ok {
		return data.ErrNotFound
	}
	key := trackingKey(userID, companyID)
	t, tracked := f.tracking[key]
	if !favourite {
		delete(f.favourites, key)
		if tracked {
			t.CheckIntervalMinutes = 360
			f.tracking[key] = t
		}
		return nil
	}
	f.favourites[key] = true
	hasATS := false
	for _, b := range f.boards {
		role, _ := sourcespec.SourceRole(b.Source)
		if b.CompanyID == companyID && b.Status == dto.BoardVerified && role == sourcespec.RoleATS {
			hasATS = true
		}
	}
	if !hasATS {
		return nil
	}
	t.UserID, t.CompanyID, t.Enabled, t.CheckIntervalMinutes = userID, companyID, true, 60
	t.ReviewState = "kept"
	f.tracking[key] = t
	return nil
}

func (f *FakeStore) ListTrackedCompaniesForUser(_ context.Context, userID string) ([]dto.TrackedCompany, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]dto.TrackedCompany, 0)
	for _, c := range f.companies {
		tr, ok := f.tracking[trackingKey(userID, c.ID)]
		if !ok {
			continue
		}
		tc := dto.TrackedCompany{
			ID: c.ID, Name: c.Name, Slug: c.Slug, Enabled: tr.Enabled, ReviewState: tr.ReviewState,
			CheckIntervalMinutes: tr.CheckIntervalMinutes, Boards: []dto.TrackedBoard{},
		}
		for _, b := range f.boards {
			if b.CompanyID == c.ID {
				tc.Boards = append(tc.Boards, dto.TrackedBoard{ID: b.ID, Source: b.Source, BoardToken: b.BoardToken, Status: b.Status})
			}
		}
		slices.SortFunc(tc.Boards, func(a, b dto.TrackedBoard) int { return cmp.Compare(a.ID, b.ID) })
		for _, j := range f.jobs {
			if j.CompanyID == c.ID {
				tc.OpenJobs++
			}
		}
		out = append(out, tc)
	}
	slices.SortFunc(out, func(a, b dto.TrackedCompany) int { return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.ID, b.ID)) })
	return out, nil
}

func (f *FakeStore) ListNewCompanies(_ context.Context, userID string, limit, offset int32) (dto.NewCompanyPage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]dto.NewCompany, 0)
	for _, c := range f.companies {
		if f.tracking[trackingKey(userID, c.ID)].ReviewState != "new" {
			continue
		}
		nc := dto.NewCompany{ID: c.ID, Name: c.Name, Slug: c.Slug, Boards: []dto.TrackedBoard{}}
		if p, ok := f.profiles[c.ID]; ok {
			nc.Profile = &p
		}
		for _, b := range f.boards {
			if b.CompanyID == c.ID {
				nc.Boards = append(nc.Boards, dto.TrackedBoard{ID: b.ID, Source: b.Source, BoardToken: b.BoardToken, Status: b.Status})
			}
		}
		slices.SortFunc(nc.Boards, func(a, b dto.TrackedBoard) int { return cmp.Compare(a.ID, b.ID) })
		out = append(out, nc)
	}
	slices.SortFunc(out, func(a, b dto.NewCompany) int { return cmp.Compare(b.ID, a.ID) })
	page := dto.NewCompanyPage{Total: len(out), Items: []dto.NewCompany{}}
	if start := int(offset); start < len(out) {
		page.Items = out[start:min(start+int(limit), len(out))]
	}
	return page, nil
}

func (f *FakeStore) SaveCompanyProfile(_ context.Context, companyID, _ string, profile dto.CompanyProfile) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.profiles[companyID] = profile
	return nil
}

func (f *FakeStore) ListNewCompanyJobs(_ context.Context, userID string, companyIDs []string) ([]dto.Job, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []dto.Job
	for _, j := range f.jobs {
		if f.tracking[trackingKey(userID, j.CompanyID)].ReviewState == "new" && slices.Contains(companyIDs, j.CompanyID) {
			out = append(out, j)
		}
	}
	return out, nil
}

func (f *FakeStore) DeleteCompanyTracking(_ context.Context, userID, companyID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := trackingKey(userID, companyID)
	if _, ok := f.tracking[key]; !ok {
		return data.ErrNotFound
	}
	delete(f.tracking, key)
	return nil
}

func (f *FakeStore) SetCompanyTracking(_ context.Context, userID, companyID string, enabled bool, checkIntervalMinutes int) (dto.CompanyTracking, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := trackingKey(userID, companyID)
	if checkIntervalMinutes == 0 {
		checkIntervalMinutes = cmp.Or(f.tracking[key].CheckIntervalMinutes, 360)
	}
	t := dto.CompanyTracking{
		UserID: userID, CompanyID: companyID, Enabled: enabled,
		ReviewState: cmp.Or(f.tracking[key].ReviewState, "kept"), CheckIntervalMinutes: checkIntervalMinutes,
	}
	f.tracking[key] = t
	return t, nil
}

func (f *FakeStore) TrackDiscoveredCompany(_ context.Context, userID, companyID string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := trackingKey(userID, companyID)
	if _, ok := f.tracking[key]; ok {
		return false, nil
	}
	f.tracking[key] = dto.CompanyTracking{UserID: userID, CompanyID: companyID, Enabled: true, ReviewState: "new", CheckIntervalMinutes: 360}
	return true, nil
}

func (f *FakeStore) SetCompanyReviewState(_ context.Context, userID, companyID, state string) (dto.CompanyTracking, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := trackingKey(userID, companyID)
	t, ok := f.tracking[key]
	if !ok {
		return dto.CompanyTracking{}, data.ErrNotFound
	}
	t.Enabled = state != "dismissed"
	if state == "dismissed" {
		delete(f.favourites, key)
	}
	t.ReviewState = state
	f.tracking[key] = t
	return t, nil
}

func (f *FakeStore) VerifyCompanyBoard(_ context.Context, companyID, source, token, method, via string) (dto.CompanyBoard, error) {
	token = sourcespec.NormalizeBoardToken(source, token)
	f.mu.Lock()
	defer f.mu.Unlock()
	for id, b := range f.boards {
		if b.CompanyID != companyID || b.Source != source || b.BoardToken != token {
			continue
		}
		if b.Status != dto.BoardCandidate {
			continue
		}
		now := time.Now()
		b.Status, b.VerificationMethod, b.VerifiedAt, b.DiscoveredVia = dto.BoardVerified, method, &now, via
		f.boards[id] = b
		return b, nil
	}
	return dto.CompanyBoard{}, data.ErrNotFound
}

func (f *FakeStore) ListCompaniesToCrawl(_ context.Context, limit int) ([]dto.Company, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]dto.Company, 0, len(f.companies))
	for _, c := range f.companies {
		if c.LastCrawledAt == nil {
			out = append(out, c)
		}
	}
	slices.SortFunc(out, func(a, b dto.Company) int { return cmp.Compare(a.ID, b.ID) })
	if limit > 0 && limit < len(out) {
		out = out[:limit]
	}
	return out, nil
}

func (f *FakeStore) TouchCompanyCrawled(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.companies[id]
	if !ok {
		return data.ErrNotFound
	}
	now := time.Now()
	c.LastCrawledAt = &now
	f.companies[id] = c
	return nil
}

func (f *FakeStore) RenameCompany(_ context.Context, id, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.companies[id]
	if !ok {
		return data.ErrNotFound
	}
	c.Name = name
	f.companies[id] = c
	return nil
}

func (f *FakeStore) ListUntrackedDiscoveredBoards(_ context.Context, uk bool, ukSince time.Time) ([]dto.CompanyBoard, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	tracked := make(map[string]bool, len(f.tracking))
	for _, t := range f.tracking {
		tracked[t.CompanyID] = true
	}
	out := make([]dto.CompanyBoard, 0)
	for _, b := range f.boards {
		if b.Status == dto.BoardVerified && b.VerificationMethod == "discovered" && !tracked[b.CompanyID] &&
			(b.UKJobsSeenAt != nil && !b.UKJobsSeenAt.Before(ukSince)) == uk {
			out = append(out, b)
		}
	}
	slices.SortFunc(out, func(a, b dto.CompanyBoard) int { return cmp.Compare(a.ID, b.ID) })
	return out, nil
}

func (f *FakeStore) ListCompanyBoards(_ context.Context, companyID string) ([]dto.CompanyBoard, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]dto.CompanyBoard, 0)
	for _, b := range f.boards {
		if b.CompanyID == companyID {
			if st := f.pollState[b.ID]; st != nil {
				b.LastReportedTotal, b.LastParsed = st.reported, st.parsed
			}
			out = append(out, b)
		}
	}
	slices.SortFunc(out, func(a, b dto.CompanyBoard) int { return cmp.Compare(a.ID, b.ID) })
	return out, nil
}

func (f *FakeStore) UpsertCandidateBoard(_ context.Context, companyID, source, token string) (dto.CompanyBoard, error) {
	token = sourcespec.NormalizeBoardToken(source, token)
	f.mu.Lock()
	defer f.mu.Unlock()
	for id, b := range f.boards {
		if b.Source == source && b.BoardToken == token {
			if b.CompanyID != companyID {
				return dto.CompanyBoard{}, store.ErrBoardConflict
			}
			return f.boards[id], nil
		}
	}
	id := f.nextID()
	board := dto.CompanyBoard{ID: id, CompanyID: companyID, Source: source, BoardToken: token, Status: dto.BoardCandidate, CreatedAt: time.Now()}
	f.boards[id] = board
	return board, nil
}

func (f *FakeStore) GetBoardCompanyID(_ context.Context, source, token string) (string, error) {
	token = sourcespec.NormalizeBoardToken(source, token)
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, b := range f.boards {
		if b.Source == source && b.BoardToken == token {
			return b.CompanyID, nil
		}
	}
	return "", data.ErrNotFound
}

func (f *FakeStore) GetVerifiedBoardID(_ context.Context, source, token string) (string, error) {
	token = sourcespec.NormalizeBoardToken(source, token)
	f.mu.Lock()
	defer f.mu.Unlock()
	for id, b := range f.boards {
		if b.Source == source && b.BoardToken == token && b.Status == dto.BoardVerified {
			return id, nil
		}
	}
	return "", data.ErrNotFound
}

func (f *FakeStore) ListVerifiedBoardsBySlug(_ context.Context, slugs []string) ([]dto.CardBoard, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []dto.CardBoard
	for _, c := range f.companies {
		if !slices.Contains(slugs, c.Slug) {
			continue
		}
		tracked := slices.ContainsFunc(slices.Collect(maps.Values(f.tracking)), func(t dto.CompanyTracking) bool {
			return t.CompanyID == c.ID && t.Enabled
		})
		for _, b := range f.boards {
			if b.CompanyID == c.ID && b.Status == dto.BoardVerified {
				st := f.pollState[b.ID]
				polled := st != nil && !st.completedAt.IsZero()
				out = append(out, dto.CardBoard{CompanySlug: c.Slug, Source: b.Source, BoardToken: b.BoardToken, Tracked: tracked, Polled: polled})
			}
		}
	}
	return out, nil
}

func (f *FakeStore) ListVerifiedCompanySlugs(_ context.Context, slugs []string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	verified := map[string]bool{}
	for _, b := range f.boards {
		if b.Status == dto.BoardVerified {
			verified[b.CompanyID] = true
		}
	}
	var out []string
	for _, c := range f.companies {
		if verified[c.ID] && slices.Contains(slugs, c.Slug) {
			out = append(out, c.Slug)
		}
	}
	return out, nil
}

func (f *FakeStore) GetLastScraped(_ context.Context, source string) (time.Time, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.lastScraped[source]
	return t, ok, nil
}

func (f *FakeStore) SetLastScraped(_ context.Context, source string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastScraped[source] = time.Now()
	return nil
}

func (f *FakeStore) pollStateFor(boardID string) *boardPollState {
	st, ok := f.pollState[boardID]
	if !ok {
		st = &boardPollState{}
		f.pollState[boardID] = st
	}
	return st
}

func (f *FakeStore) ClaimBoard(_ context.Context, id string, manual bool) (dto.BoardPoll, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.boards[id]
	if !ok {
		return dto.BoardPoll{}, store.ErrBoardClaimUnavailable
	}
	st := f.pollStateFor(id)
	if st.leaseOwner != "" && time.Now().Before(st.leaseExpires) {
		return dto.BoardPoll{}, store.ErrBoardClaimUnavailable
	}
	st.leaseOwner = f.nextID()
	st.leaseExpires = time.Now().Add(10 * time.Minute)
	st.version++
	return dto.BoardPoll{
		ID: id, CompanyID: b.CompanyID, CompanySlug: f.companies[b.CompanyID].Slug,
		Source: b.Source, Token: b.BoardToken, IntervalMinutes: 360,
		LeaseOwner: st.leaseOwner, Version: st.version, StartedAt: time.Now(), Manual: manual,
	}, nil
}

func leaseHeldBy(st *boardPollState, poll dto.BoardPoll) bool {
	return st != nil && st.leaseOwner == poll.LeaseOwner && st.version == poll.Version
}

func (f *FakeStore) FailBoard(_ context.Context, poll dto.BoardPoll) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	st := f.pollState[poll.ID]
	if !leaseHeldBy(st, poll) {
		return store.ErrBoardClaimUnavailable
	}
	st.leaseOwner = ""
	return nil
}

func (f *FakeStore) CompleteBoard(_ context.Context, snapshot dto.BoardSnapshot) ([]dto.JobChange, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !snapshot.Complete {
		return nil, errors.New("incomplete board snapshot")
	}
	poll := snapshot.Poll
	st := f.pollState[poll.ID]
	if !leaseHeldBy(st, poll) {
		return nil, store.ErrBoardClaimUnavailable
	}
	st.leaseOwner = ""
	urls := make(map[string]bool, len(snapshot.Jobs))
	for _, job := range snapshot.Jobs {
		urls[job.URL] = true
	}
	st.reported, st.parsed = snapshot.Reported, len(urls)
	st.completedAt = time.Now()
	if !poll.Manual {
		st.checkPublishedAt = time.Time{}
	}
	return nil, nil
}

func boardPollFor(id string, b dto.CompanyBoard) dto.BoardPoll {
	return dto.BoardPoll{ID: id, CompanyID: b.CompanyID, Source: b.Source, Token: b.BoardToken, IntervalMinutes: 360}
}

func (f *FakeStore) ListDueBoards(context.Context) ([]dto.BoardPoll, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]dto.BoardPoll, 0)
	for id, b := range f.boards {
		if b.Status != dto.BoardVerified {
			continue
		}
		if st := f.pollState[id]; st != nil && (st.leaseOwner != "" || time.Since(st.checkPublishedAt) < 24*time.Hour) {
			continue
		}
		out = append(out, boardPollFor(id, b))
	}
	slices.SortFunc(out, func(a, b dto.BoardPoll) int { return cmp.Compare(a.ID, b.ID) })
	return out, nil
}

func (f *FakeStore) MarkBoardCheckPublished(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pollStateFor(id).checkPublishedAt = time.Now()
	return nil
}

func (f *FakeStore) ClearBoardCheckPublished(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if st := f.pollState[id]; st != nil {
		st.checkPublishedAt = time.Time{}
	}
	return nil
}

func (f *FakeStore) ListActiveBoards(context.Context) ([]dto.BoardPoll, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]dto.BoardPoll, 0)
	for id, b := range f.boards {
		if b.Status == dto.BoardRetired {
			continue
		}
		out = append(out, boardPollFor(id, b))
	}
	slices.SortFunc(out, func(a, b dto.BoardPoll) int { return cmp.Compare(a.ID, b.ID) })
	return out, nil
}

func (f *FakeStore) CreateSourceTarget(_ context.Context, userID, source, value string, enabled bool, filters map[string]string, window dto.RunWindow, nextRunAt *time.Time) (dto.SourceTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := targetKey(userID, source, value)
	if _, ok := f.targetByKey[key]; ok {
		return dto.SourceTarget{}, store.ErrSourceTargetExists
	}
	id := f.nextID()
	manual := runwindow.Default()
	manual.IntervalMinutes = nil
	t := dto.SourceTarget{
		ID: id, UserID: userID, Source: source, Value: value, Enabled: enabled, Filters: filters,
		RunWindow: overlayWindow(manual, window), NextRunAt: nextRunAt, UpdatedAt: time.Now(),
	}
	f.sourceTargets[id] = t
	f.targetByKey[key] = id
	return t, nil
}

func (f *FakeStore) CreateSourceTargetWithRun(ctx context.Context, userID, source, value string, enabled bool, filters map[string]string, window dto.RunWindow, nextRunAt *time.Time) (dto.SourceTarget, error) {
	t, err := f.CreateSourceTarget(ctx, userID, source, value, enabled, filters, window, nextRunAt)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	return f.StartSourceTargetRun(ctx, t.ID, nextRunAt)
}

func overlayWindow(base, set dto.RunWindow) dto.RunWindow {
	base.IntervalMinutes = set.IntervalMinutes
	if len(set.Weekdays) > 0 {
		base.Weekdays = runwindow.Weekdays(runwindow.WeekdayMask(set.Weekdays))
	}
	base.Start = cmp.Or(set.Start, base.Start)
	base.End = cmp.Or(set.End, base.End)
	base.Timezone = cmp.Or(set.Timezone, base.Timezone)
	return base
}

func (f *FakeStore) UpdateSourceTarget(_ context.Context, id, userID string, enabled *bool, window *dto.RunWindow, nextRunAt *time.Time) (dto.SourceTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.sourceTargets[id]
	if !ok || t.UserID != userID {
		return dto.SourceTarget{}, data.ErrNotFound
	}
	if enabled != nil {
		t.Enabled = *enabled
		if *enabled {
			t.DisabledReason = ""
		}
	}
	if window != nil {
		t.RunWindow = overlayWindow(t.RunWindow, *window)
		t.NextRunAt = nextRunAt
	}
	f.sourceTargets[id] = t
	return t, nil
}

func (f *FakeStore) DeleteSourceTarget(_ context.Context, id, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.sourceTargets[id]
	if !ok || t.UserID != userID {
		return nil
	}
	delete(f.sourceTargets, id)
	delete(f.targetByKey, targetKey(userID, t.Source, t.Value))
	return nil
}

func (f *FakeStore) ListSourceTargetsByUser(_ context.Context, userID string) ([]dto.SourceTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]dto.SourceTarget, 0)
	for _, t := range f.sourceTargets {
		if t.UserID == userID {
			out = append(out, t)
		}
	}
	slices.SortFunc(out, func(a, b dto.SourceTarget) int { return cmp.Compare(a.ID, b.ID) })
	return out, nil
}

func (f *FakeStore) StartSourceTargetRun(_ context.Context, id string, nextRunAt *time.Time) (dto.SourceTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.sourceTargets[id]
	if !ok {
		return dto.SourceTarget{}, data.ErrNotFound
	}
	t.RunID = f.nextID()
	t.NextRunAt = nextRunAt
	t.RunStatus, t.LastRunError, t.Enabled, t.DisabledReason = "queued", "", true, ""
	f.sourceTargets[id] = t
	return t, nil
}

func (f *FakeStore) ClaimDueSourceTargets(_ context.Context, limit int, nextRunAt func(dto.RunWindow) *time.Time) ([]dto.SourceTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now()
	var due []dto.SourceTarget
	for _, t := range f.sourceTargets {
		if t.Enabled && t.RunWindow.IntervalMinutes != nil && t.NextRunAt != nil && !t.NextRunAt.After(now) &&
			t.RunStatus != "queued" && t.RunStatus != "running" {
			due = append(due, t)
		}
	}
	slices.SortFunc(due, func(a, b dto.SourceTarget) int { return a.NextRunAt.Compare(*b.NextRunAt) })
	due = due[:min(limit, len(due))]
	for i, t := range due {
		t.RunID = f.nextID()
		t.NextRunAt = nextRunAt(t.RunWindow)
		t.RunStatus, t.LastRunError = "queued", ""
		f.sourceTargets[t.ID] = t
		due[i] = t
	}
	return due, nil
}

func (f *FakeStore) GetSourceTarget(_ context.Context, id string) (dto.SourceTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return lookup(f.sourceTargets, id)
}

func (f *FakeStore) TransitionSourceTargetRun(_ context.Context, id, runID, status, runError string) (dto.SourceTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.sourceTargets[id]
	if !ok || t.RunID != runID {
		return dto.SourceTarget{}, data.ErrNotFound
	}
	t.RunStatus, t.LastRunError = status, runError
	now := time.Now()
	t.LastRunAt = &now
	if status == "succeeded" {
		t.LastSucceededAt = &now
	}
	f.sourceTargets[id] = t
	return t, nil
}

func (f *FakeStore) DisableSourceTargets(_ context.Context, source, reason string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var n int64
	for id, t := range f.sourceTargets {
		if t.Source != source || !t.Enabled {
			continue
		}
		t.Enabled, t.DisabledReason = false, reason
		if t.RunStatus == "queued" || t.RunStatus == "running" {
			t.RunStatus, t.LastRunError = "failed", reason
		}
		f.sourceTargets[id] = t
		n++
	}
	return n, nil
}

func (f *FakeStore) ListRecoverableSourceTargets(context.Context) ([]dto.SourceTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]dto.SourceTarget, 0)
	for _, t := range f.sourceTargets {
		if t.RunStatus == "running" {
			out = append(out, t)
		}
	}
	slices.SortFunc(out, func(a, b dto.SourceTarget) int { return cmp.Compare(a.ID, b.ID) })
	return out, nil
}

func (f *FakeStore) ClaimRecoverableSourceTarget(_ context.Context, id, runID string) (dto.SourceTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.sourceTargets[id]
	if !ok || t.RunID != runID || t.RunStatus != "running" {
		return dto.SourceTarget{}, data.ErrNotFound
	}
	t.RunStatus = "recovering"
	f.sourceTargets[id] = t
	return t, nil
}

func (f *FakeStore) SaveCards(_ context.Context, target dto.SourceTarget, cards []dto.Job) ([]sourcetargets.Candidate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]sourcetargets.Candidate, 0, len(cards))
	for _, card := range cards {
		if card.URL == "" {
			continue
		}
		normalized, err := store.NormalizeCandidateURL(card.URL)
		if err != nil {
			return nil, err
		}
		card.URL = normalized
		id, ok := f.candidateByURL[normalized]
		if !ok {
			id = f.nextID()
			f.candidateByURL[normalized] = id
		}
		card.Source = target.Source
		cand := sourcetargets.Candidate{ID: id, URL: card.URL, Card: card}
		out = append(out, cand)
		if f.discoveries[id] == nil {
			f.discoveries[id] = map[string]discovery{}
		}
		f.discoveries[id][target.ID] = discovery{userID: target.UserID, filters: target.Filters}
		if !sourcespec.CardComplete(target.Source) && card.RemoteCheckURL == "" && card.WorkArrangement == "" {
			cand.Card = dto.Job{URL: card.URL, Title: card.Title, CompanySlug: card.CompanySlug, Location: card.Location, Source: card.Source, PostedAt: card.PostedAt}
		}
		f.candidates[id] = cand
		f.rememberCardDate(card)
		if jobID, ok := f.byURL[card.URL]; ok && card.PostedAt != nil {
			f.applyPostedDate(jobID, *card.PostedAt)
		}
	}
	return out, nil
}

func (f *FakeStore) rememberCardDate(card dto.Job) {
	if card.PostedAt == nil {
		return
	}
	if prev, ok := f.datedCards[card.URL]; ok && !card.PostedAt.Before(*prev.PostedAt) {
		return
	}
	f.datedCards[card.URL] = card
}

func (f *FakeStore) applyPostedDate(jobID string, postedAt time.Time) {
	job := f.jobs[jobID]
	job.PostedAt, job.RepostedAt = applyPostedAt(job, &postedAt)
	f.jobs[jobID] = job
}

func cardMatchesJob(card, job dto.Job) bool {
	return card.CompanySlug == job.CompanySlug && jobmatch.Title(card.Title) == jobmatch.Title(job.Title) &&
		jobmatch.LocationsCompatible(jobmatch.Location(card.Location), jobmatch.Location(job.Location))
}

func (f *FakeStore) ApplyBoardCards(_ context.Context, _ string, cards []dto.Job) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, card := range cards {
		if card.PostedAt == nil || card.URL == "" {
			continue
		}
		var matches []string
		for _, id := range f.jobOrder {
			if cardMatchesJob(card, f.jobs[id]) {
				matches = append(matches, id)
			}
		}
		switch len(matches) {
		case 0:
			normalized, err := store.NormalizeCandidateURL(card.URL)
			if err != nil {
				return err
			}
			card.URL = normalized
			f.rememberCardDate(card)
		case 1:
			f.applyPostedDate(matches[0], *card.PostedAt)
		}
	}
	return nil
}

func (f *FakeStore) ListForUser(_ context.Context, _, afterID string, limit int) ([]sourcetargets.Candidate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ids := make([]string, 0, len(f.candidates))
	for id := range f.candidates {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	out := make([]sourcetargets.Candidate, 0, len(ids))
	skip := afterID != ""
	for _, id := range ids {
		if skip {
			if id == afterID {
				skip = false
			}
			continue
		}
		out = append(out, f.candidates[id])
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

type discovery struct {
	userID  string
	filters map[string]string
}

func (f *FakeStore) DiscoveryFilters(_ context.Context, userID string, candidateIDs []string) (map[string][]map[string]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string][]map[string]string{}
	for _, id := range candidateIDs {
		for _, d := range f.discoveries[id] {
			if d.userID == userID {
				out[id] = append(out[id], d.filters)
			}
		}
	}
	return out, nil
}

func (f *FakeStore) Assess(_ context.Context, candidateID, _ string, _ time.Time, passes bool) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return passes && !f.detailPending[candidateID], nil
}

func (f *FakeStore) MarkDetailPending(_ context.Context, candidateID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.detailPending[candidateID] = true
	return nil
}

func (f *FakeStore) DeleteExpiredCandidates(context.Context) error {
	return nil
}

func (f *FakeStore) ScrapeRunPageDone(context.Context, string) error { return nil }

func (f *FakeStore) ScrapeRunDeferred(context.Context, string) error { return nil }

func (f *FakeStore) ScrapeRunAttemptFailed(context.Context, string, int, int, string) error {
	return nil
}

func (f *FakeStore) FinishScrapeRun(context.Context, string, dto.StopReason) error { return nil }

func (f *FakeStore) BoardRunAttemptFailed(context.Context, dto.BoardRunStart, int, string) error {
	return nil
}

func (f *FakeStore) FinishBoardRun(context.Context, dto.BoardRunStart, dto.StopReason, dto.BoardTally) error {
	return nil
}

func (f *FakeStore) DeleteExpiredScrapeRuns(context.Context) error { return nil }

func (f *FakeStore) ListScrapeRuns(context.Context, string, dto.ScrapeRunListOptions) ([]dto.ScrapeRun, error) {
	return nil, nil
}

func (f *FakeStore) ListScrapeRunJobSnapshots(context.Context, string, string) ([]dto.ScrapeRunJobSnapshot, error) {
	return nil, nil
}

func (f *FakeStore) ListSourceHealth(context.Context, string) ([]dto.SourceHealth, error) {
	return nil, nil
}

func (f *FakeStore) ScrapeRunSummary(context.Context, string, dto.ScrapeRunSummaryOptions) (dto.ScrapeRunSummary, error) {
	return dto.ScrapeRunSummary{}, nil
}

var _ Store = (*FakeStore)(nil)

func (f *FakeStore) LookupFetch(_ context.Context, url string) (dto.CachedResponse, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	resp, ok := f.fetches[url]
	return resp, ok, nil
}

func (f *FakeStore) PutFetch(_ context.Context, resp dto.CachedResponse) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fetches[resp.URL] = resp
	return nil
}

func (f *FakeStore) ForgetFetches(_ context.Context, urls []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range urls {
		delete(f.fetches, u)
	}
	return nil
}

func (f *FakeStore) DeleteExpiredFetches(context.Context) error { return nil }

func (f *FakeStore) MarkBoardUKJobsSeen(_ context.Context, source, token string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now()
	for id, b := range f.boards {
		if b.Source == source && b.BoardToken == token {
			b.UKJobsSeenAt = &now
			f.boards[id] = b
		}
	}
	return nil
}

func (f *FakeStore) RecordDiscoverMiss(_ context.Context, source, token string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.misses[source+"/"+token] = time.Now()
	return nil
}

func (f *FakeStore) DiscoverMissedSince(_ context.Context, source, token string, since time.Time) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	at, ok := f.misses[source+"/"+token]
	return ok && !at.Before(since), nil
}

func (f *FakeStore) HarvestSeenIDs(_ context.Context, source string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Collect(maps.Keys(f.harvestSeen[source])), nil
}

func (f *FakeStore) RecordHarvestSeenIDs(_ context.Context, source, outcome string, ids []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.harvestSeen[source] == nil {
		f.harvestSeen[source] = make(map[string]string)
	}
	for _, id := range ids {
		f.harvestSeen[source][id] = outcome
	}
	return nil
}

func (f *FakeStore) RetireHarvestSeenIDs(_ context.Context, source string, ids []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, id := range ids {
		delete(f.harvestSeen[source], id)
	}
	return nil
}

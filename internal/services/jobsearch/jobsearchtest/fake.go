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
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
	"github.com/ollymarsters/job-scraper/internal/services/sourcetargets"
)

type boardPollState struct {
	leaseOwner   string
	leaseExpires time.Time
	version      int64
}

type FakeStore struct {
	mu sync.Mutex

	seq int

	jobs     map[string]dto.Job
	jobOrder []string
	byURL    map[string]string
	byBoard  map[string]string

	companies     map[string]dto.Company
	companyBySlug map[string]string
	tracking      map[string]dto.CompanyTracking

	boards    map[string]dto.CompanyBoard
	pollState map[string]*boardPollState

	lastScraped map[string]time.Time
	fetches     map[string]dto.CachedResponse
	profiles    map[string]dto.CompanyProfile

	sourceTargets map[string]dto.SourceTarget
	targetByKey   map[string]string

	candidates     map[string]sourcetargets.Candidate
	candidateByURL map[string]string
	detailPending  map[string]bool
}

func NewFakeStore() *FakeStore {
	return &FakeStore{
		jobs:           make(map[string]dto.Job),
		byURL:          make(map[string]string),
		byBoard:        make(map[string]string),
		companies:      make(map[string]dto.Company),
		companyBySlug:  make(map[string]string),
		tracking:       make(map[string]dto.CompanyTracking),
		boards:         make(map[string]dto.CompanyBoard),
		pollState:      make(map[string]*boardPollState),
		lastScraped:    make(map[string]time.Time),
		fetches:        make(map[string]dto.CachedResponse),
		profiles:       make(map[string]dto.CompanyProfile),
		sourceTargets:  make(map[string]dto.SourceTarget),
		targetByKey:    make(map[string]string),
		candidates:     make(map[string]sourcetargets.Candidate),
		candidateByURL: make(map[string]string),
		detailPending:  make(map[string]bool),
	}
}

func (f *FakeStore) nextID(prefix string) string {
	f.seq++
	return fmt.Sprintf("%s-%06d", prefix, f.seq)
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

func (f *FakeStore) SaveCanonical(_ context.Context, job dto.Job) (dto.Job, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	job.URL = normalizeJobURL(job.URL)
	fingerprint := jobFingerprint(job)
	id, ok := f.jobIdentity(job)
	if !ok {
		id = f.nextID("job")
		job.ID, job.ContentFingerprint = id, fingerprint
		f.jobs[id] = job
		f.jobOrder = append(f.jobOrder, id)
		f.indexJob(job)
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
	job.ID, job.ContentFingerprint, job.URL = id, fingerprint, prev.URL
	f.jobs[id] = job
	f.indexJob(job)
	return job, status, nil
}

func (f *FakeStore) Page(_ context.Context, _ string, options dto.JobPageOptions) (dto.JobPage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	items := make([]dto.Job, 0, len(f.jobOrder))
	skip := options.CursorID != ""
	for _, id := range f.jobOrder {
		if skip {
			if id == options.CursorID {
				skip = false
			}
			continue
		}
		items = append(items, f.jobs[id])
	}
	if limit := int(options.Limit); limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	return dto.JobPage{Items: items}, nil
}

func (f *FakeStore) GetJob(_ context.Context, jobID, _ string) (dto.Job, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return lookup(f.jobs, jobID)
}

func (f *FakeStore) ListJobs(_ context.Context, _ string) ([]dto.Job, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]dto.Job, 0, len(f.jobOrder))
	for _, id := range f.jobOrder {
		out = append(out, f.jobs[id])
	}
	return out, nil
}

func (f *FakeStore) NewURLs(_ context.Context, urls []string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(urls))
	for _, u := range urls {
		if _, ok := f.byURL[u]; !ok {
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
		existing.Name = c.Name
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
	id := f.nextID("company")
	company := dto.Company{
		ID: id, Slug: c.Slug, Name: c.Name, ATSSource: c.ATSSource, ATSToken: c.ATSToken,
		Domain: c.Domain, LinkedInCompanyID: c.LinkedInCompanyID, FirstSeenAt: time.Now(),
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

func (f *FakeStore) ListCompaniesForUser(_ context.Context, userID string) ([]dto.Company, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]dto.Company, 0, len(f.companies))
	for _, c := range f.companies {
		if tr, ok := f.tracking[trackingKey(userID, c.ID)]; ok {
			c.Tracked, c.ReviewState, c.CheckIntervalMinutes = tr.Enabled, tr.ReviewState, tr.CheckIntervalMinutes
		}
		out = append(out, c)
	}
	slices.SortFunc(out, func(a, b dto.Company) int { return cmp.Compare(a.ID, b.ID) })
	return out, nil
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

func (f *FakeStore) ListNewCompanies(_ context.Context, userID string) ([]dto.NewCompany, error) {
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
	return out, nil
}

func (f *FakeStore) SaveCompanyProfile(_ context.Context, companyID, _ string, profile dto.CompanyProfile) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.profiles[companyID] = profile
	return nil
}

func (f *FakeStore) ListNewCompanyJobs(_ context.Context, userID string) ([]dto.Job, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []dto.Job
	for _, j := range f.jobs {
		if f.tracking[trackingKey(userID, j.CompanyID)].ReviewState == "new" {
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
	t.ReviewState = state
	f.tracking[key] = t
	return t, nil
}

func (f *FakeStore) VerifyCompanyBoard(_ context.Context, companyID, source, token, method string) (dto.CompanyBoard, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for id, b := range f.boards {
		if b.CompanyID != companyID || b.Source != source || b.BoardToken != token {
			continue
		}
		now := time.Now()
		b.Status, b.VerificationMethod, b.VerifiedAt = dto.BoardVerified, method, &now
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

func (f *FakeStore) ListUntrackedDiscoveredBoards(_ context.Context) ([]dto.CompanyBoard, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	tracked := make(map[string]bool, len(f.tracking))
	for _, t := range f.tracking {
		tracked[t.CompanyID] = true
	}
	out := make([]dto.CompanyBoard, 0)
	for _, b := range f.boards {
		if b.Status == dto.BoardVerified && b.VerificationMethod == "discovered" && !tracked[b.CompanyID] {
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
			out = append(out, b)
		}
	}
	slices.SortFunc(out, func(a, b dto.CompanyBoard) int { return cmp.Compare(a.ID, b.ID) })
	return out, nil
}

func (f *FakeStore) UpsertCandidateBoard(_ context.Context, companyID, source, token string) (dto.CompanyBoard, error) {
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
	id := f.nextID("board")
	board := dto.CompanyBoard{ID: id, CompanyID: companyID, Source: source, BoardToken: token, Status: dto.BoardCandidate, CreatedAt: time.Now()}
	f.boards[id] = board
	return board, nil
}

func (f *FakeStore) GetBoardCompanyID(_ context.Context, source, token string) (string, error) {
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
	f.mu.Lock()
	defer f.mu.Unlock()
	for id, b := range f.boards {
		if b.Source == source && b.BoardToken == token && b.Status == dto.BoardVerified {
			return id, nil
		}
	}
	return "", data.ErrNotFound
}

func (f *FakeStore) ListPolledCompanySlugs(_ context.Context, slugs []string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, c := range f.companies {
		if !slices.Contains(slugs, c.Slug) {
			continue
		}
		verified := slices.ContainsFunc(slices.Collect(maps.Values(f.boards)), func(b dto.CompanyBoard) bool {
			return b.CompanyID == c.ID && b.Status == dto.BoardVerified
		})
		tracked := slices.ContainsFunc(slices.Collect(maps.Values(f.tracking)), func(t dto.CompanyTracking) bool {
			return t.CompanyID == c.ID && t.Enabled
		})
		if verified && tracked {
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
	st.leaseOwner = f.nextID("lease")
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

func (f *FakeStore) CompleteBoard(_ context.Context, snapshot dto.BoardSnapshot) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !snapshot.Complete {
		return errors.New("incomplete board snapshot")
	}
	poll := snapshot.Poll
	st := f.pollState[poll.ID]
	if !leaseHeldBy(st, poll) {
		return store.ErrBoardClaimUnavailable
	}
	st.leaseOwner = ""
	return nil
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
		if st := f.pollState[id]; st != nil && st.leaseOwner != "" {
			continue
		}
		out = append(out, boardPollFor(id, b))
	}
	slices.SortFunc(out, func(a, b dto.BoardPoll) int { return cmp.Compare(a.ID, b.ID) })
	return out, nil
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

func (f *FakeStore) CreateSourceTarget(_ context.Context, userID, source, value string, enabled bool, filters map[string]string) (dto.SourceTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := targetKey(userID, source, value)
	if _, ok := f.targetByKey[key]; ok {
		return dto.SourceTarget{}, store.ErrSourceTargetExists
	}
	id := f.nextID("target")
	t := dto.SourceTarget{ID: id, UserID: userID, Source: source, Value: value, Enabled: enabled, Filters: filters, UpdatedAt: time.Now()}
	f.sourceTargets[id] = t
	f.targetByKey[key] = id
	return t, nil
}

func (f *FakeStore) CreateSourceTargetWithRun(ctx context.Context, userID, source, value string, enabled bool, filters map[string]string) (dto.SourceTarget, error) {
	t, err := f.CreateSourceTarget(ctx, userID, source, value, enabled, filters)
	if err != nil {
		return dto.SourceTarget{}, err
	}
	return f.StartSourceTargetRun(ctx, t.ID)
}

func (f *FakeStore) UpdateSourceTarget(_ context.Context, id, userID string, enabled *bool, checkIntervalMinutes *int) (dto.SourceTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.sourceTargets[id]
	if !ok || t.UserID != userID {
		return dto.SourceTarget{}, data.ErrNotFound
	}
	if enabled != nil {
		t.Enabled = *enabled
	}
	if checkIntervalMinutes != nil {
		t.CheckIntervalMinutes = *checkIntervalMinutes
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

func (f *FakeStore) StartSourceTargetRun(_ context.Context, id string) (dto.SourceTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.sourceTargets[id]
	if !ok {
		return dto.SourceTarget{}, data.ErrNotFound
	}
	t.RunID = f.nextID("run")
	t.RunStatus, t.LastRunError, t.Enabled = "queued", "", true
	f.sourceTargets[id] = t
	return t, nil
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
	f.sourceTargets[id] = t
	return t, nil
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
		id, ok := f.candidateByURL[card.URL]
		if !ok {
			id = f.nextID("candidate")
			f.candidateByURL[card.URL] = id
		}
		card.Source = target.Source
		cand := sourcetargets.Candidate{ID: id, URL: card.URL, Card: card}
		f.candidates[id] = cand
		out = append(out, cand)
	}
	return out, nil
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

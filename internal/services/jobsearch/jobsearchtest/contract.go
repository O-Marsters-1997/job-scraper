package jobsearchtest

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/runwindow"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/store"
	"github.com/ollymarsters/job-scraper/internal/services/sourcetargets"
)

type Store interface {
	jobsearch.Store
	sourcetargets.Store
}

func NewDeps(st Store) jobsearch.Deps {
	return jobsearch.Deps{
		Store:         st,
		SourceTargets: st,
		Scoring:       NewNoopScoring(),
		Queue:         NoopQueue{},
	}
}

func RunStoreContract(t *testing.T, newStore func(t *testing.T) (Store, string)) {
	t.Helper()

	t.Run("fetch cache round trips a redirect and forgets by url", func(t *testing.T) {
		t.Parallel()
		st, _ := newStore(t)
		ctx := t.Context()
		want := dto.CachedResponse{URL: "https://x.test/a", Status: 302, Header: http.Header{"Location": {"https://x.test/b"}}, Body: []byte("body")}
		if _, ok, err := st.LookupFetch(ctx, want.URL); ok || err != nil {
			t.Fatalf("LookupFetch(empty) = %v, %v, want miss", ok, err)
		}
		if err := st.PutFetch(ctx, want); err != nil {
			t.Fatal(err)
		}
		got, ok, err := st.LookupFetch(ctx, want.URL)
		if err != nil || !ok {
			t.Fatalf("LookupFetch = %v, %v, want hit", ok, err)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("LookupFetch (-want +got):\n%s", diff)
		}
		if err := st.ForgetFetches(ctx, []string{want.URL}); err != nil {
			t.Fatal(err)
		}
		if _, ok, _ := st.LookupFetch(ctx, want.URL); ok {
			t.Error("LookupFetch after ForgetFetches hit, want miss")
		}
	})

	t.Run("a recorded discover miss is seen only since before it", func(t *testing.T) {
		t.Parallel()
		st, _ := newStore(t)
		ctx := t.Context()
		if err := st.RecordDiscoverMiss(ctx, "greenhouse", "acme"); err != nil {
			t.Fatal(err)
		}
		for name, tc := range map[string]struct {
			token string
			since time.Time
			want  bool
		}{
			"recent window": {"acme", time.Now().Add(-time.Hour), true},
			"future window": {"acme", time.Now().Add(time.Hour), false},
			"other board":   {"other", time.Now().Add(-time.Hour), false},
		} {
			got, err := st.DiscoverMissedSince(ctx, "greenhouse", tc.token, tc.since)
			if err != nil || got != tc.want {
				t.Errorf("%s: DiscoverMissedSince = %v, %v, want %v", name, got, err, tc.want)
			}
		}
		if err := st.RecordDiscoverMiss(ctx, "greenhouse", "acme"); err != nil {
			t.Errorf("RecordDiscoverMiss(repeat) = %v, want nil", err)
		}
	})

	t.Run("marking a board's UK jobs seen stamps it", func(t *testing.T) {
		t.Parallel()
		st, _ := newStore(t)
		ctx := t.Context()
		company, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := st.UpsertCandidateBoard(ctx, company.ID, "greenhouse", "acme"); err != nil {
			t.Fatal(err)
		}
		if err := st.MarkBoardUKJobsSeen(ctx, "greenhouse", "acme"); err != nil {
			t.Fatal(err)
		}
		boards, err := st.ListCompanyBoards(ctx, company.ID)
		if err != nil || len(boards) != 1 || boards[0].UKJobsSeenAt == nil {
			t.Fatalf("ListCompanyBoards() = %+v, %v, want one stamped board", boards, err)
		}
	})

	t.Run("get an unknown job returns not found", func(t *testing.T) {
		t.Parallel()
		st, _ := newStore(t)
		_, err := st.GetJob(t.Context(), missingID, missingID)
		if !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("GetJob(...) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("save then get a canonical job round trips", func(t *testing.T) {
		t.Parallel()
		st, _ := newStore(t)
		ctx := t.Context()
		saved, status, err := st.SaveCanonical(ctx, dto.Job{Title: "Engineer", URL: "https://example.com/jobs/1"})
		if err != nil || status != "new" || saved.ID == "" {
			t.Fatalf("SaveCanonical(...) = %+v, %q, %v, want a new job", saved, status, err)
		}
		got, err := st.GetJob(ctx, saved.ID, missingID)
		if err != nil || got.Title != "Engineer" {
			t.Fatalf("GetJob(...) = %+v, %v, want the saved job", got, err)
		}
	})

	t.Run("marking seen is idempotent and unseen clears it", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		saved, _, err := st.SaveCanonical(ctx, dto.Job{Title: "Engineer", URL: "https://example.com/jobs/seen"})
		if err != nil {
			t.Fatalf("SaveCanonical(...) = %v", err)
		}
		isSeen := func() bool {
			t.Helper()
			got, err := st.GetJob(ctx, saved.ID, userID)
			if err != nil {
				t.Fatalf("GetJob(...) = %v", err)
			}
			return got.Seen
		}
		if isSeen() {
			t.Fatal("GetJob(new job).Seen = true, want false")
		}
		for range 2 {
			if err := st.MarkJobsSeen(ctx, userID, []string{saved.ID, missingID}, true); err != nil {
				t.Fatalf("MarkJobsSeen(true) = %v", err)
			}
		}
		if !isSeen() {
			t.Error("GetJob(seen job).Seen = false, want true")
		}
		if err := st.MarkJobsSeen(ctx, userID, []string{saved.ID}, false); err != nil {
			t.Fatalf("MarkJobsSeen(false) = %v", err)
		}
		if isSeen() {
			t.Error("GetJob(unseen job).Seen = true, want false")
		}
	})

	t.Run("resaving the same content is unchanged, changed content is changed", func(t *testing.T) {
		t.Parallel()
		st, _ := newStore(t)
		ctx := t.Context()
		job := dto.Job{Title: "Engineer", URL: "https://example.com/jobs/2"}
		saved, _, err := st.SaveCanonical(ctx, job)
		if err != nil {
			t.Fatalf("SaveCanonical(...) = %v", err)
		}

		resaved, status, err := st.SaveCanonical(ctx, job)
		if err != nil || status != "unchanged" || resaved.ID != saved.ID {
			t.Fatalf("resave = %+v, %q, %v, want unchanged with the same ID", resaved, status, err)
		}

		job.Title = "Senior Engineer"
		changed, status, err := st.SaveCanonical(ctx, job)
		if err != nil || status != "changed" || changed.ID != saved.ID {
			t.Fatalf("changed resave = %+v, %q, %v, want changed with the same ID", changed, status, err)
		}
	})

	runPostedContract(t, newStore)

	t.Run("listings are every url of a job in first-seen order", func(t *testing.T) {
		t.Parallel()
		st, _ := newStore(t)
		ctx := t.Context()
		posting := dto.Job{Title: "Engineer", BoardID: "33333333-3333-3333-3333-333333333333", ProviderPostingID: "p-1"}
		primary, secondary := posting, posting
		primary.URL, primary.Source = "https://boards.example.com/jobs/1", "greenhouse"
		secondary.URL, secondary.Source = "https://www.linkedin.com/jobs/view/1?utm_source=x", "linkedin"
		saved, _, err := st.SaveCanonical(ctx, primary)
		if err != nil {
			t.Fatalf("SaveCanonical(primary) = %v", err)
		}
		if _, _, err := st.SaveCanonical(ctx, secondary); err != nil {
			t.Fatalf("SaveCanonical(secondary) = %v", err)
		}
		if _, _, err := st.SaveCanonical(ctx, primary); err != nil {
			t.Fatalf("SaveCanonical(primary replay) = %v", err)
		}

		got, err := st.ListJobListings(ctx, saved.ID)
		if err != nil {
			t.Fatalf("ListJobListings(...) = %v", err)
		}
		want := []dto.JobListing{
			{Source: "greenhouse", URL: "https://boards.example.com/jobs/1"},
			{Source: "linkedin", URL: "https://www.linkedin.com/jobs/view/1"},
		}
		ignoreFirstSeen := cmpopts.IgnoreFields(dto.JobListing{}, "FirstSeenAt")
		if diff := cmp.Diff(want, got, ignoreFirstSeen); diff != "" {
			t.Fatalf("ListJobListings (-want +got):\n%s", diff)
		}
		if got[0].FirstSeenAt.IsZero() || got[1].FirstSeenAt.Before(got[0].FirstSeenAt) {
			t.Errorf("FirstSeenAt = %v, %v, want set and ascending", got[0].FirstSeenAt, got[1].FirstSeenAt)
		}
	})

	t.Run("listings of an unknown job are empty", func(t *testing.T) {
		t.Parallel()
		st, _ := newStore(t)
		got, err := st.ListJobListings(t.Context(), missingID)
		if err != nil || len(got) != 0 {
			t.Fatalf("ListJobListings(missing) = %v, %v, want none", got, err)
		}
	})

	t.Run("FindJobByPosting finds a job by board posting or by url and reports a miss", func(t *testing.T) {
		t.Parallel()
		st, _ := newStore(t)
		ctx := t.Context()
		company, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "find-co", Name: "Find Co"})
		if err != nil {
			t.Fatalf("UpsertCompany(...) = %v", err)
		}
		board, err := st.UpsertCandidateBoard(ctx, company.ID, "greenhouse", "find-co")
		if err != nil {
			t.Fatalf("UpsertCandidateBoard(...) = %v", err)
		}
		byBoard, _, err := st.SaveCanonical(ctx, dto.Job{
			Title: "Engineer", URL: "https://boards.greenhouse.io/find-co/jobs/42", Source: "greenhouse",
			CompanySlug: "find-co", BoardID: board.ID, ProviderPostingID: "42",
		})
		if err != nil {
			t.Fatalf("SaveCanonical(board) = %v", err)
		}
		byURL, _, err := st.SaveCanonical(ctx, dto.Job{Title: "Designer", URL: "https://example.com/jobs/find-1", Source: "other", CompanySlug: "find-co"})
		if err != nil {
			t.Fatalf("SaveCanonical(url) = %v", err)
		}

		for _, tt := range []struct {
			name                          string
			url, source, token, postingID string
			wantID                        string
		}{
			{"posting ID on the board", "https://elsewhere.test/x", "greenhouse", "find-co", "42", byBoard.ID},
			{"url alone", "https://example.com/jobs/find-1?utm_source=x", "", "", "", byURL.ID},
		} {
			got, err := st.FindJobByPosting(ctx, tt.url, tt.source, tt.token, tt.postingID)
			if err != nil || got.ID != tt.wantID {
				t.Errorf("FindJobByPosting(%s) = %q, %v, want %q", tt.name, got.ID, err, tt.wantID)
			}
		}
		_, err = st.FindJobByPosting(ctx, "https://example.com/jobs/none", "greenhouse", "find-co", "43")
		if !errors.Is(err, data.ErrNotFound) {
			t.Errorf("FindJobByPosting(miss) err = %v, want %v", err, data.ErrNotFound)
		}
	})

	t.Run("urls differing only by tracking parameters are one job", func(t *testing.T) {
		t.Parallel()
		st, _ := newStore(t)
		ctx := t.Context()
		first, _, err := st.SaveCanonical(ctx, dto.Job{Title: "Engineer", URL: "https://example.com/jobs/9?gh_jid=7&utm_source=x&team=a"})
		if err != nil {
			t.Fatalf("SaveCanonical(...) = %v", err)
		}
		second, status, err := st.SaveCanonical(ctx, dto.Job{Title: "Engineer", URL: "https://example.com/jobs/9?ref=y&gh_jid=7&team=a&gh_src=z"})
		if err != nil || status != "unchanged" || second.ID != first.ID {
			t.Fatalf("SaveCanonical(...) = %+v, %q, %v, want unchanged with ID %s", second, status, err, first.ID)
		}
		if want := "https://example.com/jobs/9?gh_jid=7&team=a"; second.URL != want {
			t.Fatalf("URL = %q, want %q", second.URL, want)
		}
	})

	t.Run("new urls filters out known ones", func(t *testing.T) {
		t.Parallel()
		st, _ := newStore(t)
		ctx := t.Context()
		if _, _, err := st.SaveCanonical(ctx, dto.Job{Title: "Engineer", URL: "https://example.com/jobs/3"}); err != nil {
			t.Fatalf("SaveCanonical(...) = %v", err)
		}
		fresh, err := st.NewURLs(ctx, []string{"https://example.com/jobs/3", "https://example.com/jobs/new"})
		if err != nil || len(fresh) != 1 || fresh[0] != "https://example.com/jobs/new" {
			t.Fatalf("NewURLs(...) = %v, %v, want only the unseen URL", fresh, err)
		}
	})

	t.Run("upsert company then get round trips", func(t *testing.T) {
		t.Parallel()
		st, _ := newStore(t)
		ctx := t.Context()
		c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "acme", Name: "Acme"})
		if err != nil || c.Slug != "acme" {
			t.Fatalf("UpsertCompany(...) = %+v, %v", c, err)
		}
		got, err := st.GetCompany(ctx, c.ID)
		if err != nil || got.Name != "Acme" {
			t.Fatalf("GetCompany(...) = %+v, %v, want Acme", got, err)
		}
	})

	t.Run("get an unknown company returns not found", func(t *testing.T) {
		t.Parallel()
		st, _ := newStore(t)
		_, err := st.GetCompany(t.Context(), missingID)
		if !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("GetCompany(...) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("get company for user carries that user's tracking", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "tracked-co", Name: "Tracked Co"})
		if err != nil {
			t.Fatalf("UpsertCompany(...) = %v", err)
		}
		if _, err := st.SetCompanyTracking(ctx, userID, c.ID, true, 180); err != nil {
			t.Fatalf("SetCompanyTracking(...) = %v", err)
		}
		got, err := st.GetCompanyForUser(ctx, userID, c.ID)
		if err != nil || got.Name != "Tracked Co" || !got.Tracked || got.CheckIntervalMinutes != 180 {
			t.Fatalf("GetCompanyForUser(...) = %+v, %v, want tracked Tracked Co", got, err)
		}
	})

	t.Run("get an unknown company for user returns not found", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		_, err := st.GetCompanyForUser(t.Context(), userID, missingID)
		if !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("GetCompanyForUser(...) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("set company tracking then list for user shows it", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "tracked-co", Name: "Tracked Co"})
		if err != nil {
			t.Fatalf("UpsertCompany(...) = %v", err)
		}
		if _, err := st.SetCompanyTracking(ctx, userID, c.ID, true, 180); err != nil {
			t.Fatalf("SetCompanyTracking(...) = %v", err)
		}
		page, err := st.PageCompaniesForUser(ctx, userID, dto.CompanyPageOptions{Limit: 10})
		if err != nil || len(page.Items) != 1 || !page.Items[0].Tracked || page.Items[0].CheckIntervalMinutes != 180 {
			t.Fatalf("PageCompaniesForUser(...) = %+v, %v, want one tracked company", page, err)
		}
	})

	t.Run("paging companies by offset visits each once in name then id order and reports the total", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		for _, c := range []dto.CompanyUpsert{
			{Slug: "delta", Name: "Delta"}, {Slug: "alpha", Name: "Alpha"}, {Slug: "twin-a", Name: "Twin"},
			{Slug: "twin-b", Name: "Twin"}, {Slug: "omega", Name: "Omega"},
		} {
			if _, err := st.UpsertCompany(ctx, c); err != nil {
				t.Fatalf("UpsertCompany(%q) = %v", c.Slug, err)
			}
		}
		var got []dto.Company
		options := dto.CompanyPageOptions{Limit: 2}
		for {
			page, err := st.PageCompaniesForUser(ctx, userID, options)
			if err != nil {
				t.Fatalf("PageCompaniesForUser(%+v) = %v", options, err)
			}
			if page.Total != 5 {
				t.Fatalf("PageCompaniesForUser(%+v).Total = %d, want 5", options, page.Total)
			}
			got = append(got, page.Items...)
			if len(page.Items) < int(options.Limit) {
				break
			}
			options.Offset += options.Limit
		}
		names := make([]string, len(got))
		ids := map[string]bool{}
		for i, c := range got {
			names[i] = c.Name
			ids[c.ID] = true
		}
		if want := []string{"Alpha", "Delta", "Omega", "Twin", "Twin"}; !slices.Equal(names, want) || len(ids) != 5 || got[3].ID >= got[4].ID {
			t.Fatalf("paged = %v (%d distinct), want %v with the Twin tie broken by id", names, len(ids), want)
		}
	})

	t.Run("relevance puts favourites first, then tracked companies, alphabetical ignores both", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		ids := map[string]string{}
		for _, name := range []string{"Alpha", "Beta", "Gamma", "Delta"} {
			c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: strings.ToLower(name), Name: name})
			if err != nil {
				t.Fatalf("UpsertCompany(%q) = %v", name, err)
			}
			ids[name] = c.ID
		}
		for _, name := range []string{"Beta", "Gamma"} {
			if _, err := st.SetCompanyTracking(ctx, userID, ids[name], true, 180); err != nil {
				t.Fatalf("SetCompanyTracking(%q) = %v", name, err)
			}
		}
		for _, name := range []string{"Gamma", "Delta"} {
			verifyBoard(t, st, ids[name], "greenhouse")
			if err := st.SetCompanyFavourite(ctx, userID, ids[name], true); err != nil {
				t.Fatalf("SetCompanyFavourite(%q) = %v", name, err)
			}
		}
		names := func(sort dto.CompanySort) []string {
			page, err := st.PageCompaniesForUser(ctx, userID, dto.CompanyPageOptions{Limit: 10, Sort: sort})
			if err != nil {
				t.Fatalf("PageCompaniesForUser(%q) = %v", sort, err)
			}
			var out []string
			for _, c := range page.Items {
				out = append(out, c.Name)
			}
			return out
		}
		if got, want := names(dto.CompanySortRelevance), []string{"Delta", "Gamma", "Beta", "Alpha"}; !slices.Equal(got, want) {
			t.Errorf("relevance = %v, want %v (favourites lead tracked companies)", got, want)
		}
		if got, want := names(dto.CompanySortOpenJobs), []string{"Alpha", "Beta", "Delta", "Gamma"}; !slices.Equal(got, want) {
			t.Errorf("open jobs = %v, want %v (no jobs, so ties fall back to name)", got, want)
		}
		if got, want := names(dto.CompanySortAlphabetical), []string{"Alpha", "Beta", "Delta", "Gamma"}; !slices.Equal(got, want) {
			t.Errorf("alphabetical = %v, want %v", got, want)
		}
	})

	t.Run("no-board filter keeps companies without any board and counts only them", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		withBoard, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "with-board", Name: "With Board"})
		if err != nil {
			t.Fatalf("UpsertCompany(...) = %v", err)
		}
		if _, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "bare", Name: "Bare"}); err != nil {
			t.Fatalf("UpsertCompany(...) = %v", err)
		}
		if _, err := st.UpsertCandidateBoard(ctx, withBoard.ID, "greenhouse", "with-board"); err != nil {
			t.Fatalf("UpsertCandidateBoard(...) = %v", err)
		}
		page, err := st.PageCompaniesForUser(ctx, userID, dto.CompanyPageOptions{Limit: 10, NoBoardOnly: true})
		if err != nil || len(page.Items) != 1 || page.Items[0].Slug != "bare" || page.Total != 1 {
			t.Fatalf("PageCompaniesForUser(no board) = %+v, %v, want only bare with total 1", page, err)
		}
	})

	t.Run("ats filter and counts consider verified boards only and skip wttj", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		for _, b := range []struct{ slug, source string }{
			{"gh-co", "greenhouse"}, {"gh-co", "lever"}, {"lever-co", "lever"}, {"wttj-co", "wttj"},
		} {
			c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: b.slug, Name: b.slug})
			if err != nil {
				t.Fatalf("UpsertCompany(%q) = %v", b.slug, err)
			}
			if _, err := st.UpsertCandidateBoard(ctx, c.ID, b.source, b.slug); err != nil {
				t.Fatalf("UpsertCandidateBoard(%q, %q) = %v", b.slug, b.source, err)
			}
			if _, err := st.VerifyCompanyBoard(ctx, c.ID, b.source, b.slug, "test", "test"); err != nil {
				t.Fatalf("VerifyCompanyBoard(%q, %q) = %v", b.slug, b.source, err)
			}
		}
		candidate, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "candidate-co", Name: "candidate-co"})
		if err != nil {
			t.Fatalf("UpsertCompany(candidate) = %v", err)
		}
		if _, err := st.UpsertCandidateBoard(ctx, candidate.ID, "greenhouse", "candidate-co"); err != nil {
			t.Fatalf("UpsertCandidateBoard(candidate) = %v", err)
		}

		counts, err := st.CountCompaniesByATS(ctx)
		want := []dto.ATSCount{{Source: "greenhouse", Companies: 1}, {Source: "lever", Companies: 2}}
		if err != nil || !slices.Equal(counts, want) {
			t.Fatalf("CountCompaniesByATS() = %+v, %v, want %+v", counts, err, want)
		}

		page, err := st.PageCompaniesForUser(ctx, userID, dto.CompanyPageOptions{Limit: 10, ATSSources: []string{"greenhouse"}})
		if err != nil || len(page.Items) != 1 || page.Items[0].Slug != "gh-co" || page.Total != 1 {
			t.Fatalf("PageCompaniesForUser(ats greenhouse) = %+v, %v, want only gh-co", page, err)
		}
		page, err = st.PageCompaniesForUser(ctx, userID, dto.CompanyPageOptions{Limit: 10, ATSSources: []string{"greenhouse", "lever"}})
		if err != nil || page.Total != 2 {
			t.Fatalf("PageCompaniesForUser(ats greenhouse+lever) = %+v, %v, want gh-co and lever-co once each", page, err)
		}
	})

	t.Run("board coverage counts verified non-WTTJ companies regardless of filters", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		for _, slug := range []string{"verified", "wttj-only", "candidate", "bare"} {
			if _, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: slug, Name: slug}); err != nil {
				t.Fatalf("UpsertCompany(%q) = %v", slug, err)
			}
		}
		page, err := st.PageCompaniesForUser(ctx, userID, dto.CompanyPageOptions{Limit: 10})
		if err != nil {
			t.Fatalf("PageCompaniesForUser(...) = %v", err)
		}
		ids := map[string]string{}
		for _, c := range page.Items {
			ids[c.Slug] = c.ID
		}
		verifyBoard(t, st, ids["verified"], "greenhouse")
		verifyBoard(t, st, ids["wttj-only"], "wttj")
		if _, err := st.UpsertCandidateBoard(ctx, ids["candidate"], "greenhouse", "candidate"); err != nil {
			t.Fatalf("UpsertCandidateBoard(...) = %v", err)
		}
		page, err = st.PageCompaniesForUser(ctx, userID, dto.CompanyPageOptions{Limit: 10, Search: "bare"})
		if err != nil || page.Total != 1 || page.AllCompanies != 4 || page.VerifiedBoardCompanies != 1 {
			t.Fatalf("PageCompaniesForUser(search bare) = %+v, %v, want total 1, 1 of 4 verified", page, err)
		}
	})

	t.Run("poster filter splits agency and likely agency from everything else", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		kinds := map[string]dto.CompanyKind{
			"agency-co": dto.KindAgency, "likely-agency-co": dto.KindLikelyAgency,
			"direct-co": dto.KindDirect, "likely-direct-co": dto.KindLikelyDirect, "plain-co": dto.KindUnclassified,
		}
		for slug, kind := range kinds {
			c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: slug, Name: slug})
			if err != nil {
				t.Fatalf("UpsertCompany(%q) = %v", slug, err)
			}
			if kind == dto.KindUnclassified {
				continue
			}
			if err := st.SetCompanyKind(ctx, c.ID, kind); err != nil {
				t.Fatalf("SetCompanyKind(%q) = %v", slug, err)
			}
		}
		for poster, want := range map[dto.Poster][]string{
			dto.PosterAgency: {"agency-co", "likely-agency-co"},
			dto.PosterDirect: {"direct-co", "likely-direct-co", "plain-co"},
		} {
			page, err := st.PageCompaniesForUser(ctx, userID, dto.CompanyPageOptions{Limit: 10, Poster: poster})
			if err != nil {
				t.Fatalf("PageCompaniesForUser(%q) = %v", poster, err)
			}
			var got []string
			for _, c := range page.Items {
				got = append(got, c.Slug)
			}
			if diff := cmp.Diff(want, got); diff != "" || page.Total != len(want) {
				t.Errorf("PageCompaniesForUser(%q) slugs (-want +got):\n%s\ntotal = %d", poster, diff, page.Total)
			}
		}
	})

	t.Run("companies search and tracked filters compose with the offset", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		var acme dto.Company
		for _, c := range []dto.CompanyUpsert{{Slug: "acme", Name: "Acme"}, {Slug: "acme-labs", Name: "Beta Works"}, {Slug: "zeta", Name: "Zeta Acme"}} {
			created, err := st.UpsertCompany(ctx, c)
			if err != nil {
				t.Fatalf("UpsertCompany(%q) = %v", c.Slug, err)
			}
			if c.Slug == "acme" {
				acme = created
			}
		}
		if _, err := st.SetCompanyTracking(ctx, userID, acme.ID, true, 180); err != nil {
			t.Fatalf("SetCompanyTracking(...) = %v", err)
		}
		slugs := func(options dto.CompanyPageOptions) []string {
			page, err := st.PageCompaniesForUser(ctx, userID, options)
			if err != nil {
				t.Fatalf("PageCompaniesForUser(%+v) = %v", options, err)
			}
			var out []string
			for _, c := range page.Items {
				out = append(out, c.Slug)
			}
			return out
		}
		if got, want := slugs(dto.CompanyPageOptions{Limit: 10, Search: "ACME"}), []string{"acme", "acme-labs", "zeta"}; !slices.Equal(got, want) {
			t.Errorf("search ACME = %v, want %v (name or slug, case-insensitive)", got, want)
		}
		if got, want := slugs(dto.CompanyPageOptions{Limit: 10, Search: "acme", Offset: 1}), []string{"acme-labs", "zeta"}; !slices.Equal(got, want) {
			t.Errorf("search acme from offset 1 = %v, want %v", got, want)
		}
		if got, want := slugs(dto.CompanyPageOptions{Limit: 10, TrackedOnly: true}), []string{"acme"}; !slices.Equal(got, want) {
			t.Errorf("tracked only = %v, want %v", got, want)
		}
	})

	t.Run("list tracked companies returns only this user's, paused included, with boards", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		active, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "active-co", Name: "Active Co"})
		if err != nil {
			t.Fatalf("UpsertCompany(...) = %v", err)
		}
		paused, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "paused-co", Name: "Paused Co"})
		if err != nil {
			t.Fatalf("UpsertCompany(...) = %v", err)
		}
		if _, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "untracked-co", Name: "Untracked Co"}); err != nil {
			t.Fatalf("UpsertCompany(...) = %v", err)
		}
		board, err := st.UpsertCandidateBoard(ctx, active.ID, "greenhouse", "active-co")
		if err != nil {
			t.Fatalf("UpsertCandidateBoard(...) = %v", err)
		}
		if _, err := st.SetCompanyTracking(ctx, userID, active.ID, true, 180); err != nil {
			t.Fatalf("SetCompanyTracking(...) = %v", err)
		}
		if _, err := st.SetCompanyTracking(ctx, userID, paused.ID, false, 0); err != nil {
			t.Fatalf("SetCompanyTracking(...) = %v", err)
		}

		got, err := st.ListTrackedCompaniesForUser(ctx, userID)
		if err != nil || len(got) != 2 {
			t.Fatalf("ListTrackedCompaniesForUser(...) = %+v, %v, want two companies", got, err)
		}
		if got[0].ID != active.ID || !got[0].Enabled || got[0].CheckIntervalMinutes != 180 ||
			len(got[0].Boards) != 1 || got[0].Boards[0].ID != board.ID || got[0].Boards[0].Status != dto.BoardCandidate {
			t.Fatalf("active company = %+v, want enabled at 180 with its board", got[0])
		}
		if got[1].ID != paused.ID || got[1].Enabled || got[1].Boards == nil || len(got[1].Boards) != 0 {
			t.Fatalf("paused company = %+v, want disabled with empty boards", got[1])
		}

		other, err := st.ListTrackedCompaniesForUser(ctx, missingID)
		if err != nil || len(other) != 0 {
			t.Fatalf("ListTrackedCompaniesForUser(other user) = %+v, %v, want none", other, err)
		}
	})

	t.Run("verifying a board records how it was discovered once", func(t *testing.T) {
		t.Parallel()
		st, _ := newStore(t)
		ctx := t.Context()
		c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "via-co", Name: "Via Co"})
		if err != nil {
			t.Fatalf("UpsertCompany(...) = %v", err)
		}
		if _, err := st.UpsertCandidateBoard(ctx, c.ID, "greenhouse", "via-co"); err != nil {
			t.Fatalf("UpsertCandidateBoard(...) = %v", err)
		}
		board, err := st.VerifyCompanyBoard(ctx, c.ID, "greenhouse", "via-co", "discovered", "linkedin")
		if err != nil || board.DiscoveredVia != "linkedin" {
			t.Fatalf("VerifyCompanyBoard(via linkedin) = %+v, %v, want DiscoveredVia linkedin", board, err)
		}
		if _, err := st.VerifyCompanyBoard(ctx, c.ID, "greenhouse", "via-co", "discovered", "wttj"); !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("re-VerifyCompanyBoard err = %v, want ErrNotFound", err)
		}
		boards, err := st.ListCompanyBoards(ctx, c.ID)
		if err != nil || len(boards) != 1 || boards[0].DiscoveredVia != "linkedin" {
			t.Fatalf("ListCompanyBoards() = %+v, %v, want DiscoveredVia linkedin", boards, err)
		}
	})

	t.Run("completing a board stores the reported and parsed counts", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "count-co", Name: "Count Co"})
		if err != nil {
			t.Fatalf("UpsertCompany(...) = %v", err)
		}
		board, err := st.UpsertCandidateBoard(ctx, c.ID, "greenhouse", "count-co")
		if err != nil {
			t.Fatalf("UpsertCandidateBoard(...) = %v", err)
		}
		if _, err := st.SetCompanyTracking(ctx, userID, c.ID, true, 60); err != nil {
			t.Fatalf("SetCompanyTracking(...) = %v", err)
		}
		if _, err := st.VerifyCompanyBoard(ctx, c.ID, "greenhouse", "count-co", "test", ""); err != nil {
			t.Fatalf("VerifyCompanyBoard(...) = %v", err)
		}
		job, _, err := st.SaveCanonical(ctx, dto.Job{
			Title: "Engineer", URL: "https://boards.greenhouse.io/count-co/jobs/1", Source: "greenhouse", CompanySlug: "count-co",
			CompanyID: c.ID, BoardID: board.ID, ProviderPostingID: "1",
		})
		if err != nil {
			t.Fatalf("SaveCanonical(...) = %v", err)
		}
		poll, err := st.ClaimBoard(ctx, board.ID, false)
		if err != nil {
			t.Fatalf("ClaimBoard(...) = %v", err)
		}
		if err := st.CompleteBoard(ctx, dto.BoardSnapshot{Poll: poll, Complete: true, Jobs: []dto.Job{job}, Reported: 5}); err != nil {
			t.Fatalf("CompleteBoard(...) = %v", err)
		}
		boards, err := st.ListCompanyBoards(ctx, c.ID)
		if err != nil || len(boards) != 1 || boards[0].LastReportedTotal != 5 || boards[0].LastParsed != 1 {
			t.Fatalf("ListCompanyBoards() = %+v, %v, want LastReportedTotal 5 and LastParsed 1", boards, err)
		}
	})

	t.Run("a published board check holds the board back until released", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "held-co", Name: "Held Co"})
		if err != nil {
			t.Fatalf("UpsertCompany(...) = %v", err)
		}
		board, err := st.UpsertCandidateBoard(ctx, c.ID, "greenhouse", "held-co")
		if err != nil {
			t.Fatalf("UpsertCandidateBoard(...) = %v", err)
		}
		if _, err := st.SetCompanyTracking(ctx, userID, c.ID, true, 60); err != nil {
			t.Fatalf("SetCompanyTracking(...) = %v", err)
		}
		if _, err := st.VerifyCompanyBoard(ctx, c.ID, "greenhouse", "held-co", "test", ""); err != nil {
			t.Fatalf("VerifyCompanyBoard(...) = %v", err)
		}
		dueIDs := func() []string {
			t.Helper()
			due, err := st.ListDueBoards(ctx)
			if err != nil {
				t.Fatalf("ListDueBoards() = %v", err)
			}
			ids := make([]string, 0, len(due))
			for _, b := range due {
				ids = append(ids, b.ID)
			}
			return ids
		}

		if err := st.MarkBoardCheckPublished(ctx, board.ID); err != nil {
			t.Fatalf("MarkBoardCheckPublished(...) = %v", err)
		}
		if got := dueIDs(); len(got) != 0 {
			t.Errorf("ListDueBoards() after publish = %v, want none", got)
		}
		if err := st.ClearBoardCheckPublished(ctx, board.ID); err != nil {
			t.Fatalf("ClearBoardCheckPublished(...) = %v", err)
		}
		if got := dueIDs(); !slices.Equal(got, []string{board.ID}) {
			t.Errorf("ListDueBoards() after release = %v, want [%s]", got, board.ID)
		}
	})

	t.Run("verified boards by slug skip candidates and flag enabled trackers and completed polls", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		boardIDs := map[string]string{}
		for _, tc := range []struct {
			slug     string
			verified bool
			enabled  bool
		}{{"polled-co", true, true}, {"candidate-co", false, true}, {"paused-co", true, false}} {
			c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: tc.slug, Name: tc.slug})
			if err != nil {
				t.Fatalf("UpsertCompany(%s) = %v", tc.slug, err)
			}
			board, err := st.UpsertCandidateBoard(ctx, c.ID, "greenhouse", tc.slug)
			if err != nil {
				t.Fatalf("UpsertCandidateBoard(%s) = %v", tc.slug, err)
			}
			boardIDs[tc.slug] = board.ID
			if tc.verified {
				if _, err := st.VerifyCompanyBoard(ctx, c.ID, "greenhouse", tc.slug, "test", ""); err != nil {
					t.Fatalf("VerifyCompanyBoard(%s) = %v", tc.slug, err)
				}
			}
			if _, err := st.SetCompanyTracking(ctx, userID, c.ID, tc.enabled, 0); err != nil {
				t.Fatalf("SetCompanyTracking(%s) = %v", tc.slug, err)
			}
		}
		poll, err := st.ClaimBoard(ctx, boardIDs["polled-co"], false)
		if err != nil {
			t.Fatalf("ClaimBoard(polled-co) = %v", err)
		}
		if err := st.CompleteBoard(ctx, dto.BoardSnapshot{Poll: poll, Complete: true}); err != nil {
			t.Fatalf("CompleteBoard(polled-co) = %v", err)
		}
		got, err := st.ListVerifiedBoardsBySlug(ctx, []string{"polled-co", "candidate-co", "paused-co", "unknown-co"})
		want := []dto.CardBoard{
			{CompanySlug: "paused-co", Source: "greenhouse", BoardToken: "paused-co"},
			{CompanySlug: "polled-co", Source: "greenhouse", BoardToken: "polled-co", Tracked: true, Polled: true},
		}
		slices.SortFunc(got, func(a, b dto.CardBoard) int { return strings.Compare(a.CompanySlug, b.CompanySlug) })
		if err != nil || !cmp.Equal(got, want) {
			t.Fatalf("ListVerifiedBoardsBySlug(...) = %v, %v, want %v", got, err, want)
		}
	})

	t.Run("verified company slugs need a verified board", func(t *testing.T) {
		t.Parallel()
		st, _ := newStore(t)
		ctx := t.Context()
		for _, tc := range []struct {
			slug     string
			verified bool
		}{{"verified-co", true}, {"candidate-co", false}} {
			c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: tc.slug, Name: tc.slug})
			if err != nil {
				t.Fatalf("UpsertCompany(%s) = %v", tc.slug, err)
			}
			if _, err := st.UpsertCandidateBoard(ctx, c.ID, "greenhouse", tc.slug); err != nil {
				t.Fatalf("UpsertCandidateBoard(%s) = %v", tc.slug, err)
			}
			if tc.verified {
				if _, err := st.VerifyCompanyBoard(ctx, c.ID, "greenhouse", tc.slug, "test", ""); err != nil {
					t.Fatalf("VerifyCompanyBoard(%s) = %v", tc.slug, err)
				}
			}
		}
		got, err := st.ListVerifiedCompanySlugs(ctx, []string{"verified-co", "candidate-co", "unknown-co"})
		if want := []string{"verified-co"}; err != nil || !cmp.Equal(got, want) {
			t.Fatalf("ListVerifiedCompanySlugs(...) = %v, %v, want %v", got, err, want)
		}
	})

	t.Run("delete company tracking removes it and keeps the company", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "untrack-co", Name: "Untrack Co"})
		if err != nil {
			t.Fatalf("UpsertCompany(...) = %v", err)
		}
		if _, err := st.SetCompanyTracking(ctx, userID, c.ID, true, 180); err != nil {
			t.Fatalf("SetCompanyTracking(...) = %v", err)
		}
		if err := st.DeleteCompanyTracking(ctx, userID, c.ID); err != nil {
			t.Fatalf("DeleteCompanyTracking(...) = %v", err)
		}
		tracked, err := st.ListTrackedCompaniesForUser(ctx, userID)
		if err != nil || len(tracked) != 0 {
			t.Fatalf("ListTrackedCompaniesForUser(...) = %+v, %v, want none", tracked, err)
		}
		if _, err := st.GetCompany(ctx, c.ID); err != nil {
			t.Fatalf("GetCompany(...) = %v, want the company kept", err)
		}
		if err := st.DeleteCompanyTracking(ctx, userID, c.ID); !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("second DeleteCompanyTracking(...) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("dismiss keeps the row disabled, undo re-enables it as new", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "review-co", Name: "Review Co"})
		if err != nil {
			t.Fatalf("UpsertCompany(...) = %v", err)
		}
		if _, err := st.SetCompanyReviewState(ctx, userID, c.ID, "kept"); !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("SetCompanyReviewState(untracked) err = %v, want ErrNotFound", err)
		}
		if _, err := st.SetCompanyTracking(ctx, userID, c.ID, true, 180); err != nil {
			t.Fatalf("SetCompanyTracking(...) = %v", err)
		}
		for _, tt := range []struct {
			state       string
			wantEnabled bool
		}{{"dismissed", false}, {"new", true}} {
			if _, err := st.SetCompanyReviewState(ctx, userID, c.ID, tt.state); err != nil {
				t.Fatalf("SetCompanyReviewState(%s) = %v", tt.state, err)
			}
			tracked, err := st.ListTrackedCompaniesForUser(ctx, userID)
			if err != nil || len(tracked) != 1 || tracked[0].ReviewState != tt.state || tracked[0].Enabled != tt.wantEnabled {
				t.Fatalf("ListTrackedCompaniesForUser(...) = %+v, %v, want one %s company enabled=%v", tracked, err, tt.state, tt.wantEnabled)
			}
		}
	})

	t.Run("track discovered company inserts as new once and never overwrites", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "found-co", Name: "Found Co"})
		if err != nil {
			t.Fatalf("UpsertCompany(...) = %v", err)
		}
		if tracked, err := st.TrackDiscoveredCompany(ctx, userID, c.ID); err != nil || !tracked {
			t.Fatalf("TrackDiscoveredCompany(...) = %v, %v, want true", tracked, err)
		}
		if _, err := st.SetCompanyReviewState(ctx, userID, c.ID, "dismissed"); err != nil {
			t.Fatalf("SetCompanyReviewState(...) = %v", err)
		}
		if tracked, err := st.TrackDiscoveredCompany(ctx, userID, c.ID); err != nil || tracked {
			t.Fatalf("second TrackDiscoveredCompany(...) = %v, %v, want false", tracked, err)
		}
		got, err := st.ListTrackedCompaniesForUser(ctx, userID)
		if err != nil || len(got) != 1 || got[0].ReviewState != "dismissed" || got[0].Enabled {
			t.Fatalf("ListTrackedCompaniesForUser(...) = %+v, %v, want one dismissed disabled company", got, err)
		}
	})

	t.Run("favouriting an untracked company stars and tracks it hourly, unfavouriting goes back to six hours", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "star-co", Name: "Star Co"})
		if err != nil {
			t.Fatalf("UpsertCompany(...) = %v", err)
		}
		verifyBoard(t, st, c.ID, "greenhouse")
		job, _, err := st.SaveCanonical(ctx, dto.Job{Title: "Engineer", URL: "https://example.com/jobs/star", CompanySlug: "star-co"})
		if err != nil {
			t.Fatalf("SaveCanonical(...) = %v", err)
		}
		for range 2 {
			if err := st.SetCompanyFavourite(ctx, userID, c.ID, true); err != nil {
				t.Fatalf("SetCompanyFavourite(true) = %v", err)
			}
		}
		got, err := st.GetCompanyForUser(ctx, userID, c.ID)
		if err != nil || !got.Favourite || !got.Tracked || got.CheckIntervalMinutes != 60 {
			t.Fatalf("GetCompanyForUser(...) = %+v, %v, want favourite, tracked, every 60 minutes", got, err)
		}
		gotJob, err := st.GetJob(ctx, job.ID, userID)
		if err != nil || !gotJob.CompanyFavourite {
			t.Fatalf("GetJob(...).CompanyFavourite = %v, %v, want true", gotJob.CompanyFavourite, err)
		}
		page, err := st.PageCompaniesForUser(ctx, userID, dto.CompanyPageOptions{Limit: 10, FavouriteOnly: true})
		if err != nil || len(page.Items) != 1 || page.Items[0].ID != c.ID {
			t.Fatalf("PageCompaniesForUser(favourite only) = %+v, %v, want Star Co", page, err)
		}
		for range 2 {
			if err := st.SetCompanyFavourite(ctx, userID, c.ID, false); err != nil {
				t.Fatalf("SetCompanyFavourite(false) = %v", err)
			}
		}
		got, err = st.GetCompanyForUser(ctx, userID, c.ID)
		if err != nil || got.Favourite || !got.Tracked || got.CheckIntervalMinutes != 360 {
			t.Fatalf("GetCompanyForUser(...) after unstar = %+v, %v, want unfavourited, tracked, every 360 minutes", got, err)
		}
		page, err = st.PageCompaniesForUser(ctx, userID, dto.CompanyPageOptions{Limit: 10, FavouriteOnly: true})
		if err != nil || len(page.Items) != 0 {
			t.Fatalf("PageCompaniesForUser(favourite only) after unstar = %+v, %v, want none", page, err)
		}
		if _, err := st.SetCompanyReviewState(ctx, userID, c.ID, "dismissed"); err != nil {
			t.Fatalf("SetCompanyReviewState(dismissed) = %v", err)
		}
		if err := st.SetCompanyFavourite(ctx, userID, c.ID, true); err != nil {
			t.Fatalf("SetCompanyFavourite(true) = %v", err)
		}
		got, err = st.GetCompanyForUser(ctx, userID, c.ID)
		if err != nil || !got.Tracked || got.ReviewState != "kept" {
			t.Fatalf("GetCompanyForUser(...) after favouriting a dismissed company = %+v, %v, want tracked and kept", got, err)
		}
	})

	t.Run("favouriting a company without a verified ats board stars it and leaves tracking alone", func(t *testing.T) {
		st, userID := newStore(t)
		ctx := t.Context()
		for _, tc := range []struct{ slug, source, board string }{
			{"no-board-co", "", ""},
			{"candidate-co", "greenhouse", "candidate"},
			{"discovery-co", "linkedin", "verified"},
		} {
			c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: tc.slug, Name: tc.slug})
			if err != nil {
				t.Fatalf("UpsertCompany(%q) = %v", tc.slug, err)
			}
			switch tc.board {
			case "candidate":
				if _, err := st.UpsertCandidateBoard(ctx, c.ID, tc.source, tc.slug); err != nil {
					t.Fatalf("UpsertCandidateBoard(%q) = %v", tc.slug, err)
				}
			case "verified":
				verifyBoard(t, st, c.ID, tc.source)
			}
			if err := st.SetCompanyFavourite(ctx, userID, c.ID, true); err != nil {
				t.Fatalf("SetCompanyFavourite(%q) = %v", tc.slug, err)
			}
			got, err := st.GetCompanyForUser(ctx, userID, c.ID)
			if err != nil || !got.Favourite || got.Tracked {
				t.Fatalf("GetCompanyForUser(%q) = %+v, %v, want favourite and untracked", tc.slug, got, err)
			}
		}
	})

	t.Run("an upsert kind overrides a likely kind, never agency, and an empty kind keeps it", func(t *testing.T) {
		t.Parallel()
		st, _ := newStore(t)
		ctx := t.Context()
		c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "ats-co", Name: "Ats Co", Kind: dto.KindDirect})
		if err != nil || c.Kind != dto.KindDirect {
			t.Fatalf("UpsertCompany(direct) = %+v, %v, want direct", c, err)
		}
		if err := st.SetCompanyKind(ctx, c.ID, dto.KindLikelyAgency); err != nil {
			t.Fatalf("SetCompanyKind(likely_agency) = %v", err)
		}
		if c, err = st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "ats-co", Name: "Ats Co"}); err != nil || c.Kind != dto.KindLikelyAgency {
			t.Fatalf("UpsertCompany(empty) = %+v, %v, want likely_agency kept", c, err)
		}
		if c, err = st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "ats-co", Name: "Ats Co", Kind: dto.KindDirect}); err != nil || c.Kind != dto.KindDirect {
			t.Fatalf("UpsertCompany(direct) = %+v, %v, want direct", c, err)
		}
		if err := st.SetCompanyKind(ctx, c.ID, dto.KindAgency); err != nil {
			t.Fatalf("SetCompanyKind(agency) = %v", err)
		}
		if c, err = st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "ats-co", Name: "Ats Co", Kind: dto.KindDirect}); err != nil || c.Kind != dto.KindAgency {
			t.Fatalf("UpsertCompany(direct) = %+v, %v, want agency kept", c, err)
		}
	})

	t.Run("a company starts unclassified and its kind shows on the company and its jobs", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "kind-co", Name: "Kind Co"})
		if err != nil || c.Kind != dto.KindUnclassified {
			t.Fatalf("UpsertCompany(...) = %+v, %v, want unclassified", c, err)
		}
		job, _, err := st.SaveCanonical(ctx, dto.Job{Title: "Engineer", URL: "https://example.com/jobs/kind", CompanySlug: "kind-co"})
		if err != nil {
			t.Fatalf("SaveCanonical(...) = %v", err)
		}
		if err := st.SetCompanyKind(ctx, c.ID, dto.KindAgency); err != nil {
			t.Fatalf("SetCompanyKind(agency) = %v", err)
		}
		got, err := st.GetCompanyForUser(ctx, userID, c.ID)
		if err != nil || got.Kind != dto.KindAgency {
			t.Fatalf("GetCompanyForUser(...) = %+v, %v, want agency", got, err)
		}
		gotJob, err := st.GetJob(ctx, job.ID, userID)
		if err != nil || gotJob.CompanyKind != dto.KindAgency {
			t.Fatalf("GetJob(...).CompanyKind = %q, %v, want agency", gotJob.CompanyKind, err)
		}
		if err := st.SetCompanyKind(ctx, c.ID, dto.KindDirect); err != nil {
			t.Fatalf("SetCompanyKind(direct) = %v", err)
		}
		page, err := st.Page(ctx, userID, dto.JobPageOptions{Limit: 10})
		if err != nil || len(page.Items) != 1 || page.Items[0].CompanyKind != dto.KindDirect {
			t.Fatalf("Page(...) = %+v, %v, want one direct job", page, err)
		}
		if err := st.SetCompanyKind(ctx, "00000000-0000-0000-0000-000000000000", dto.KindAgency); !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("SetCompanyKind(unknown) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("suggesting a company kind writes only over unclassified", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "suggest-co", Name: "Suggest Co"})
		if err != nil {
			t.Fatalf("UpsertCompany(...) = %v", err)
		}
		wrote, err := st.SuggestCompanyKind(ctx, c.ID, dto.KindLikelyAgency)
		if err != nil || !wrote {
			t.Fatalf("SuggestCompanyKind(first) = %v, %v, want true", wrote, err)
		}
		wrote, err = st.SuggestCompanyKind(ctx, c.ID, dto.KindLikelyDirect)
		if err != nil || wrote {
			t.Fatalf("SuggestCompanyKind(second) = %v, %v, want false", wrote, err)
		}
		got, err := st.GetCompanyForUser(ctx, userID, c.ID)
		if err != nil || got.Kind != dto.KindLikelyAgency {
			t.Fatalf("GetCompanyForUser(...) = %+v, %v, want likely_agency", got, err)
		}
		if err := st.SetCompanyKind(ctx, c.ID, dto.KindDirect); err != nil {
			t.Fatalf("SetCompanyKind(direct) = %v", err)
		}
		if wrote, err = st.SuggestCompanyKind(ctx, c.ID, dto.KindLikelyAgency); err != nil || wrote {
			t.Fatalf("SuggestCompanyKind(over direct) = %v, %v, want false", wrote, err)
		}
	})

	t.Run("favouriting a new tracked company keeps it, and dismissing it removes the star", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "found-co", Name: "Found Co"})
		if err != nil {
			t.Fatalf("UpsertCompany(...) = %v", err)
		}
		if _, err := st.TrackDiscoveredCompany(ctx, userID, c.ID); err != nil {
			t.Fatalf("TrackDiscoveredCompany(...) = %v", err)
		}
		verifyBoard(t, st, c.ID, "greenhouse")
		if err := st.SetCompanyFavourite(ctx, userID, c.ID, true); err != nil {
			t.Fatalf("SetCompanyFavourite(true) = %v", err)
		}
		got, err := st.GetCompanyForUser(ctx, userID, c.ID)
		if err != nil || got.ReviewState != "kept" || !got.Favourite {
			t.Fatalf("GetCompanyForUser(...) = %+v, %v, want kept and favourite", got, err)
		}
		if _, err := st.SetCompanyReviewState(ctx, userID, c.ID, "dismissed"); err != nil {
			t.Fatalf("SetCompanyReviewState(dismissed) = %v", err)
		}
		got, err = st.GetCompanyForUser(ctx, userID, c.ID)
		if err != nil || got.Favourite {
			t.Fatalf("GetCompanyForUser(...) after dismiss = %+v, %v, want not favourite", got, err)
		}
	})

	t.Run("untracked discovered boards exclude tracked, dismissed, candidate and user-confirmed ones", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		cases := []struct {
			slug, method string
			verify       bool
			track        bool
		}{
			{"untracked-co", "discovered", true, false},
			{"tracked-co", "discovered", true, true},
			{"candidate-co", "", false, false},
			{"confirmed-co", "user_confirmed", true, false},
		}
		for _, tc := range cases {
			c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: tc.slug, Name: tc.slug})
			if err != nil {
				t.Fatalf("UpsertCompany(%s) = %v", tc.slug, err)
			}
			if _, err := st.UpsertCandidateBoard(ctx, c.ID, "greenhouse", tc.slug); err != nil {
				t.Fatalf("UpsertCandidateBoard(%s) = %v", tc.slug, err)
			}
			if tc.verify {
				if _, err := st.VerifyCompanyBoard(ctx, c.ID, "greenhouse", tc.slug, tc.method, ""); err != nil {
					t.Fatalf("VerifyCompanyBoard(%s) = %v", tc.slug, err)
				}
			}
			if tc.track {
				if _, err := st.TrackDiscoveredCompany(ctx, userID, c.ID); err != nil {
					t.Fatalf("TrackDiscoveredCompany(%s) = %v", tc.slug, err)
				}
			}
		}
		got, err := st.ListUntrackedDiscoveredBoards(ctx, false, time.Now().Add(-30*24*time.Hour))
		if err != nil || len(got) != 1 || got[0].BoardToken != "untracked-co" {
			t.Fatalf("ListUntrackedDiscoveredBoards() = %+v, %v, want only untracked-co", got, err)
		}
	})

	t.Run("set company tracking without an interval keeps the existing one", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "keep-co", Name: "Keep Co"})
		if err != nil {
			t.Fatalf("UpsertCompany(...) = %v", err)
		}
		if _, err := st.SetCompanyTracking(ctx, userID, c.ID, true, 180); err != nil {
			t.Fatalf("SetCompanyTracking(...) = %v", err)
		}
		got, err := st.SetCompanyTracking(ctx, userID, c.ID, false, 0)
		if err != nil || got.Enabled || got.CheckIntervalMinutes != 180 {
			t.Fatalf("SetCompanyTracking(..., false, 0) = %+v, %v, want disabled at 180", got, err)
		}
	})

	t.Run("saving a profile twice keeps one, shown on the new company card", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "profiled", Name: "Profiled"})
		if err != nil {
			t.Fatalf("UpsertCompany(...) = %v", err)
		}
		if _, err := st.SetCompanyTracking(ctx, userID, c.ID, true, 0); err != nil {
			t.Fatalf("SetCompanyTracking(...) = %v", err)
		}
		if _, err := st.SetCompanyReviewState(ctx, userID, c.ID, "new"); err != nil {
			t.Fatalf("SetCompanyReviewState(...) = %v", err)
		}
		got, err := st.ListNewCompanies(ctx, userID)
		if err != nil || len(got) != 1 || got[0].Profile != nil {
			t.Fatalf("ListNewCompanies(no profile) = %+v, %v, want nil profile", got, err)
		}
		for _, hq := range []string{"Paris", "London"} {
			if err := st.SaveCompanyProfile(ctx, c.ID, "wttj", dto.CompanyProfile{HQ: hq, Sectors: []string{"fintech"}}); err != nil {
				t.Fatalf("SaveCompanyProfile(%s) = %v", hq, err)
			}
		}
		got, err = st.ListNewCompanies(ctx, userID)
		if err != nil || len(got) != 1 || got[0].Profile == nil || got[0].Profile.HQ != "London" {
			t.Fatalf("ListNewCompanies(profiled) = %+v, %v, want the London profile", got, err)
		}
	})

	t.Run("new companies list only review state new, newest first, with their open jobs", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		var ids []string
		for _, slug := range []string{"older-co", "newer-co", "kept-co"} {
			c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: slug, Name: slug})
			if err != nil {
				t.Fatalf("UpsertCompany(%s) = %v", slug, err)
			}
			if _, err := st.SetCompanyTracking(ctx, userID, c.ID, true, 0); err != nil {
				t.Fatalf("SetCompanyTracking(%s) = %v", slug, err)
			}
			state := "new"
			if slug == "kept-co" {
				state = "kept"
			}
			if _, err := st.SetCompanyReviewState(ctx, userID, c.ID, state); err != nil {
				t.Fatalf("SetCompanyReviewState(%s) = %v", slug, err)
			}
			ids = append(ids, c.ID)
		}
		if _, err := st.UpsertCandidateBoard(ctx, ids[0], "greenhouse", "older-co"); err != nil {
			t.Fatalf("UpsertCandidateBoard(...) = %v", err)
		}
		for i, id := range ids {
			if _, _, err := st.SaveCanonical(ctx, dto.Job{
				Title: "Go Engineer", URL: fmt.Sprintf("https://example.com/new-co/%d", i), CompanyID: id, CompanySlug: fmt.Sprintf("new-co-%d", i),
			}); err != nil {
				t.Fatalf("SaveCanonical(...) = %v", err)
			}
		}

		got, err := st.ListNewCompanies(ctx, userID)
		if err != nil || len(got) != 2 || got[0].ID != ids[1] || got[1].ID != ids[0] {
			t.Fatalf("ListNewCompanies(...) = %+v, %v, want newer-co then older-co", got, err)
		}
		if len(got[0].Boards) != 0 || len(got[1].Boards) != 1 {
			t.Fatalf("ListNewCompanies(...) boards = %+v, want one on older-co only", got)
		}
		jobs, err := st.ListNewCompanyJobs(ctx, userID)
		if err != nil || len(jobs) != 2 {
			t.Fatalf("ListNewCompanyJobs(...) = %+v, %v, want the two new companies' jobs", jobs, err)
		}
		other, err := st.ListNewCompanies(ctx, missingID)
		if err != nil || len(other) != 0 {
			t.Fatalf("ListNewCompanies(other user) = %+v, %v, want none", other, err)
		}
	})

	t.Run("create then list source targets for a user", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		target, err := st.CreateSourceTarget(ctx, userID, "linkedin", "acme", true, map[string]string{}, dto.RunWindow{}, nil)
		if err != nil {
			t.Fatalf("CreateSourceTarget(...) = %v", err)
		}
		targets, err := st.ListSourceTargetsByUser(ctx, userID)
		if err != nil || len(targets) != 1 || targets[0].ID != target.ID {
			t.Fatalf("ListSourceTargetsByUser(...) = %+v, %v, want the created target", targets, err)
		}
	})

	t.Run("create a duplicate source target returns already exists", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		if _, err := st.CreateSourceTarget(ctx, userID, "linkedin", "dup", true, map[string]string{}, dto.RunWindow{}, nil); err != nil {
			t.Fatalf("CreateSourceTarget(...) = %v", err)
		}
		if _, err := st.CreateSourceTarget(ctx, userID, "linkedin", "dup", true, map[string]string{}, dto.RunWindow{}, nil); !errors.Is(err, store.ErrSourceTargetExists) {
			t.Fatalf("duplicate CreateSourceTarget(...) err = %v, want ErrSourceTargetExists", err)
		}
	})

	t.Run("restarting a failed run clears its error", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		target, err := st.CreateSourceTarget(ctx, userID, "linkedin", "retry", true, map[string]string{}, dto.RunWindow{}, nil)
		if err != nil {
			t.Fatalf("CreateSourceTarget(...) = %v", err)
		}
		started, err := st.StartSourceTargetRun(ctx, target.ID, nil)
		if err != nil {
			t.Fatalf("StartSourceTargetRun(...) = %v", err)
		}
		if _, err := st.TransitionSourceTargetRun(ctx, target.ID, started.RunID, "failed", "boom"); err != nil {
			t.Fatalf("TransitionSourceTargetRun(...) = %v", err)
		}
		restarted, err := st.StartSourceTargetRun(ctx, target.ID, nil)
		if err != nil || restarted.RunStatus != "queued" || restarted.LastRunError != "" {
			t.Fatalf("StartSourceTargetRun(...) = %+v, %v, want queued with no error", restarted, err)
		}
	})

	t.Run("disabling a source records the reason until the target is re-enabled", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		running, err := st.CreateSourceTargetWithRun(ctx, userID, "indeed", "go", true, map[string]string{}, dto.RunWindow{}, nil)
		if err != nil {
			t.Fatalf("CreateSourceTargetWithRun(...) = %v", err)
		}
		other, err := st.CreateSourceTarget(ctx, userID, "linkedin", "go", true, map[string]string{}, dto.RunWindow{}, nil)
		if err != nil {
			t.Fatalf("CreateSourceTarget(...) = %v", err)
		}

		n, err := st.DisableSourceTargets(ctx, "indeed", "indeed key rejected")
		if err != nil || n != 1 {
			t.Fatalf("DisableSourceTargets(...) = %d, %v, want 1", n, err)
		}
		got, err := st.GetSourceTarget(ctx, running.ID)
		if err != nil || got.Enabled || got.DisabledReason != "indeed key rejected" || got.RunStatus != "failed" || got.LastRunError != "indeed key rejected" {
			t.Fatalf("GetSourceTarget(...) = %+v, %v, want disabled, failed with the reason", got, err)
		}
		if got, _ := st.GetSourceTarget(ctx, other.ID); !got.Enabled || got.DisabledReason != "" {
			t.Fatalf("other source target = %+v, want untouched", got)
		}
		if n, err := st.DisableSourceTargets(ctx, "indeed", "again"); err != nil || n != 0 {
			t.Fatalf("second DisableSourceTargets(...) = %d, %v, want 0", n, err)
		}

		enabled := true
		got, err = st.UpdateSourceTarget(ctx, running.ID, userID, &enabled, nil, nil)
		if err != nil || !got.Enabled || got.DisabledReason != "" {
			t.Fatalf("UpdateSourceTarget(enable) = %+v, %v, want enabled with no reason", got, err)
		}
	})

	t.Run("starting a run clears the disabled reason", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		target, err := st.CreateSourceTarget(ctx, userID, "indeed", "go", true, map[string]string{}, dto.RunWindow{}, nil)
		if err != nil {
			t.Fatalf("CreateSourceTarget(...) = %v", err)
		}
		if _, err := st.DisableSourceTargets(ctx, "indeed", "indeed key rejected"); err != nil {
			t.Fatalf("DisableSourceTargets(...) = %v", err)
		}
		started, err := st.StartSourceTargetRun(ctx, target.ID, nil)
		if err != nil || !started.Enabled || started.DisabledReason != "" {
			t.Fatalf("StartSourceTargetRun(...) = %+v, %v, want enabled with no reason", started, err)
		}
	})

	t.Run("update then delete a source target", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		target, err := st.CreateSourceTarget(ctx, userID, "linkedin", "update-me", true, map[string]string{}, dto.RunWindow{}, nil)
		if err != nil {
			t.Fatalf("CreateSourceTarget(...) = %v", err)
		}
		disabled := false
		updated, err := st.UpdateSourceTarget(ctx, target.ID, userID, &disabled, nil, nil)
		if err != nil || updated.Enabled {
			t.Fatalf("UpdateSourceTarget(...) = %+v, %v, want disabled", updated, err)
		}
		if err := st.DeleteSourceTarget(ctx, target.ID, userID); err != nil {
			t.Fatalf("DeleteSourceTarget(...) = %v", err)
		}
		targets, err := st.ListSourceTargetsByUser(ctx, userID)
		if err != nil || len(targets) != 0 {
			t.Fatalf("ListSourceTargetsByUser(...) after delete = %+v, %v, want none", targets, err)
		}
	})

	t.Run("a new source target is manual on the default window", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		target, err := st.CreateSourceTarget(t.Context(), userID, "linkedin", "defaults", true, map[string]string{}, dto.RunWindow{}, nil)
		if err != nil {
			t.Fatalf("CreateSourceTarget(...) = %v", err)
		}
		want := runwindow.Default()
		want.IntervalMinutes = nil
		if diff := cmp.Diff(want, target.RunWindow); diff != "" {
			t.Errorf("CreateSourceTarget(...).RunWindow mismatch (-want +got):\n%s", diff)
		}
		if target.NextRunAt != nil {
			t.Errorf("CreateSourceTarget(...).NextRunAt = %v, want nil", target.NextRunAt)
		}
	})

	t.Run("run window and next run round trip through create update and rerun", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		interval := 180
		window := dto.RunWindow{
			IntervalMinutes: &interval,
			Weekdays:        []time.Weekday{time.Monday, time.Sunday},
			Start:           "09:30",
			End:             "17:45",
			Timezone:        "America/New_York",
		}
		next := time.Date(2026, time.October, 12, 13, 30, 0, 0, time.UTC)

		created, err := st.CreateSourceTargetWithRun(ctx, userID, "linkedin", "windowed", true, map[string]string{}, window, &next)
		if err != nil {
			t.Fatalf("CreateSourceTargetWithRun(...) = %v", err)
		}
		if diff := cmp.Diff(window, created.RunWindow); diff != "" {
			t.Errorf("created RunWindow mismatch (-want +got):\n%s", diff)
		}
		if created.NextRunAt == nil || !created.NextRunAt.Equal(next) {
			t.Errorf("created NextRunAt = %v, want %v", created.NextRunAt, next)
		}

		manual := window
		manual.IntervalMinutes = nil
		updated, err := st.UpdateSourceTarget(ctx, created.ID, userID, nil, &manual, nil)
		if err != nil {
			t.Fatalf("UpdateSourceTarget(manual) = %v", err)
		}
		if updated.RunWindow.IntervalMinutes != nil || updated.NextRunAt != nil {
			t.Errorf("manual update interval = %v, next run = %v, want both nil", updated.RunWindow.IntervalMinutes, updated.NextRunAt)
		}
		if updated.RunWindow.Timezone != window.Timezone || updated.RunWindow.Start != window.Start {
			t.Errorf("manual update RunWindow = %+v, want the window kept", updated.RunWindow)
		}

		later := next.Add(time.Hour)
		rerun, err := st.StartSourceTargetRun(ctx, created.ID, &later)
		if err != nil {
			t.Fatalf("StartSourceTargetRun(...) = %v", err)
		}
		if rerun.NextRunAt == nil || !rerun.NextRunAt.Equal(later) {
			t.Errorf("rerun NextRunAt = %v, want %v", rerun.NextRunAt, later)
		}
	})

	t.Run("get an unknown source target returns not found", func(t *testing.T) {
		t.Parallel()
		st, _ := newStore(t)
		_, err := st.GetSourceTarget(t.Context(), missingID)
		if !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("GetSourceTarget(...) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("save cards then list for user round trips", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		target, err := st.CreateSourceTarget(ctx, userID, "linkedin", "candidate-search", true, map[string]string{}, dto.RunWindow{}, nil)
		if err != nil {
			t.Fatalf("CreateSourceTarget(...) = %v", err)
		}
		saved, err := st.SaveCards(ctx, target, []dto.Job{{URL: "https://example.com/candidate/1", Title: "Engineer"}})
		if err != nil || len(saved) != 1 {
			t.Fatalf("SaveCards(...) = %+v, %v, want one candidate", saved, err)
		}
		listed, err := st.ListForUser(ctx, userID, "", 10)
		if err != nil || len(listed) != 1 || listed[0].ID != saved[0].ID {
			t.Fatalf("ListForUser(...) = %+v, %v, want the saved candidate", listed, err)
		}
	})

	t.Run("new urls counts saved candidates as known under any host or tracking params", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		target, err := st.CreateSourceTarget(ctx, userID, "linkedin", "known-search", true, map[string]string{}, dto.RunWindow{}, nil)
		if err != nil {
			t.Fatalf("CreateSourceTarget(...) = %v", err)
		}
		if _, err := st.SaveCards(ctx, target, []dto.Job{{URL: "https://www.linkedin.com/jobs/view/111", Title: "Engineer"}}); err != nil {
			t.Fatalf("SaveCards(...) = %v", err)
		}
		fresh := "https://www.linkedin.com/jobs/view/222"
		got, err := st.NewURLs(ctx, []string{
			"https://www.linkedin.com/jobs/view/111",
			"https://uk.linkedin.com/jobs/view/111?ref=x&utm_source=y",
			fresh,
		})
		if err != nil || len(got) != 1 || got[0] != fresh {
			t.Fatalf("NewURLs(...) = %v, %v, want only %q", got, err, fresh)
		}
	})

	t.Run("list for user returns the full card only for complete sources", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		complete, err := st.CreateSourceTarget(ctx, userID, "remoteok", "complete-search", true, map[string]string{}, dto.RunWindow{}, nil)
		if err != nil {
			t.Fatalf("CreateSourceTarget(remoteok) = %v", err)
		}
		partial, err := st.CreateSourceTarget(ctx, userID, "linkedin", "partial-search", true, map[string]string{}, dto.RunWindow{}, nil)
		if err != nil {
			t.Fatalf("CreateSourceTarget(linkedin) = %v", err)
		}
		if _, err := st.SaveCards(ctx, complete, []dto.Job{{URL: "https://example.com/full", Title: "Engineer", Description: "full text"}}); err != nil {
			t.Fatalf("SaveCards(remoteok) = %v", err)
		}
		if _, err := st.SaveCards(ctx, partial, []dto.Job{{URL: "https://example.com/partial", Title: "Engineer", Description: "full text"}}); err != nil {
			t.Fatalf("SaveCards(linkedin) = %v", err)
		}
		listed, err := st.ListForUser(ctx, userID, "", 10)
		if err != nil {
			t.Fatalf("ListForUser(...) = %v", err)
		}
		got := map[string]string{}
		for _, c := range listed {
			got[c.Card.Source] = c.Card.Description
		}
		want := map[string]string{"remoteok": "full text", "linkedin": ""}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("ListForUser(...) descriptions by source (-want +got):\n%s", diff)
		}
	})

	t.Run("a card keeps its remote-check page and its discovering targets' filters", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		remote := map[string]string{"arrangement": "2"}
		first, err := st.CreateSourceTarget(ctx, userID, "linkedin", "remote-search", true, remote, dto.RunWindow{}, nil)
		if err != nil {
			t.Fatalf("CreateSourceTarget(remote) = %v", err)
		}
		second, err := st.CreateSourceTarget(ctx, userID, "linkedin", "london-search", true, map[string]string{}, dto.RunWindow{}, nil)
		if err != nil {
			t.Fatalf("CreateSourceTarget(london) = %v", err)
		}
		card := dto.Job{URL: "https://www.linkedin.com/jobs/view/777", Title: "Engineer", RemoteCheckURL: "https://uk.linkedin.com/jobs/view/engineer-777", WorkArrangement: "remote"}
		var saved []sourcetargets.Candidate
		for _, target := range []dto.SourceTarget{first, second} {
			if saved, err = st.SaveCards(ctx, target, []dto.Job{card}); err != nil || len(saved) != 1 {
				t.Fatalf("SaveCards(%s) = %+v, %v, want one candidate", target.Value, saved, err)
			}
		}

		listed, err := st.ListForUser(ctx, userID, "", 10)
		if err != nil || len(listed) != 1 {
			t.Fatalf("ListForUser(...) = %+v, %v, want one candidate", listed, err)
		}
		if got := listed[0].Card; got.RemoteCheckURL != card.RemoteCheckURL || got.WorkArrangement != "remote" {
			t.Errorf("listed card = {RemoteCheckURL: %q, WorkArrangement: %q}, want both kept", got.RemoteCheckURL, got.WorkArrangement)
		}

		got, err := st.DiscoveryFilters(ctx, userID, []string{saved[0].ID})
		if err != nil {
			t.Fatalf("DiscoveryFilters(...) = %v", err)
		}
		slices.SortFunc(got[saved[0].ID], func(a, b map[string]string) int { return len(b) - len(a) })
		want := map[string][]map[string]string{saved[0].ID: {remote, {}}}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("DiscoveryFilters(...) (-want +got):\n%s", diff)
		}
		if other, err := st.DiscoveryFilters(ctx, missingID, []string{saved[0].ID}); err != nil || len(other) != 0 {
			t.Errorf("DiscoveryFilters(other user) = %v, %v, want none", other, err)
		}
	})

	t.Run("assess a relevant candidate requests detail once", func(t *testing.T) {
		t.Parallel()
		st, userID := newStore(t)
		ctx := t.Context()
		target, err := st.CreateSourceTarget(ctx, userID, "linkedin", "assess-search", true, map[string]string{}, dto.RunWindow{}, nil)
		if err != nil {
			t.Fatalf("CreateSourceTarget(...) = %v", err)
		}
		saved, err := st.SaveCards(ctx, target, []dto.Job{{URL: "https://example.com/candidate/2", Title: "Engineer"}})
		if err != nil || len(saved) != 1 {
			t.Fatalf("SaveCards(...) = %+v, %v", saved, err)
		}
		version := time.Now()
		requested, err := st.Assess(ctx, saved[0].ID, userID, version, true)
		if err != nil || !requested {
			t.Fatalf("Assess(...) = %v, %v, want requested", requested, err)
		}
		if err := st.MarkDetailPending(ctx, saved[0].ID); err != nil {
			t.Fatalf("MarkDetailPending(...) = %v", err)
		}
		requested, err = st.Assess(ctx, saved[0].ID, userID, version.Add(time.Second), true)
		if err != nil || requested {
			t.Fatalf("Assess(...) after pending = %v, %v, want not requested", requested, err)
		}
	})
}

const missingID = "00000000-0000-0000-0000-000000000000"

func verifyBoard(t *testing.T, st Store, companyID, source string) {
	t.Helper()
	ctx := t.Context()
	if _, err := st.UpsertCandidateBoard(ctx, companyID, source, companyID); err != nil {
		t.Fatalf("UpsertCandidateBoard(%q) = %v", source, err)
	}
	if _, err := st.VerifyCompanyBoard(ctx, companyID, source, companyID, "discovered", ""); err != nil {
		t.Fatalf("VerifyCompanyBoard(%q) = %v", source, err)
	}
}

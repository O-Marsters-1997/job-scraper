package jobsearchtest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
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

	t.Run("get an unknown job returns not found", func(t *testing.T) {
		st, _ := newStore(t)
		_, err := st.GetJob(context.Background(), missingID, missingID)
		if !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("GetJob(...) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("save then get a canonical job round trips", func(t *testing.T) {
		st, _ := newStore(t)
		ctx := context.Background()
		saved, status, err := st.SaveCanonical(ctx, dto.Job{Title: "Engineer", URL: "https://example.com/jobs/1"})
		if err != nil || status != "new" || saved.ID == "" {
			t.Fatalf("SaveCanonical(...) = %+v, %q, %v, want a new job", saved, status, err)
		}
		got, err := st.GetJob(ctx, saved.ID, missingID)
		if err != nil || got.Title != "Engineer" {
			t.Fatalf("GetJob(...) = %+v, %v, want the saved job", got, err)
		}
	})

	t.Run("resaving the same content is unchanged, changed content is changed", func(t *testing.T) {
		st, _ := newStore(t)
		ctx := context.Background()
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

	t.Run("new urls filters out known ones", func(t *testing.T) {
		st, _ := newStore(t)
		ctx := context.Background()
		if _, _, err := st.SaveCanonical(ctx, dto.Job{Title: "Engineer", URL: "https://example.com/jobs/3"}); err != nil {
			t.Fatalf("SaveCanonical(...) = %v", err)
		}
		fresh, err := st.NewURLs(ctx, []string{"https://example.com/jobs/3", "https://example.com/jobs/new"})
		if err != nil || len(fresh) != 1 || fresh[0] != "https://example.com/jobs/new" {
			t.Fatalf("NewURLs(...) = %v, %v, want only the unseen URL", fresh, err)
		}
	})

	t.Run("upsert company then get round trips", func(t *testing.T) {
		st, _ := newStore(t)
		ctx := context.Background()
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
		st, _ := newStore(t)
		_, err := st.GetCompany(context.Background(), missingID)
		if !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("GetCompany(...) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("set company tracking then list for user shows it", func(t *testing.T) {
		st, userID := newStore(t)
		ctx := context.Background()
		c, err := st.UpsertCompany(ctx, dto.CompanyUpsert{Slug: "tracked-co", Name: "Tracked Co"})
		if err != nil {
			t.Fatalf("UpsertCompany(...) = %v", err)
		}
		if _, err := st.SetCompanyTracking(ctx, userID, c.ID, true, 180); err != nil {
			t.Fatalf("SetCompanyTracking(...) = %v", err)
		}
		companies, err := st.ListCompaniesForUser(ctx, userID)
		if err != nil || len(companies) != 1 || !companies[0].Tracked || companies[0].CheckIntervalMinutes != 180 {
			t.Fatalf("ListCompaniesForUser(...) = %+v, %v, want one tracked company", companies, err)
		}
	})

	t.Run("list tracked companies returns only this user's, paused included, with boards", func(t *testing.T) {
		st, userID := newStore(t)
		ctx := context.Background()
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

	t.Run("delete company tracking removes it and keeps the company", func(t *testing.T) {
		st, userID := newStore(t)
		ctx := context.Background()
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

	t.Run("set company tracking without an interval keeps the existing one", func(t *testing.T) {
		st, userID := newStore(t)
		ctx := context.Background()
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

	t.Run("create then list source targets for a user", func(t *testing.T) {
		st, userID := newStore(t)
		ctx := context.Background()
		target, err := st.CreateSourceTarget(ctx, userID, "linkedin", "acme", true, map[string]string{})
		if err != nil {
			t.Fatalf("CreateSourceTarget(...) = %v", err)
		}
		targets, err := st.ListSourceTargetsByUser(ctx, userID)
		if err != nil || len(targets) != 1 || targets[0].ID != target.ID {
			t.Fatalf("ListSourceTargetsByUser(...) = %+v, %v, want the created target", targets, err)
		}
	})

	t.Run("create a duplicate source target returns already exists", func(t *testing.T) {
		st, userID := newStore(t)
		ctx := context.Background()
		if _, err := st.CreateSourceTarget(ctx, userID, "linkedin", "dup", true, map[string]string{}); err != nil {
			t.Fatalf("CreateSourceTarget(...) = %v", err)
		}
		if _, err := st.CreateSourceTarget(ctx, userID, "linkedin", "dup", true, map[string]string{}); !errors.Is(err, store.ErrSourceTargetExists) {
			t.Fatalf("duplicate CreateSourceTarget(...) err = %v, want ErrSourceTargetExists", err)
		}
	})

	t.Run("restarting a failed run clears its error", func(t *testing.T) {
		st, userID := newStore(t)
		ctx := context.Background()
		target, err := st.CreateSourceTarget(ctx, userID, "linkedin", "retry", true, map[string]string{})
		if err != nil {
			t.Fatalf("CreateSourceTarget(...) = %v", err)
		}
		started, err := st.StartSourceTargetRun(ctx, target.ID)
		if err != nil {
			t.Fatalf("StartSourceTargetRun(...) = %v", err)
		}
		if _, err := st.TransitionSourceTargetRun(ctx, target.ID, started.RunID, "failed", "boom"); err != nil {
			t.Fatalf("TransitionSourceTargetRun(...) = %v", err)
		}
		restarted, err := st.StartSourceTargetRun(ctx, target.ID)
		if err != nil || restarted.RunStatus != "queued" || restarted.LastRunError != "" {
			t.Fatalf("StartSourceTargetRun(...) = %+v, %v, want queued with no error", restarted, err)
		}
	})

	t.Run("update then delete a source target", func(t *testing.T) {
		st, userID := newStore(t)
		ctx := context.Background()
		target, err := st.CreateSourceTarget(ctx, userID, "linkedin", "update-me", true, map[string]string{})
		if err != nil {
			t.Fatalf("CreateSourceTarget(...) = %v", err)
		}
		disabled := false
		updated, err := st.UpdateSourceTarget(ctx, target.ID, userID, &disabled, nil)
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

	t.Run("get an unknown source target returns not found", func(t *testing.T) {
		st, _ := newStore(t)
		_, err := st.GetSourceTarget(context.Background(), missingID)
		if !errors.Is(err, data.ErrNotFound) {
			t.Fatalf("GetSourceTarget(...) err = %v, want ErrNotFound", err)
		}
	})

	t.Run("save cards then list for user round trips", func(t *testing.T) {
		st, userID := newStore(t)
		ctx := context.Background()
		target, err := st.CreateSourceTarget(ctx, userID, "linkedin", "candidate-search", true, map[string]string{})
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

	t.Run("assess a relevant candidate requests detail once", func(t *testing.T) {
		st, userID := newStore(t)
		ctx := context.Background()
		target, err := st.CreateSourceTarget(ctx, userID, "linkedin", "assess-search", true, map[string]string{})
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

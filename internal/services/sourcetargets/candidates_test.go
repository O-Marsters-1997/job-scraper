package sourcetargets_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/queue/queuetest"
	"github.com/ollymarsters/job-scraper/internal/services/jobsearch/jobsearchtest"
	"github.com/ollymarsters/job-scraper/internal/services/sourcetargets"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
)

type failingJobQueue struct {
	*queuetest.Recorder
	err error
}

func (q *failingJobQueue) EnqueueJobs(ctx context.Context, jobs []dto.QueuedJob) error {
	if q.err != nil {
		return q.err
	}
	return q.Recorder.EnqueueJobs(ctx, jobs)
}

var _ sourcetargets.QueuePublisher = (*failingJobQueue)(nil)

var (
	captureTarget = dto.SourceTarget{ID: "target-1", UserID: userID, Source: "wis"}
	captureCard   = dto.Job{URL: "https://example.com/1", Title: "Engineer"}
)

func TestCapturePage(t *testing.T) {
	t.Run("retries after a queue failure", func(t *testing.T) {
		q := &failingJobQueue{Recorder: queuetest.NewRecorder(), err: errors.New("queue unavailable")}
		svc := serviceOver(jobsearchtest.NewFakeStore(), q)
		config := dto.SearchConfig{UserID: userID, UpdatedAt: time.Now().UTC()}
		if err := svc.CapturePage(t.Context(), captureTarget, []dto.Job{captureCard}, config); err == nil {
			t.Fatal("CapturePage() err = nil, want the queue failure")
		}

		q.err = nil
		if err := svc.Reconsider(t.Context(), config); err != nil {
			t.Fatalf("Reconsider() err = %v", err)
		}
		if got := len(q.Jobs()); got != 1 {
			t.Errorf("retry queued %d details, want 1", got)
		}
	})

	t.Run("drops LinkedIn cards for tracked companies before detail fetch", func(t *testing.T) {
		st := jobsearchtest.NewFakeStore()
		q := queuetest.NewRecorder()
		svc := sourcetargets.New(st, fakeSearchConfigReader{}, q, verifiedBoards{{CompanySlug: "polled-co", Source: "ashby", BoardToken: "polled", Tracked: true, Polled: true}}, sourcetargets.DefaultMaxAutomatic)
		linkedin := dto.SourceTarget{ID: "target-2", UserID: userID, Source: "linkedin"}
		polled := dto.Job{URL: "https://linkedin.com/jobs/view/1", Title: "Engineer", CompanySlug: "polled-co"}
		unpolled := dto.Job{URL: "https://linkedin.com/jobs/view/2", Title: "Engineer", CompanySlug: "other-co"}
		config := dto.SearchConfig{UserID: userID, UpdatedAt: time.Now().UTC()}
		if err := svc.CapturePage(t.Context(), linkedin, []dto.Job{polled, unpolled}, config); err != nil {
			t.Fatalf("CapturePage(linkedin) err = %v", err)
		}
		if jobs := q.Jobs(); len(jobs) != 1 || jobs[0].URL != unpolled.URL {
			t.Errorf("queued after linkedin page = %+v, want only %s", jobs, unpolled.URL)
		}

		other := dto.Job{URL: "https://example.com/3", Title: "Engineer", CompanySlug: "polled-co"}
		if err := svc.CapturePage(t.Context(), captureTarget, []dto.Job{other}, config); err != nil {
			t.Fatalf("CapturePage(wis) err = %v", err)
		}
		if got := len(q.Jobs()); got != 2 {
			t.Errorf("queued %d details after non-linkedin card, want 2", got)
		}
		if got := q.Tasks(); len(got) != 0 {
			t.Errorf("published %+v for tracked company, want nothing", got)
		}
	})

	t.Run("drops LinkedIn cards for untracked verified companies and harvests the board once", func(t *testing.T) {
		st := jobsearchtest.NewFakeStore()
		q := queuetest.NewRecorder()
		svc := sourcetargets.New(st, fakeSearchConfigReader{}, q, verifiedBoards{{CompanySlug: "acme", Source: "ashby", BoardToken: "acme", Polled: true}}, sourcetargets.DefaultMaxAutomatic)
		linkedin := dto.SourceTarget{ID: "target-2", UserID: userID, Source: "linkedin"}
		cards := []dto.Job{
			{URL: "https://linkedin.com/jobs/view/1", Title: "Engineer", CompanySlug: "acme"},
			{URL: "https://linkedin.com/jobs/view/2", Title: "Designer", CompanySlug: "acme"},
			{URL: "https://linkedin.com/jobs/view/3", Title: "Engineer", CompanySlug: "other-co"},
		}
		config := dto.SearchConfig{UserID: userID, UpdatedAt: time.Now().UTC()}
		if err := svc.CapturePage(t.Context(), linkedin, cards, config); err != nil {
			t.Fatalf("CapturePage() err = %v", err)
		}
		if jobs := q.Jobs(); len(jobs) != 1 || jobs[0].URL != cards[2].URL {
			t.Errorf("queued = %+v, want only %s", jobs, cards[2].URL)
		}
		tasks := q.Tasks()
		if len(tasks) != 1 || tasks[0].Kind != queue.BoardDiscoverTask || tasks[0].Source != "ashby" || tasks[0].BoardToken != "acme" {
			t.Errorf("published = %+v, want one board_discover for ashby/acme", tasks)
		}
	})

	t.Run("keeps LinkedIn cards whose verified board has never been polled", func(t *testing.T) {
		q := queuetest.NewRecorder()
		boards := verifiedBoards{{CompanySlug: "platformed", Source: "wttj", BoardToken: "Platformed"}}
		svc := sourcetargets.New(jobsearchtest.NewFakeStore(), fakeSearchConfigReader{}, q, boards, sourcetargets.DefaultMaxAutomatic)
		linkedin := dto.SourceTarget{ID: "target-2", UserID: userID, Source: "linkedin"}
		card := dto.Job{URL: "https://linkedin.com/jobs/view/1", Title: "Senior Software Engineer", CompanySlug: "platformed"}
		config := dto.SearchConfig{UserID: userID, UpdatedAt: time.Now().UTC()}
		if err := svc.CapturePage(t.Context(), linkedin, []dto.Job{card}, config); err != nil {
			t.Fatalf("CapturePage() err = %v", err)
		}
		if jobs := q.Jobs(); len(jobs) != 1 || jobs[0].URL != card.URL {
			t.Errorf("queued = %+v, want only %s", jobs, card.URL)
		}
		tasks := q.Tasks()
		if len(tasks) != 1 || tasks[0].Kind != queue.BoardDiscoverTask || tasks[0].BoardToken != "Platformed" {
			t.Errorf("published = %+v, want one board_discover for wttj/Platformed", tasks)
		}
	})

	t.Run("a dropped board card dates the held job it matches", func(t *testing.T) {
		st := jobsearchtest.NewFakeStore()
		svc := sourcetargets.New(st, fakeSearchConfigReader{}, queuetest.NewRecorder(), verifiedBoards{{CompanySlug: "acme", Source: "ashby", BoardToken: "acme", Polled: true}}, sourcetargets.DefaultMaxAutomatic)
		held, _, err := st.SaveCanonical(t.Context(), dto.Job{Title: "Engineer", CompanySlug: "acme", URL: "https://jobs.example.com/acme/1"})
		if err != nil {
			t.Fatalf("SaveCanonical() err = %v", err)
		}
		postedAt := time.Date(2026, 10, 9, 8, 0, 0, 0, time.UTC)
		card := dto.Job{URL: "https://linkedin.com/jobs/view/9", Title: "Engineer", CompanySlug: "acme", PostedAt: &postedAt}
		linkedin := dto.SourceTarget{ID: "target-2", UserID: userID, Source: "linkedin"}
		config := dto.SearchConfig{UserID: userID, UpdatedAt: time.Now().UTC()}
		if err := svc.CapturePage(t.Context(), linkedin, []dto.Job{card}, config); err != nil {
			t.Fatalf("CapturePage() err = %v", err)
		}
		got, err := st.GetJob(t.Context(), held.ID, "")
		if err != nil || got.PostedAt == nil || !got.PostedAt.Equal(postedAt) {
			t.Errorf("GetJob() = %+v, %v, want PostedAt %v", got, err, postedAt)
		}
	})

	t.Run("retains a rejected card and reconsideration queues it once", func(t *testing.T) {
		svc, _, q := newService(t)
		card := captureCard
		card.Title = "Senior Engineer"
		first := dto.SearchConfig{UserID: userID, ExcludedTitleKeywords: []string{"senior"}, UpdatedAt: time.Now().UTC()}
		if err := svc.CapturePage(t.Context(), captureTarget, []dto.Job{card, card}, first); err != nil {
			t.Fatalf("CapturePage() err = %v", err)
		}
		if got := len(q.Jobs()); got != 0 {
			t.Fatalf("rejected card queued %d details, want 0", got)
		}

		changed := dto.SearchConfig{UserID: userID, UpdatedAt: first.UpdatedAt.Add(time.Second)}
		for range 2 {
			if err := svc.Reconsider(t.Context(), changed); err != nil {
				t.Fatalf("Reconsider() err = %v", err)
			}
		}
		if jobs := q.Jobs(); len(jobs) != 1 || jobs[0].URL != card.URL {
			t.Errorf("queued after reconsideration = %+v, want one job for %s", jobs, card.URL)
		}
	})

	const slugPage = "https://uk.linkedin.com/jobs/view/engineer-at-acme-1"
	countryCard := dto.Job{URL: "https://www.linkedin.com/jobs/view/1", Title: "Engineer", Location: "United Kingdom", WorkArrangement: "remote", RemoteCheckURL: slugPage}
	cityCard := dto.Job{URL: "https://www.linkedin.com/jobs/view/2", Title: "Engineer", Location: "London, England, United Kingdom", RemoteCheckURL: slugPage}
	linkedinTarget := func(id string, filters map[string]string) dto.SourceTarget {
		return dto.SourceTarget{ID: id, UserID: userID, Source: "linkedin", Filters: filters}
	}
	remoteOnly := map[string]string{"arrangement": sourcespec.LinkedInRemote}
	remoteWithCityCards := map[string]string{"arrangement": sourcespec.LinkedInRemote, "city_cards": sourcespec.LinkedInCityCardsFullPage}
	hybrid := map[string]string{"arrangement": sourcespec.LinkedInHybrid}

	t.Run("a LinkedIn target enforces its own arrangement on its cards", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name      string
			filters   map[string]string
			card      dto.Job
			wantQueue bool
			wantCheck string
		}{
			{"no arrangement queues a country card on the fragment", nil, countryCard, true, ""},
			{"no arrangement queues a city card on the fragment", nil, cityCard, true, ""},
			{"remote queues a country-only card on the fragment", remoteOnly, countryCard, true, ""},
			{"remote rejects a city card", remoteOnly, cityCard, false, ""},
			{"remote with city cards queues a city card for its full page", remoteWithCityCards, cityCard, true, slugPage},
			{"remote with city cards rejects a city card with no full page link", remoteWithCityCards, dto.Job{URL: cityCard.URL, Location: cityCard.Location}, false, ""},
			{"not remote queues a city card", hybrid, cityCard, true, ""},
			{"not remote rejects a country-only card", hybrid, countryCard, false, ""},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				svc, _, q := newService(t)
				config := dto.SearchConfig{UserID: userID, UpdatedAt: time.Now().UTC()}
				if err := svc.CapturePage(t.Context(), linkedinTarget("target-l", tt.filters), []dto.Job{tt.card}, config); err != nil {
					t.Fatalf("CapturePage() err = %v", err)
				}
				jobs := q.Jobs()
				if (len(jobs) == 1) != tt.wantQueue || len(jobs) > 1 {
					t.Fatalf("queued %d details, want queued = %t", len(jobs), tt.wantQueue)
				}
				if len(jobs) == 1 && jobs[0].Card.RemoteCheckURL != tt.wantCheck {
					t.Errorf("queued card RemoteCheckURL = %q, want %q", jobs[0].Card.RemoteCheckURL, tt.wantCheck)
				}
			})
		}
	})

	t.Run("a card goes to detail if any target that found it accepts it", func(t *testing.T) {
		t.Parallel()
		svc, _, q := newService(t)
		held := dto.SearchConfig{UserID: userID, ExcludedTitleKeywords: []string{"engineer"}, UpdatedAt: time.Now().UTC()}
		for _, target := range []dto.SourceTarget{linkedinTarget("remote-target", remoteWithCityCards), linkedinTarget("london-target", nil)} {
			if err := svc.CapturePage(t.Context(), target, []dto.Job{cityCard}, held); err != nil {
				t.Fatalf("CapturePage(%s) err = %v", target.ID, err)
			}
		}
		open := dto.SearchConfig{UserID: userID, UpdatedAt: held.UpdatedAt.Add(time.Second)}
		if err := svc.Reconsider(t.Context(), open); err != nil {
			t.Fatalf("Reconsider() err = %v", err)
		}
		jobs := q.Jobs()
		if len(jobs) != 1 {
			t.Fatalf("queued %d details, want 1", len(jobs))
		}
		if got := jobs[0].Card.RemoteCheckURL; got != "" {
			t.Errorf("queued card RemoteCheckURL = %q, want the fragment path", got)
		}
	})

	t.Run("reconsideration applies the filters of every target that found the card", func(t *testing.T) {
		t.Parallel()
		svc, _, q := newService(t)
		first := dto.SearchConfig{UserID: userID, ExcludedTitleKeywords: []string{"engineer"}, UpdatedAt: time.Now().UTC()}
		for _, target := range []dto.SourceTarget{linkedinTarget("remote-target", remoteOnly), linkedinTarget("hybrid-target", hybrid)} {
			if err := svc.CapturePage(t.Context(), target, []dto.Job{cityCard, countryCard}, first); err != nil {
				t.Fatalf("CapturePage(%s) err = %v", target.ID, err)
			}
		}
		changed := dto.SearchConfig{UserID: userID, UpdatedAt: first.UpdatedAt.Add(time.Second)}
		if err := svc.Reconsider(t.Context(), changed); err != nil {
			t.Fatalf("Reconsider() err = %v", err)
		}
		if got := len(q.Jobs()); got != 2 {
			t.Errorf("queued %d details, want both cards: each is accepted by one of the two targets", got)
		}
	})

	t.Run("reconsideration keeps the description of a complete card", func(t *testing.T) {
		svc, _, q := newService(t)
		remoteok := dto.SourceTarget{ID: "target-3", UserID: userID, Source: "remoteok"}
		card := dto.Job{URL: "https://remoteok.com/1", Title: "Senior Engineer", Description: "full text"}
		first := dto.SearchConfig{UserID: userID, ExcludedTitleKeywords: []string{"senior"}, UpdatedAt: time.Now().UTC()}
		if err := svc.CapturePage(t.Context(), remoteok, []dto.Job{card}, first); err != nil {
			t.Fatalf("CapturePage() err = %v", err)
		}

		changed := dto.SearchConfig{UserID: userID, UpdatedAt: first.UpdatedAt.Add(time.Second)}
		if err := svc.Reconsider(t.Context(), changed); err != nil {
			t.Fatalf("Reconsider() err = %v", err)
		}
		if jobs := q.Jobs(); len(jobs) != 1 || jobs[0].Card.Description != card.Description {
			t.Errorf("queued after reconsideration = %+v, want the card with its description", jobs)
		}
	})
}

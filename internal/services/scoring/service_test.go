package scoring_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/scoringtest"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/store"
	"github.com/ollymarsters/job-scraper/internal/telemetry"
)

var bank = []dto.ScoringOption{
	{ID: "tech:go", Dimension: dto.DimensionTech, Label: "Go", Question: "Does the role use Go?"},
	{ID: "tech:rust", Dimension: dto.DimensionTech, Label: "Rust", Question: "Does the role use Rust?"},
	{ID: "role:backend", Dimension: dto.DimensionRole, Label: "Backend", Question: "Is this primarily a backend role?"},
	{ID: "tech:kubernetes", Dimension: dto.DimensionTech, Label: "Kubernetes", Question: "Does the role use Kubernetes?"},
	{ID: "domain:gambling", Dimension: dto.DimensionDomain, Label: "Gambling", Question: "Is the company's main business gambling?"},
	{ID: "seniority:senior", Dimension: dto.DimensionSeniority, Label: "Senior", Question: "Seniority?"},
}

func newFakeStore() *scoringtest.FakeStore {
	st := scoringtest.NewFakeStore()
	st.SeedOptions(bank)
	return st
}

type fakeAnswerer struct {
	t         *testing.T
	forbidden bool
	calls     [][]string
	answers   map[string]dto.Answer
}

func (f *fakeAnswerer) Answer(_ context.Context, _ string, _ dto.Job, questions []string) (map[string]dto.Answer, dto.Usage, error) {
	if f.forbidden {
		f.t.Fatal("Answer called but no questions should have been missing")
	}
	f.calls = append(f.calls, questions)
	out := make(map[string]dto.Answer, len(questions))
	for _, q := range questions {
		if a, ok := f.answers[q]; ok {
			out[q] = a
			continue
		}
		out[q] = dto.Answer{PYes: 0.9, PNo: 0.05, PNotStated: 0.05}
	}
	return out, dto.Usage{Model: "typesafe/jev-1.13-test", Cost: 0.0004}, nil
}

type failingAnswerer struct{}

func (f *failingAnswerer) Answer(context.Context, string, dto.Job, []string) (map[string]dto.Answer, dto.Usage, error) {
	return nil, dto.Usage{}, errors.New("jev boom")
}

type fakeCredentials struct{ key string }

func (f *fakeCredentials) Get(context.Context, string, string) (string, error) {
	if f.key == "" {
		return "", data.ErrNotFound
	}
	return f.key, nil
}

type fakeAlerter struct{ notified []string }

func (f *fakeAlerter) NotifyNewJob(_ context.Context, _ dto.Job, email string) error {
	f.notified = append(f.notified, email)
	return nil
}

type fakeProfiles struct{ emails map[string]string }

func (f *fakeProfiles) GetProfile(_ context.Context, userID string) (dto.Profile, error) {
	return dto.Profile{Email: f.emails[userID]}, nil
}

func captureScoreCallLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return buf
}

func scoreCallLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var lines []map[string]any
	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		var line map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			t.Fatalf("unmarshal log line: %v", err)
		}
		if line["event"] == telemetry.EventScoreCall {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan log buffer: %v", err)
	}
	return lines
}

func TestRunTick(t *testing.T) {
	t.Run("all cached answers make no jev call", func(t *testing.T) {
		st := newFakeStore()
		job := dto.Job{ID: "job-1", Title: "Backend Engineer", ContentFingerprint: "fp-1", Source: "greenhouse"}
		cfg := dto.SearchConfig{UserID: "user-1", NotifyThreshold: 70, Preferences: dto.Preferences{
			Picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "manual"}},
		}}
		st.SeedJob(job, []dto.SearchConfig{cfg})
		st.SeedAnswers(job.ID, job.ContentFingerprint, "typesafe/jev-1.13", map[string]dto.Answer{
			scoring.QuestionHash("Does the role use Go?"):             {PYes: 0.9, PNo: 0.05, PNotStated: 0.05},
			scoring.QuestionHash("Does the role use Rust?"):           {PYes: 0.1, PNo: 0.85, PNotStated: 0.05},
			scoring.QuestionHash("Is this primarily a backend role?"): {PYes: 0.9, PNo: 0.05, PNotStated: 0.05},
		})
		st.SeedEffect(dto.AnswerEffect{ID: "effect-1", JobID: job.ID, Fingerprint: job.ContentFingerprint, Attempts: 1, FirstDiscovery: true})

		answerer := &fakeAnswerer{t: t, forbidden: true}
		svc := scoring.NewService(scoring.Deps{Store: st, Answerer: answerer, Credentials: &fakeCredentials{key: "sk-or-test"}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}})

		if err := svc.RunTick(context.Background()); err != nil {
			t.Fatalf("RunTick: %v", err)
		}
		completed := st.Completed()
		if len(completed) != 1 {
			t.Fatalf("completed effects = %d, want 1", len(completed))
		}
		if len(completed[0].Answers) != 0 {
			t.Fatalf("new answers written = %d, want 0 (everything cached)", len(completed[0].Answers))
		}
	})

	t.Run("missing questions are sent exactly", func(t *testing.T) {
		st := newFakeStore()
		job := dto.Job{ID: "job-2", Title: "Backend Engineer", ContentFingerprint: "fp-2", Source: "greenhouse"}
		cfg := dto.SearchConfig{UserID: "user-1", NotifyThreshold: 70, Preferences: dto.Preferences{
			Picks: []dto.Pick{
				{OptionID: "tech:go", Stance: "nice", Source: "manual"},
				{OptionID: "tech:rust", Stance: "avoid", Source: "manual"},
				{OptionID: "role:backend", Stance: "nice", Source: "manual"},
			},
		}}
		st.SeedJob(job, []dto.SearchConfig{cfg})
		st.SeedEffect(dto.AnswerEffect{ID: "effect-2", JobID: job.ID, Fingerprint: job.ContentFingerprint, Attempts: 1})

		answerer := &fakeAnswerer{t: t}
		svc := scoring.NewService(scoring.Deps{Store: st, Answerer: answerer, Credentials: &fakeCredentials{key: "sk-or-test"}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}})

		if err := svc.RunTick(context.Background()); err != nil {
			t.Fatalf("RunTick: %v", err)
		}
		if len(answerer.calls) != 1 {
			t.Fatalf("Answer calls = %d, want 1", len(answerer.calls))
		}
		got := slices.Clone(answerer.calls[0])
		slices.Sort(got)
		want := []string{"Does the role use Go?", "Does the role use Rust?", "Is this primarily a backend role?"}
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Fatalf("questions sent = %v, want %v", got, want)
		}
		completed := st.Completed()
		if len(completed) != 1 || len(completed[0].Answers) != 3 {
			t.Fatalf("completed effects = %+v, want 1 effect with 3 new answers", completed)
		}
	})

	t.Run("a successful answer emits a score call log", func(t *testing.T) {
		buf := captureScoreCallLogs(t)

		st := newFakeStore()
		job := dto.Job{ID: "job-score-call", Title: "Backend Engineer", ContentFingerprint: "fp-score-call", Source: "greenhouse"}
		cfg := dto.SearchConfig{UserID: "user-1", NotifyThreshold: 70, Preferences: dto.Preferences{
			Picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "manual"}},
		}}
		st.SeedJob(job, []dto.SearchConfig{cfg})
		st.SeedEffect(dto.AnswerEffect{ID: "effect-score-call", JobID: job.ID, Fingerprint: job.ContentFingerprint, Attempts: 1})

		svc := scoring.NewService(scoring.Deps{Store: st, Answerer: &fakeAnswerer{t: t}, Credentials: &fakeCredentials{key: "sk-or-test"}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}})

		if err := svc.RunTick(context.Background()); err != nil {
			t.Fatalf("RunTick: %v", err)
		}

		lines := scoreCallLines(t, buf)
		if len(lines) != 1 {
			t.Fatalf("score.call lines = %d, want 1: %v", len(lines), lines)
		}
		if lines[0]["user_id"] != "user-1" || lines[0]["model"] != "typesafe/jev-1.13-test" || lines[0]["cost_usd"] != 0.0004 {
			t.Fatalf("score.call line = %+v, want user_id=user-1 model=typesafe/jev-1.13-test cost_usd=0.0004", lines[0])
		}
	})

	t.Run("a failed answer emits no score call log", func(t *testing.T) {
		buf := captureScoreCallLogs(t)

		st := newFakeStore()
		job := dto.Job{ID: "job-score-call-fail", Title: "Backend Engineer", ContentFingerprint: "fp-score-call-fail", Source: "greenhouse"}
		cfg := dto.SearchConfig{UserID: "user-1", NotifyThreshold: 70, Preferences: dto.Preferences{
			Picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "manual"}},
		}}
		st.SeedJob(job, []dto.SearchConfig{cfg})
		st.SeedEffect(dto.AnswerEffect{ID: "effect-score-call-fail", JobID: job.ID, Fingerprint: job.ContentFingerprint, Attempts: 1})

		svc := scoring.NewService(scoring.Deps{Store: st, Answerer: &failingAnswerer{}, Credentials: &fakeCredentials{key: "sk-or-test"}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}})

		if err := svc.RunTick(context.Background()); err != nil {
			t.Fatalf("RunTick: %v", err)
		}

		if lines := scoreCallLines(t, buf); len(lines) != 0 {
			t.Fatalf("score.call lines = %v, want none after a failed Answer call", lines)
		}
		if failed := st.Failed(); len(failed) != 1 {
			t.Fatalf("failed effects = %d, want 1", len(failed))
		}
	})

	t.Run("only picked questions are sent", func(t *testing.T) {
		st := newFakeStore()
		job := dto.Job{ID: "job-picked", Title: "Backend Engineer", ContentFingerprint: "fp-picked", Source: "greenhouse"}
		cfg := dto.SearchConfig{UserID: "user-1", NotifyThreshold: 70, Preferences: dto.Preferences{
			Picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "manual"}},
		}}
		st.SeedJob(job, []dto.SearchConfig{cfg})
		st.SeedEffect(dto.AnswerEffect{ID: "effect-picked", JobID: job.ID, Fingerprint: job.ContentFingerprint, Attempts: 1})

		answerer := &fakeAnswerer{t: t}
		svc := scoring.NewService(scoring.Deps{Store: st, Answerer: answerer, Credentials: &fakeCredentials{key: "sk-or-test"}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}})

		if err := svc.RunTick(context.Background()); err != nil {
			t.Fatalf("RunTick: %v", err)
		}
		if len(answerer.calls) != 1 || !slices.Equal(answerer.calls[0], []string{"Does the role use Go?"}) {
			t.Fatalf("questions sent = %v, want exactly [%q] (only the picked question)", answerer.calls, "Does the role use Go?")
		}
	})

	t.Run("overlapping picks are unioned without duplicates", func(t *testing.T) {
		st := newFakeStore()
		job := dto.Job{ID: "job-union", Title: "Backend Engineer", ContentFingerprint: "fp-union", Source: "greenhouse"}
		cfg1 := dto.SearchConfig{UserID: "user-1", NotifyThreshold: 70, Preferences: dto.Preferences{
			Picks: []dto.Pick{
				{OptionID: "tech:go", Stance: "nice", Source: "manual"},
				{OptionID: "tech:rust", Stance: "avoid", Source: "manual"},
			},
		}}
		cfg2 := dto.SearchConfig{UserID: "user-2", NotifyThreshold: 70, Preferences: dto.Preferences{
			Picks: []dto.Pick{
				{OptionID: "tech:rust", Stance: "avoid", Source: "manual"},
				{OptionID: "role:backend", Stance: "nice", Source: "manual"},
			},
		}}
		st.SeedJob(job, []dto.SearchConfig{cfg1, cfg2})
		st.SeedEffect(dto.AnswerEffect{ID: "effect-union", JobID: job.ID, Fingerprint: job.ContentFingerprint, Attempts: 1})

		answerer := &fakeAnswerer{t: t}
		svc := scoring.NewService(scoring.Deps{Store: st, Answerer: answerer, Credentials: &fakeCredentials{key: "sk-or-test"}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}})

		if err := svc.RunTick(context.Background()); err != nil {
			t.Fatalf("RunTick: %v", err)
		}
		if len(answerer.calls) != 1 {
			t.Fatalf("Answer calls = %d, want 1 (one call covering the union)", len(answerer.calls))
		}
		got := slices.Clone(answerer.calls[0])
		slices.Sort(got)
		want := []string{"Does the role use Go?", "Does the role use Rust?", "Is this primarily a backend role?"}
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Fatalf("questions sent = %v, want %v (union, no duplicates)", got, want)
		}
	})

	t.Run("a retired option pick is never sent", func(t *testing.T) {
		st := newFakeStore()
		retiredAt := time.Now().Add(-time.Hour)
		st.SeedOptions(append(append([]dto.ScoringOption{}, bank...), dto.ScoringOption{
			ID: "tech:cobol", Dimension: dto.DimensionTech, Label: "COBOL", Question: "Does the role use COBOL?", RetiredAt: &retiredAt,
		}))
		job := dto.Job{ID: "job-retired", Title: "Backend Engineer", ContentFingerprint: "fp-retired", Source: "greenhouse"}
		cfg := dto.SearchConfig{UserID: "user-1", NotifyThreshold: 70, Preferences: dto.Preferences{
			Picks: []dto.Pick{
				{OptionID: "tech:go", Stance: "nice", Source: "manual"},
				{OptionID: "tech:cobol", Stance: "avoid", Source: "manual"},
			},
		}}
		st.SeedJob(job, []dto.SearchConfig{cfg})
		st.SeedEffect(dto.AnswerEffect{ID: "effect-retired", JobID: job.ID, Fingerprint: job.ContentFingerprint, Attempts: 1})

		answerer := &fakeAnswerer{t: t}
		svc := scoring.NewService(scoring.Deps{Store: st, Answerer: answerer, Credentials: &fakeCredentials{key: "sk-or-test"}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}})

		if err := svc.RunTick(context.Background()); err != nil {
			t.Fatalf("RunTick: %v", err)
		}
		if len(answerer.calls) != 1 || !slices.Equal(answerer.calls[0], []string{"Does the role use Go?"}) {
			t.Fatalf("questions sent = %v, want exactly [%q] (retired option never asked)", answerer.calls, "Does the role use Go?")
		}
	})

	t.Run("no picks skips the jev call and still writes a score", func(t *testing.T) {
		st := newFakeStore()
		job := dto.Job{ID: "job-nopicks", Title: "Backend Engineer", ContentFingerprint: "fp-nopicks", Source: "greenhouse"}
		cfg := dto.SearchConfig{UserID: "user-1", NotifyThreshold: 70}
		st.SeedJob(job, []dto.SearchConfig{cfg})
		st.SeedEffect(dto.AnswerEffect{ID: "effect-nopicks", JobID: job.ID, Fingerprint: job.ContentFingerprint, Attempts: 1})

		svc := scoring.NewService(scoring.Deps{Store: st, Answerer: &fakeAnswerer{t: t, forbidden: true}, Credentials: &fakeCredentials{key: "sk-or-test"}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}})

		if err := svc.RunTick(context.Background()); err != nil {
			t.Fatalf("RunTick: %v", err)
		}
		completed := st.Completed()
		if len(completed) != 1 || len(completed[0].Scores) != 1 {
			t.Fatalf("completed effects = %+v, want 1 effect with a score written from the prior", completed)
		}
	})

	t.Run("alerts only on first discovery above threshold", func(t *testing.T) {
		st := newFakeStore()
		job := dto.Job{ID: "job-3", Title: "Backend Engineer", ContentFingerprint: "fp-3", Source: "greenhouse"}
		cfg := dto.SearchConfig{UserID: "user-1", NotifyThreshold: 50, Preferences: dto.Preferences{
			Picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "manual"}},
		}}
		st.SeedJob(job, []dto.SearchConfig{cfg})
		st.SeedAnswers(job.ID, job.ContentFingerprint, "typesafe/jev-1.13", map[string]dto.Answer{
			scoring.QuestionHash("Does the role use Go?"):             {PYes: 0.9, PNo: 0.05, PNotStated: 0.05},
			scoring.QuestionHash("Does the role use Rust?"):           {PYes: 0.05, PNo: 0.9, PNotStated: 0.05},
			scoring.QuestionHash("Is this primarily a backend role?"): {PYes: 0.9, PNo: 0.05, PNotStated: 0.05},
		})
		st.SeedEffect(dto.AnswerEffect{ID: "effect-3", JobID: job.ID, Fingerprint: job.ContentFingerprint, Attempts: 1, FirstDiscovery: true})

		alerter := &fakeAlerter{}
		svc := scoring.NewService(scoring.Deps{Store: st, Answerer: &fakeAnswerer{t: t, forbidden: true}, Credentials: &fakeCredentials{key: "sk-or-test"}, Alerter: alerter, Profiles: &fakeProfiles{emails: map[string]string{"user-1": "user@example.com"}}})

		if err := svc.RunTick(context.Background()); err != nil {
			t.Fatalf("RunTick: %v", err)
		}
		if len(alerter.notified) != 1 || alerter.notified[0] != "user@example.com" {
			t.Fatalf("notified = %v, want [user@example.com]", alerter.notified)
		}
	})
}

type flakyClaimStore struct {
	*scoringtest.FakeStore
	failsLeft int
	completed chan struct{}
}

func (f *flakyClaimStore) ClaimAnswerEffect(ctx context.Context) (dto.AnswerEffect, error) {
	if f.failsLeft > 0 {
		f.failsLeft--
		return dto.AnswerEffect{}, errors.New("claim boom")
	}
	return f.FakeStore.ClaimAnswerEffect(ctx)
}

func (f *flakyClaimStore) CompleteAnswerEffect(ctx context.Context, effect dto.AnswerEffect, answers map[string]dto.Answer, scores []dto.JobScore) ([]string, error) {
	saved, err := f.FakeStore.CompleteAnswerEffect(ctx, effect, answers, scores)
	f.completed <- struct{}{}
	return saved, err
}

func TestRun(t *testing.T) {
	t.Run("returns when the context is cancelled", func(t *testing.T) {
		svc := scoring.NewService(scoring.Deps{Store: newFakeStore(), Answerer: &fakeAnswerer{t: t, forbidden: true}, Credentials: &fakeCredentials{}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}})

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- svc.Run(ctx) }()
		cancel()

		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("Run returned %v, want nil", err)
			}
		case <-time.After(time.Second):
			t.Fatal("Run did not return after context cancellation")
		}
	})

	t.Run("keeps ticking after a failed tick", func(t *testing.T) {
		job := dto.Job{ID: "job-4", Title: "Backend Engineer", ContentFingerprint: "fp-4", Source: "greenhouse"}
		cfg := dto.SearchConfig{UserID: "user-1", NotifyThreshold: 70, Preferences: dto.Preferences{
			Picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "manual"}},
		}}
		inner := newFakeStore()
		inner.SeedJob(job, []dto.SearchConfig{cfg})
		inner.SeedAnswers(job.ID, job.ContentFingerprint, "typesafe/jev-1.13", map[string]dto.Answer{
			scoring.QuestionHash("Does the role use Go?"):             {PYes: 0.9, PNo: 0.05, PNotStated: 0.05},
			scoring.QuestionHash("Does the role use Rust?"):           {PYes: 0.1, PNo: 0.85, PNotStated: 0.05},
			scoring.QuestionHash("Is this primarily a backend role?"): {PYes: 0.9, PNo: 0.05, PNotStated: 0.05},
		})
		inner.SeedEffect(dto.AnswerEffect{ID: "effect-4", JobID: job.ID, Fingerprint: job.ContentFingerprint, Attempts: 1})

		st := &flakyClaimStore{FakeStore: inner, failsLeft: 1, completed: make(chan struct{}, 1)}
		svc := scoring.NewService(scoring.Deps{Store: st, Answerer: &fakeAnswerer{t: t, forbidden: true}, Credentials: &fakeCredentials{key: "sk-or-test"}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}, TickInterval: time.Millisecond})

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- svc.Run(ctx) }()

		select {
		case <-st.completed:
		case <-time.After(time.Second):
			t.Fatal("effect never completed after the failed tick")
		}

		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("Run returned %v, want nil", err)
			}
		case <-time.After(time.Second):
			t.Fatal("Run did not return after context cancellation")
		}
	})
}

type pickCase struct {
	id        string
	dimension dto.Dimension
	stance    string
	retired   bool
	answer    dto.Answer
}

func nicePick(dim dto.Dimension, id string, pYes, pNo, pNotStated float64) pickCase {
	return pickCase{id: id, dimension: dim, stance: "nice", answer: dto.Answer{PYes: pYes, PNo: pNo, PNotStated: pNotStated}}
}

func avoidPick(id string, pYes, pNo, pNotStated float64) pickCase {
	return pickCase{id: id, dimension: dto.DimensionTech, stance: "avoid", answer: dto.Answer{PYes: pYes, PNo: pNo, PNotStated: pNotStated}}
}

func blockPick(id string, pYes, pNo, pNotStated float64) pickCase {
	return pickCase{id: id, dimension: dto.DimensionDomain, stance: "block", answer: dto.Answer{PYes: pYes, PNo: pNo, PNotStated: pNotStated}}
}

func retiredPick(dim dto.Dimension, id string) pickCase {
	return pickCase{id: id, dimension: dim, stance: "nice", retired: true, answer: dto.Answer{PYes: 0.9, PNo: 0.05, PNotStated: 0.05}}
}

func TestRecompute(t *testing.T) {
	t.Run("never calls the answerer or the alerter", func(t *testing.T) {
		st := newFakeStore()
		st.SeedSearchConfig(dto.SearchConfig{
			UserID: "user-1", NotifyThreshold: 70,
			Preferences: dto.Preferences{Picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "manual"}}},
		})
		st.SeedScoringInputs("user-1", []store.ScoringInput{
			{Job: dto.Job{ID: "job-1"}, Answers: map[string]dto.Answer{
				scoring.QuestionHash("Does the role use Go?"): {PYes: 0.9, PNo: 0.05, PNotStated: 0.05},
			}},
		})

		alerter := &fakeAlerter{}
		svc := scoring.NewService(scoring.Deps{Store: st, Answerer: &fakeAnswerer{t: t, forbidden: true}, Credentials: &fakeCredentials{}, Alerter: alerter, Profiles: &fakeProfiles{}})

		result, err := svc.Recompute(context.Background(), "user-1")
		if err != nil {
			t.Fatalf("Recompute: %v", err)
		}
		if result.Recomputed != 1 {
			t.Fatalf("recomputed = %d, want 1", result.Recomputed)
		}
		saved := st.Recomputed()
		if len(saved) != 1 || saved[0].Score != 63 {
			t.Fatalf("saved scores = %+v, want one score of 63", saved)
		}
		if len(alerter.notified) != 0 {
			t.Fatalf("alerter called on recompute: %v", alerter.notified)
		}
	})

	t.Run("a salary floor re-ranks without calling the answerer", func(t *testing.T) {
		st := newFakeStore()
		st.SeedSearchConfig(dto.SearchConfig{
			UserID: "user-1", NotifyThreshold: 70,
			Preferences: dto.Preferences{SalaryFloor: &dto.Money{Amount: 55000, Currency: "GBP"}},
		})
		st.SeedScoringInputs("user-1", []store.ScoringInput{
			{Job: dto.Job{ID: "job-1", SalaryRaw: "£40k"}, Answers: map[string]dto.Answer{}},
		})

		svc := scoring.NewService(scoring.Deps{Store: st, Answerer: &fakeAnswerer{t: t, forbidden: true}, Credentials: &fakeCredentials{}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}})

		result, err := svc.Recompute(context.Background(), "user-1")
		if err != nil {
			t.Fatalf("Recompute: %v", err)
		}
		if result.Recomputed != 1 {
			t.Fatalf("recomputed = %d, want 1", result.Recomputed)
		}
		saved := st.Recomputed()
		if len(saved) != 1 || saved[0].Score != 30 {
			t.Fatalf("saved scores = %+v, want one score of 30 (salary below floor)", saved)
		}
	})

	t.Run("a manual pick overrides a text pick on the same option", func(t *testing.T) {
		st := newFakeStore()
		st.SeedOptions([]dto.ScoringOption{
			{ID: "tech:kubernetes", Dimension: dto.DimensionTech, Label: "Kubernetes", Question: "Does the role use Kubernetes?"},
		})
		st.SeedSearchConfig(dto.SearchConfig{UserID: "user-1", Preferences: dto.Preferences{Picks: []dto.Pick{
			{OptionID: "tech:kubernetes", Stance: "avoid", Source: "text"},
			{OptionID: "tech:kubernetes", Stance: "nice", Source: "manual"},
		}}})
		st.SeedScoringInputs("user-1", []store.ScoringInput{
			{Job: dto.Job{ID: "job-1"}, Answers: map[string]dto.Answer{
				scoring.QuestionHash("Does the role use Kubernetes?"): {PYes: 0.9, PNo: 0.05, PNotStated: 0.05},
			}},
		})

		svc := scoring.NewService(scoring.Deps{Store: st, Answerer: &fakeAnswerer{t: t, forbidden: true}, Credentials: &fakeCredentials{}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}})

		if _, err := svc.Recompute(context.Background(), "user-1"); err != nil {
			t.Fatalf("Recompute: %v", err)
		}
		saved := st.Recomputed()
		if len(saved) != 1 || saved[0].Score != 63 {
			t.Fatalf("saved scores = %+v, want a nice-stance score of 63 (manual overrides text; a text avoid would score 21)", saved)
		}
	})

	t.Run("scores pick combinations and salary floors", func(t *testing.T) {
		cases := []struct {
			name      string
			picks     []pickCase
			salaryRaw string
			floor     *dto.Money
			wantScore int
			wantRows  []dto.ScoreRow
		}{
			{name: "nothing known scores 50", wantScore: 50},
			{
				name:      "one nice match scores 63",
				picks:     []pickCase{nicePick(dto.DimensionTech, "tech:go", 0.9, 0.05, 0.05)},
				wantScore: 63,
				wantRows:  []dto.ScoreRow{{Resolved: "yes", Effect: "meets"}},
			},
			{
				name: "four nice dimensions all matched scores 79",
				picks: []pickCase{
					nicePick(dto.DimensionTech, "tech:go", 0.9, 0.05, 0.05),
					nicePick(dto.DimensionRole, "role:backend", 0.9, 0.05, 0.05),
					nicePick(dto.DimensionSeniority, "seniority:senior", 0.9, 0.05, 0.05),
					nicePick(dto.DimensionWork, "work:remote", 0.9, 0.05, 0.05),
				},
				wantScore: 79,
			},
			{
				name: "two avoids hit, nothing else known scores 21",
				picks: []pickCase{
					avoidPick("tech:java", 0.9, 0.05, 0.05),
					avoidPick("tech:php", 0.9, 0.05, 0.05),
				},
				wantScore: 21,
			},
			{
				name: "two nice picks in one dimension with one matched counted once",
				picks: []pickCase{
					nicePick(dto.DimensionTech, "tech:go", 0.9, 0.05, 0.05),
					nicePick(dto.DimensionTech, "tech:rust", 0.05, 0.9, 0.05),
				},
				wantScore: 63,
				wantRows: []dto.ScoreRow{
					{Resolved: "yes", Effect: "meets"},
					{Resolved: "no", Effect: "misses"},
				},
			},
			{
				name:      "nice pick with every answer unknown is excluded",
				picks:     []pickCase{nicePick(dto.DimensionTech, "tech:go", 0.2, 0.2, 0.6)},
				wantScore: 50,
				wantRows:  []dto.ScoreRow{{Resolved: "unknown", Effect: "unknown"}},
			},
			{
				name:      "an avoid the job lacks is neutral",
				picks:     []pickCase{avoidPick("tech:java", 0.05, 0.9, 0.05)},
				wantScore: 50,
				wantRows:  []dto.ScoreRow{{Resolved: "no", Effect: "neutral"}},
			},
			{
				name:      "a top probability below 0.6 resolves unknown",
				picks:     []pickCase{nicePick(dto.DimensionTech, "tech:go", 0.55, 0.35, 0.1)},
				wantScore: 50,
				wantRows:  []dto.ScoreRow{{Resolved: "unknown", Effect: "unknown"}},
			},
			{
				name: "a block that resolves yes zeroes the score even with nice matches",
				picks: []pickCase{
					nicePick(dto.DimensionTech, "tech:go", 0.9, 0.05, 0.05),
					blockPick("domain:gambling", 0.9, 0.05, 0.05),
				},
				wantScore: 0,
				wantRows: []dto.ScoreRow{
					{Resolved: "yes", Effect: "meets"},
					{Resolved: "yes", Effect: "blocked"},
				},
			},
			{
				name:      "a block the job lacks is neutral",
				picks:     []pickCase{blockPick("domain:gambling", 0.05, 0.9, 0.05)},
				wantScore: 50,
				wantRows:  []dto.ScoreRow{{Resolved: "no", Effect: "neutral"}},
			},
			{
				name:      "a retired pick scores as if omitted",
				picks:     []pickCase{retiredPick(dto.DimensionTech, "tech:cobol")},
				wantScore: 50,
				wantRows:  []dto.ScoreRow{{Resolved: "retired", Effect: "retired"}},
			},
			{
				name: "a retired pick alongside a matched nice pick doesn't affect the score",
				picks: []pickCase{
					nicePick(dto.DimensionTech, "tech:go", 0.9, 0.05, 0.05),
					retiredPick(dto.DimensionRole, "role:backend"),
				},
				wantScore: 63,
				wantRows: []dto.ScoreRow{
					{Resolved: "yes", Effect: "meets"},
					{Resolved: "retired", Effect: "retired"},
				},
			},
			{
				name:      "salary below floor in the same currency is an avoid hit",
				salaryRaw: "£40k",
				floor:     &dto.Money{Amount: 55000, Currency: "GBP"},
				wantScore: 30,
				wantRows:  []dto.ScoreRow{{Resolved: "yes", Effect: "misses"}},
			},
			{
				name:      "salary above floor in the same currency is neutral",
				salaryRaw: "£70k",
				floor:     &dto.Money{Amount: 55000, Currency: "GBP"},
				wantScore: 50,
				wantRows:  []dto.ScoreRow{{Resolved: "no", Effect: "neutral"}},
			},
			{
				name:      "salary in a different currency is unknown",
				salaryRaw: "$70k",
				floor:     &dto.Money{Amount: 55000, Currency: "GBP"},
				wantScore: 50,
				wantRows:  []dto.ScoreRow{{Resolved: "unknown", Effect: "unknown"}},
			},
			{
				name:      "unparseable salary is unknown",
				salaryRaw: "Competitive",
				floor:     &dto.Money{Amount: 55000, Currency: "GBP"},
				wantScore: 50,
				wantRows:  []dto.ScoreRow{{Resolved: "unknown", Effect: "unknown"}},
			},
			{
				name:      "a salary range with a k suffix takes the upper bound",
				salaryRaw: "£55k–70k",
				floor:     &dto.Money{Amount: 60000, Currency: "GBP"},
				wantScore: 50,
				wantRows:  []dto.ScoreRow{{Resolved: "no", Effect: "neutral"}},
			},
			{
				name:      "thousands separators with a currency code and per annum take the upper bound",
				salaryRaw: "55,000 - 70,000 GBP per annum",
				floor:     &dto.Money{Amount: 60000, Currency: "GBP"},
				wantScore: 50,
				wantRows:  []dto.ScoreRow{{Resolved: "no", Effect: "neutral"}},
			},
			{
				name:      "a day rate is not treated as an annual salary",
				salaryRaw: "£600 per day",
				floor:     &dto.Money{Amount: 55000, Currency: "GBP"},
				wantScore: 50,
				wantRows:  []dto.ScoreRow{{Resolved: "unknown", Effect: "unknown"}},
			},
			{
				name:      "a dollar sign is detected as USD",
				salaryRaw: "$120k",
				floor:     &dto.Money{Amount: 100000, Currency: "USD"},
				wantScore: 50,
				wantRows:  []dto.ScoreRow{{Resolved: "no", Effect: "neutral"}},
			},
			{
				name:      "an empty salary is unknown",
				salaryRaw: "",
				floor:     &dto.Money{Amount: 55000, Currency: "GBP"},
				wantScore: 50,
				wantRows:  []dto.ScoreRow{{Resolved: "unknown", Effect: "unknown"}},
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				options := make([]dto.ScoringOption, 0, len(tc.picks))
				picks := make([]dto.Pick, 0, len(tc.picks))
				answers := make(map[string]dto.Answer, len(tc.picks))
				for _, p := range tc.picks {
					question := p.id + "?"
					opt := dto.ScoringOption{ID: p.id, Dimension: p.dimension, Label: p.id, Question: question}
					if p.retired {
						retiredAt := time.Now().Add(-time.Hour)
						opt.RetiredAt = &retiredAt
					}
					options = append(options, opt)
					picks = append(picks, dto.Pick{OptionID: p.id, Stance: p.stance, Source: "manual"})
					answers[scoring.QuestionHash(question)] = p.answer
				}

				st := newFakeStore()
				st.SeedOptions(options)
				st.SeedSearchConfig(dto.SearchConfig{
					UserID:      "user-1",
					Preferences: dto.Preferences{Picks: picks, SalaryFloor: tc.floor},
				})
				st.SeedScoringInputs("user-1", []store.ScoringInput{
					{Job: dto.Job{ID: "job-1", SalaryRaw: tc.salaryRaw}, Answers: answers},
				})

				svc := scoring.NewService(scoring.Deps{Store: st, Answerer: &fakeAnswerer{t: t, forbidden: true}, Credentials: &fakeCredentials{}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}})
				result, err := svc.Recompute(context.Background(), "user-1")
				if err != nil {
					t.Fatalf("Recompute: %v", err)
				}
				if result.Recomputed != 1 {
					t.Fatalf("recomputed = %d, want 1", result.Recomputed)
				}

				saved := st.Recomputed()
				if len(saved) != 1 {
					t.Fatalf("saved scores = %+v, want 1", saved)
				}
				if saved[0].Score != tc.wantScore {
					t.Errorf("score = %d, want %d", saved[0].Score, tc.wantScore)
				}
				for i, want := range tc.wantRows {
					if i >= len(saved[0].Rows) {
						t.Fatalf("rows[%d] missing, want %+v", i, want)
					}
					if saved[0].Rows[i].Resolved != want.Resolved || saved[0].Rows[i].Effect != want.Effect {
						t.Errorf("rows[%d] = {Resolved: %q, Effect: %q}, want {Resolved: %q, Effect: %q}",
							i, saved[0].Rows[i].Resolved, saved[0].Rows[i].Effect, want.Resolved, want.Effect)
					}
				}
			})
		}
	})
}

func TestFillMissingAnswers(t *testing.T) {
	t.Run("queues for the currently picked hashes", func(t *testing.T) {
		st := newFakeStore()
		st.SeedSearchConfig(dto.SearchConfig{
			UserID: "user-1",
			Preferences: dto.Preferences{Picks: []dto.Pick{
				{OptionID: "tech:go", Stance: "nice", Source: "manual"},
				{OptionID: "tech:rust", Stance: "avoid", Source: "manual"},
			}},
		})

		svc := scoring.NewService(scoring.Deps{Store: st, Answerer: &fakeAnswerer{t: t, forbidden: true}, Credentials: &fakeCredentials{}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}})

		queued, err := svc.FillMissingAnswers(context.Background(), "user-1")
		if err != nil {
			t.Fatalf("FillMissingAnswers: %v", err)
		}
		if queued != 2 {
			t.Fatalf("queued = %d, want 2", queued)
		}
		calls := st.QueuedMissing()
		if len(calls) != 1 {
			t.Fatalf("queue calls = %d, want 1", len(calls))
		}
		call := calls[0]
		if call.UserID != "user-1" || call.Model != jev.Model {
			t.Fatalf("call = %+v, want user-1 with model %q", call, jev.Model)
		}
		wantHashes := map[string]bool{
			scoring.QuestionHash("Does the role use Go?"):   true,
			scoring.QuestionHash("Does the role use Rust?"): true,
		}
		if len(call.Hashes) != len(wantHashes) {
			t.Fatalf("hashes = %v, want %v", call.Hashes, wantHashes)
		}
		for _, h := range call.Hashes {
			if !wantHashes[h] {
				t.Fatalf("unexpected hash %q", h)
			}
		}
	})

	t.Run("no picks queues nothing", func(t *testing.T) {
		st := newFakeStore()
		st.SeedSearchConfig(dto.SearchConfig{UserID: "user-1"})

		svc := scoring.NewService(scoring.Deps{Store: st, Answerer: &fakeAnswerer{t: t, forbidden: true}, Credentials: &fakeCredentials{}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}})

		queued, err := svc.FillMissingAnswers(context.Background(), "user-1")
		if err != nil {
			t.Fatalf("FillMissingAnswers: %v", err)
		}
		if queued != 0 || len(st.QueuedMissing()) != 0 {
			t.Fatalf("queued = %d, calls = %d, want 0 and no store call", queued, len(st.QueuedMissing()))
		}
	})

	t.Run("no search config queues nothing", func(t *testing.T) {
		st := newFakeStore()
		svc := scoring.NewService(scoring.Deps{Store: st, Answerer: &fakeAnswerer{t: t, forbidden: true}, Credentials: &fakeCredentials{}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}})

		queued, err := svc.FillMissingAnswers(context.Background(), "user-1")
		if err != nil {
			t.Fatalf("FillMissingAnswers: %v", err)
		}
		if queued != 0 {
			t.Fatalf("queued = %d, want 0", queued)
		}
	})

	t.Run("a retired pick is never asked", func(t *testing.T) {
		st := newFakeStore()
		retired := time.Now()
		st.SeedOptions(append(slices.Clone(bank), dto.ScoringOption{
			ID: "tech:cobol", Dimension: dto.DimensionTech, Label: "COBOL",
			Question: "Does the role use COBOL?", RetiredAt: &retired,
		}))
		st.SeedSearchConfig(dto.SearchConfig{
			UserID:      "user-1",
			Preferences: dto.Preferences{Picks: []dto.Pick{{OptionID: "tech:cobol", Stance: "nice", Source: "manual"}}},
		})

		svc := scoring.NewService(scoring.Deps{Store: st, Answerer: &fakeAnswerer{t: t, forbidden: true}, Credentials: &fakeCredentials{}, Alerter: &fakeAlerter{}, Profiles: &fakeProfiles{}})

		queued, err := svc.FillMissingAnswers(context.Background(), "user-1")
		if err != nil {
			t.Fatalf("FillMissingAnswers: %v", err)
		}
		if queued != 0 || len(st.QueuedMissing()) != 0 {
			t.Fatalf("queued = %d, calls = %d, want 0 (retired option never asked)", queued, len(st.QueuedMissing()))
		}
	})
}

func TestAsk(t *testing.T) {
	ctx := context.Background()
	job := dto.Job{ID: "job-1", ContentFingerprint: "fp-1"}
	newSvc := func(t *testing.T, key string) (*scoring.Service, *fakeAnswerer) {
		t.Helper()
		st := newFakeStore()
		st.SeedJob(job, nil)
		answerer := &fakeAnswerer{t: t}
		return scoring.NewService(scoring.Deps{Store: st, Answerer: answerer, Credentials: &fakeCredentials{key: key}}), answerer
	}

	t.Run("second ask with the same questions makes no Jev call", func(t *testing.T) {
		svc, answerer := newSvc(t, "sk-or-test")
		qs := []string{"Is it remote?", "Is it senior?"}
		first, err := svc.Ask(ctx, "user-1", job.ID, qs)
		if err != nil {
			t.Fatal(err)
		}
		second, err := svc.Ask(ctx, "user-1", job.ID, qs)
		if err != nil {
			t.Fatal(err)
		}
		if len(answerer.calls) != 1 {
			t.Fatalf("Answer calls = %v, want exactly one", answerer.calls)
		}
		if len(second) != 2 || second["Is it remote?"] != first["Is it remote?"] {
			t.Fatalf("second = %+v, want the cached answers %+v", second, first)
		}
	})

	t.Run("changing one question asks only that question", func(t *testing.T) {
		svc, answerer := newSvc(t, "sk-or-test")
		if _, err := svc.Ask(ctx, "user-1", job.ID, []string{"Is it remote?", "Is it senior?"}); err != nil {
			t.Fatal(err)
		}
		got, err := svc.Ask(ctx, "user-1", job.ID, []string{"Is it remote?", "Is it junior?"})
		if err != nil {
			t.Fatal(err)
		}
		if len(answerer.calls) != 2 || !slices.Equal(answerer.calls[1], []string{"Is it junior?"}) {
			t.Fatalf("Answer calls = %v, want a second call for only %q", answerer.calls, "Is it junior?")
		}
		if len(got) != 2 {
			t.Fatalf("got %+v, want answers for both questions", got)
		}
	})

	t.Run("a user with no key gets an unprocessable error", func(t *testing.T) {
		svc, _ := newSvc(t, "")
		_, err := svc.Ask(ctx, "user-1", job.ID, []string{"Is it remote?"})
		ae, ok := errors.AsType[*apperr.Error](err)
		if !ok || ae.Kind() != apperr.KindUnprocessable {
			t.Fatalf("Ask(...) err = %v, want an unprocessable apperr", err)
		}
	})

	t.Run("an unknown job is not found", func(t *testing.T) {
		svc, _ := newSvc(t, "sk-or-test")
		_, err := svc.Ask(ctx, "user-1", "missing", []string{"Is it remote?"})
		ae, ok := errors.AsType[*apperr.Error](err)
		if !ok || ae.Kind() != apperr.KindNotFound {
			t.Fatalf("Ask(...) err = %v, want a not-found apperr", err)
		}
	})
}

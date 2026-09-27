package suitability

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

var bank = []dto.ScoringOption{
	{ID: "tech:go", Dimension: dto.DimensionTech, Label: "Go", Question: "Does the role use Go?"},
	{ID: "tech:rust", Dimension: dto.DimensionTech, Label: "Rust", Question: "Does the role use Rust?"},
	{ID: "role:backend", Dimension: dto.DimensionRole, Label: "Backend", Question: "Is this primarily a backend role?"},
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

type fakeCredentials struct{ key string }

func (f *fakeCredentials) Get(context.Context, string, string) (string, error) {
	if f.key == "" {
		return "", errors.New("no credential")
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

func newOptions() *providers.MockScoringOptionsProvider {
	options := providers.NewMockScoringOptionsProvider()
	options.Seed(bank)
	return options
}

func TestProcess_AllAnswersCachedMakesNoJevCall(t *testing.T) {
	store := providers.NewMockSuitabilityProvider()
	job := dto.Job{ID: "job-1", Title: "Backend Engineer", ContentFingerprint: "fp-1", Source: "greenhouse"}
	cfg := dto.SearchConfig{UserID: "user-1", NotifyThreshold: 70, Preferences: dto.Preferences{
		Picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "manual"}},
	}}
	store.SeedJob(job, []dto.SearchConfig{cfg})
	store.SeedAnswers(job.ID, job.ContentFingerprint, "typesafe/jev-1.13", map[string]dto.Answer{
		questionHash("Does the role use Go?"):             {PYes: 0.9, PNo: 0.05, PNotStated: 0.05},
		questionHash("Does the role use Rust?"):           {PYes: 0.1, PNo: 0.85, PNotStated: 0.05},
		questionHash("Is this primarily a backend role?"): {PYes: 0.9, PNo: 0.05, PNotStated: 0.05},
	})
	store.SeedEffect(dto.AnswerEffect{ID: "effect-1", JobID: job.ID, Fingerprint: job.ContentFingerprint, Attempts: 1, FirstDiscovery: true})

	answerer := &fakeAnswerer{t: t, forbidden: true}
	svc := New(store, newOptions(), providers.NewMockSearchConfigProvider(), answerer,
		&fakeCredentials{key: "sk-or-test"}, &fakeAlerter{}, &fakeProfiles{})

	if err := svc.RunTick(context.Background()); err != nil {
		t.Fatalf("RunTick: %v", err)
	}
	if len(store.Completed) != 1 {
		t.Fatalf("completed effects = %d, want 1", len(store.Completed))
	}
	if len(store.Completed[0].Answers) != 0 {
		t.Fatalf("new answers written = %d, want 0 (everything cached)", len(store.Completed[0].Answers))
	}
}

func TestProcess_MissingQuestionsSendsExactlyThose(t *testing.T) {
	store := providers.NewMockSuitabilityProvider()
	job := dto.Job{ID: "job-2", Title: "Backend Engineer", ContentFingerprint: "fp-2", Source: "greenhouse"}
	cfg := dto.SearchConfig{UserID: "user-1", NotifyThreshold: 70, Preferences: dto.Preferences{
		Picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "manual"}},
	}}
	store.SeedJob(job, []dto.SearchConfig{cfg})
	store.SeedEffect(dto.AnswerEffect{ID: "effect-2", JobID: job.ID, Fingerprint: job.ContentFingerprint, Attempts: 1})

	answerer := &fakeAnswerer{t: t}
	svc := New(store, newOptions(), providers.NewMockSearchConfigProvider(), answerer,
		&fakeCredentials{key: "sk-or-test"}, &fakeAlerter{}, &fakeProfiles{})

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
	if len(store.Completed) != 1 || len(store.Completed[0].Answers) != 3 {
		t.Fatalf("completed effects = %+v, want 1 effect with 3 new answers", store.Completed)
	}
}

func TestProcess_AlertsOnlyOnFirstDiscoveryAboveThreshold(t *testing.T) {
	store := providers.NewMockSuitabilityProvider()
	job := dto.Job{ID: "job-3", Title: "Backend Engineer", ContentFingerprint: "fp-3", Source: "greenhouse"}
	cfg := dto.SearchConfig{UserID: "user-1", NotifyThreshold: 50, Preferences: dto.Preferences{
		Picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "manual"}},
	}}
	store.SeedJob(job, []dto.SearchConfig{cfg})
	store.SeedAnswers(job.ID, job.ContentFingerprint, "typesafe/jev-1.13", map[string]dto.Answer{
		questionHash("Does the role use Go?"):             {PYes: 0.9, PNo: 0.05, PNotStated: 0.05},
		questionHash("Does the role use Rust?"):           {PYes: 0.05, PNo: 0.9, PNotStated: 0.05},
		questionHash("Is this primarily a backend role?"): {PYes: 0.9, PNo: 0.05, PNotStated: 0.05},
	})
	store.SeedEffect(dto.AnswerEffect{ID: "effect-3", JobID: job.ID, Fingerprint: job.ContentFingerprint, Attempts: 1, FirstDiscovery: true})

	alerter := &fakeAlerter{}
	svc := New(store, newOptions(), providers.NewMockSearchConfigProvider(),
		&fakeAnswerer{t: t, forbidden: true}, &fakeCredentials{key: "sk-or-test"}, alerter,
		&fakeProfiles{emails: map[string]string{"user-1": "user@example.com"}})

	if err := svc.RunTick(context.Background()); err != nil {
		t.Fatalf("RunTick: %v", err)
	}
	if len(alerter.notified) != 1 || alerter.notified[0] != "user@example.com" {
		t.Fatalf("notified = %v, want [user@example.com]", alerter.notified)
	}
}

type flakyClaimStore struct {
	*providers.MockSuitabilityProvider
	failsLeft int
	completed chan struct{}
}

func (f *flakyClaimStore) ClaimAnswerEffect(ctx context.Context) (dto.AnswerEffect, error) {
	if f.failsLeft > 0 {
		f.failsLeft--
		return dto.AnswerEffect{}, errors.New("claim boom")
	}
	return f.MockSuitabilityProvider.ClaimAnswerEffect(ctx)
}

func (f *flakyClaimStore) CompleteAnswerEffect(ctx context.Context, effect dto.AnswerEffect, answers map[string]dto.Answer, scores []dto.JobScore) ([]string, error) {
	saved, err := f.MockSuitabilityProvider.CompleteAnswerEffect(ctx, effect, answers, scores)
	f.completed <- struct{}{}
	return saved, err
}

func TestRun_ReturnsWhenContextCancelled(t *testing.T) {
	svc := New(providers.NewMockSuitabilityProvider(), newOptions(), providers.NewMockSearchConfigProvider(),
		&fakeAnswerer{t: t, forbidden: true}, &fakeCredentials{}, &fakeAlerter{}, &fakeProfiles{})

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
}

func TestRun_KeepsTickingAfterFailedTick(t *testing.T) {
	original := answerEffectTickInterval
	answerEffectTickInterval = time.Millisecond
	t.Cleanup(func() { answerEffectTickInterval = original })

	job := dto.Job{ID: "job-4", Title: "Backend Engineer", ContentFingerprint: "fp-4", Source: "greenhouse"}
	cfg := dto.SearchConfig{UserID: "user-1", NotifyThreshold: 70, Preferences: dto.Preferences{
		Picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "manual"}},
	}}
	inner := providers.NewMockSuitabilityProvider()
	inner.SeedJob(job, []dto.SearchConfig{cfg})
	inner.SeedAnswers(job.ID, job.ContentFingerprint, "typesafe/jev-1.13", map[string]dto.Answer{
		questionHash("Does the role use Go?"):             {PYes: 0.9, PNo: 0.05, PNotStated: 0.05},
		questionHash("Does the role use Rust?"):           {PYes: 0.1, PNo: 0.85, PNotStated: 0.05},
		questionHash("Is this primarily a backend role?"): {PYes: 0.9, PNo: 0.05, PNotStated: 0.05},
	})
	inner.SeedEffect(dto.AnswerEffect{ID: "effect-4", JobID: job.ID, Fingerprint: job.ContentFingerprint, Attempts: 1})

	store := &flakyClaimStore{MockSuitabilityProvider: inner, failsLeft: 1, completed: make(chan struct{}, 1)}
	svc := New(store, newOptions(), providers.NewMockSearchConfigProvider(),
		&fakeAnswerer{t: t, forbidden: true}, &fakeCredentials{key: "sk-or-test"}, &fakeAlerter{}, &fakeProfiles{})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- svc.Run(ctx) }()

	select {
	case <-store.completed:
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
}

func TestRecompute_NeverCallsAnswererOrAlerter(t *testing.T) {
	store := providers.NewMockSuitabilityProvider()
	configs := providers.NewMockSearchConfigProvider()
	if _, err := configs.UpsertSearchConfig(context.Background(), dto.SearchConfig{
		UserID: "user-1", NotifyThreshold: 70,
		Preferences: dto.Preferences{Picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "manual"}}},
	}); err != nil {
		t.Fatal(err)
	}
	store.SeedScoringInputs("user-1", []providers.ScoringInput{
		{Job: dto.Job{ID: "job-1"}, Answers: map[string]dto.Answer{
			questionHash("Does the role use Go?"): {PYes: 0.9, PNo: 0.05, PNotStated: 0.05},
		}},
	})

	alerter := &fakeAlerter{}
	svc := New(store, newOptions(), configs, &fakeAnswerer{t: t, forbidden: true},
		&fakeCredentials{}, alerter, &fakeProfiles{})

	result, err := svc.Recompute(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if result.Recomputed != 1 {
		t.Fatalf("recomputed = %d, want 1", result.Recomputed)
	}
	if len(store.Saved) != 1 || store.Saved[0].Score != 63 {
		t.Fatalf("saved scores = %+v, want one score of 63", store.Saved)
	}
	if len(alerter.notified) != 0 {
		t.Fatalf("alerter called on recompute: %v", alerter.notified)
	}
}

func TestRecompute_SalaryFloorReRanksWithoutAnswerer(t *testing.T) {
	store := providers.NewMockSuitabilityProvider()
	configs := providers.NewMockSearchConfigProvider()
	if _, err := configs.UpsertSearchConfig(context.Background(), dto.SearchConfig{
		UserID: "user-1", NotifyThreshold: 70,
		Preferences: dto.Preferences{SalaryFloor: &dto.Money{Amount: 55000, Currency: "GBP"}},
	}); err != nil {
		t.Fatal(err)
	}
	store.SeedScoringInputs("user-1", []providers.ScoringInput{
		{Job: dto.Job{ID: "job-1", SalaryRaw: "£40k"}, Answers: map[string]dto.Answer{}},
	})

	svc := New(store, newOptions(), configs, &fakeAnswerer{t: t, forbidden: true},
		&fakeCredentials{}, &fakeAlerter{}, &fakeProfiles{})

	result, err := svc.Recompute(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if result.Recomputed != 1 {
		t.Fatalf("recomputed = %d, want 1", result.Recomputed)
	}
	if len(store.Saved) != 1 || store.Saved[0].Score != 30 {
		t.Fatalf("saved scores = %+v, want one score of 30 (salary below floor)", store.Saved)
	}
}

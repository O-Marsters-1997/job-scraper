package scoring

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/store"
)

var bank = []dto.ScoringOption{
	{ID: "tech:go", Dimension: dto.DimensionTech, Label: "Go", Question: "Does the role use Go?"},
	{ID: "tech:rust", Dimension: dto.DimensionTech, Label: "Rust", Question: "Does the role use Rust?"},
	{ID: "role:backend", Dimension: dto.DimensionRole, Label: "Backend", Question: "Is this primarily a backend role?"},
}

type fakeStore struct {
	mu sync.Mutex

	effects []dto.AnswerEffect
	jobs    map[string]dto.Job
	configs map[string][]dto.SearchConfig
	answers map[string]map[string]dto.Answer
	inputs  map[string][]store.ScoringInput
	options []dto.ScoringOption
	search  map[string]dto.SearchConfig

	ClaimErr error

	Failed    []dto.ScoringFailure
	Completed []completedEffect
	Saved     []dto.JobScore
}

type completedEffect struct {
	Effect  dto.AnswerEffect
	Answers map[string]dto.Answer
	Scores  []dto.JobScore
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		jobs:    make(map[string]dto.Job),
		configs: make(map[string][]dto.SearchConfig),
		answers: make(map[string]map[string]dto.Answer),
		inputs:  make(map[string][]store.ScoringInput),
		search:  make(map[string]dto.SearchConfig),
		options: bank,
	}
}

func answerKey(jobID, fingerprint, model string) string {
	return jobID + "|" + fingerprint + "|" + model
}

func (f *fakeStore) SeedEffect(effect dto.AnswerEffect) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.effects = append(f.effects, effect)
}

func (f *fakeStore) SeedJob(job dto.Job, configs []dto.SearchConfig) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.jobs[job.ID] = job
	f.configs[job.ID] = configs
}

func (f *fakeStore) SeedAnswers(jobID, fingerprint, model string, answers map[string]dto.Answer) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.answers[answerKey(jobID, fingerprint, model)] = answers
}

func (f *fakeStore) SeedScoringInputs(userID string, inputs []store.ScoringInput) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.inputs[userID] = inputs
}

func (f *fakeStore) ClaimAnswerEffect(context.Context) (dto.AnswerEffect, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.ClaimErr != nil {
		return dto.AnswerEffect{}, f.ClaimErr
	}
	if len(f.effects) == 0 {
		return dto.AnswerEffect{}, store.ErrNotFound
	}
	e := f.effects[0]
	f.effects = f.effects[1:]
	return e, nil
}

func (f *fakeStore) FailAnswerEffect(_ context.Context, _ string, _ int, failure dto.ScoringFailure) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Failed = append(f.Failed, failure)
	return nil
}

func (f *fakeStore) GetJobForScoring(_ context.Context, jobID string) (dto.Job, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	job, ok := f.jobs[jobID]
	if !ok {
		return dto.Job{}, store.ErrNotFound
	}
	return job, nil
}

func (f *fakeStore) ListInterestedConfigs(_ context.Context, jobID string, _ bool) ([]dto.SearchConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.configs[jobID], nil
}

func (f *fakeStore) ListScoringOptions(context.Context) ([]dto.ScoringOption, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]dto.ScoringOption, len(f.options))
	copy(out, f.options)
	return out, nil
}

func (f *fakeStore) ListAnswers(_ context.Context, jobID, fingerprint, model string) (map[string]dto.Answer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := answerKey(jobID, fingerprint, model)
	out := make(map[string]dto.Answer, len(f.answers[key]))
	for h, a := range f.answers[key] {
		out[h] = a
	}
	return out, nil
}

func (f *fakeStore) CompleteAnswerEffect(_ context.Context, effect dto.AnswerEffect, answers map[string]dto.Answer, scores []dto.JobScore) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Completed = append(f.Completed, completedEffect{Effect: effect, Answers: answers, Scores: scores})
	saved := make([]string, len(scores))
	for i, s := range scores {
		saved[i] = s.UserID
	}
	return saved, nil
}

func (f *fakeStore) GetSearchConfig(_ context.Context, userID string) (dto.SearchConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	cfg, ok := f.search[userID]
	if !ok {
		return dto.SearchConfig{}, store.ErrNotFound
	}
	return cfg, nil
}

func (f *fakeStore) UpsertSearchConfig(cfg dto.SearchConfig) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.search[cfg.UserID] = cfg
}

func (f *fakeStore) ListScoringInputs(_ context.Context, userID, _ string) ([]store.ScoringInput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.inputs[userID], nil
}

func (f *fakeStore) SaveScores(_ context.Context, scores []dto.JobScore) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Saved = append(f.Saved, scores...)
	return nil
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

func TestProcess_AllAnswersCachedMakesNoJevCall(t *testing.T) {
	st := newFakeStore()
	job := dto.Job{ID: "job-1", Title: "Backend Engineer", ContentFingerprint: "fp-1", Source: "greenhouse"}
	cfg := dto.SearchConfig{UserID: "user-1", NotifyThreshold: 70, Preferences: dto.Preferences{
		Picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "manual"}},
	}}
	st.SeedJob(job, []dto.SearchConfig{cfg})
	st.SeedAnswers(job.ID, job.ContentFingerprint, "typesafe/jev-1.13", map[string]dto.Answer{
		questionHash("Does the role use Go?"):             {PYes: 0.9, PNo: 0.05, PNotStated: 0.05},
		questionHash("Does the role use Rust?"):           {PYes: 0.1, PNo: 0.85, PNotStated: 0.05},
		questionHash("Is this primarily a backend role?"): {PYes: 0.9, PNo: 0.05, PNotStated: 0.05},
	})
	st.SeedEffect(dto.AnswerEffect{ID: "effect-1", JobID: job.ID, Fingerprint: job.ContentFingerprint, Attempts: 1, FirstDiscovery: true})

	answerer := &fakeAnswerer{t: t, forbidden: true}
	svc := NewService(st, answerer, &fakeCredentials{key: "sk-or-test"}, &fakeAlerter{}, &fakeProfiles{})

	if err := svc.RunTick(context.Background()); err != nil {
		t.Fatalf("RunTick: %v", err)
	}
	if len(st.Completed) != 1 {
		t.Fatalf("completed effects = %d, want 1", len(st.Completed))
	}
	if len(st.Completed[0].Answers) != 0 {
		t.Fatalf("new answers written = %d, want 0 (everything cached)", len(st.Completed[0].Answers))
	}
}

func TestProcess_MissingQuestionsSendsExactlyThose(t *testing.T) {
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
	svc := NewService(st, answerer, &fakeCredentials{key: "sk-or-test"}, &fakeAlerter{}, &fakeProfiles{})

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
	if len(st.Completed) != 1 || len(st.Completed[0].Answers) != 3 {
		t.Fatalf("completed effects = %+v, want 1 effect with 3 new answers", st.Completed)
	}
}

func TestProcess_OnlyPickedQuestionsSent(t *testing.T) {
	st := newFakeStore()
	job := dto.Job{ID: "job-picked", Title: "Backend Engineer", ContentFingerprint: "fp-picked", Source: "greenhouse"}
	cfg := dto.SearchConfig{UserID: "user-1", NotifyThreshold: 70, Preferences: dto.Preferences{
		Picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "manual"}},
	}}
	st.SeedJob(job, []dto.SearchConfig{cfg})
	st.SeedEffect(dto.AnswerEffect{ID: "effect-picked", JobID: job.ID, Fingerprint: job.ContentFingerprint, Attempts: 1})

	answerer := &fakeAnswerer{t: t}
	svc := NewService(st, answerer, &fakeCredentials{key: "sk-or-test"}, &fakeAlerter{}, &fakeProfiles{})

	if err := svc.RunTick(context.Background()); err != nil {
		t.Fatalf("RunTick: %v", err)
	}
	if len(answerer.calls) != 1 || !slices.Equal(answerer.calls[0], []string{"Does the role use Go?"}) {
		t.Fatalf("questions sent = %v, want exactly [%q] (only the picked question)", answerer.calls, "Does the role use Go?")
	}
}

func TestProcess_UnionOfOverlappingPicksNoDuplicates(t *testing.T) {
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
	svc := NewService(st, answerer, &fakeCredentials{key: "sk-or-test"}, &fakeAlerter{}, &fakeProfiles{})

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
}

func TestProcess_RetiredOptionPickNeverSent(t *testing.T) {
	st := newFakeStore()
	retiredAt := time.Now().Add(-time.Hour)
	st.options = append(append([]dto.ScoringOption{}, bank...), dto.ScoringOption{
		ID: "tech:cobol", Dimension: dto.DimensionTech, Label: "COBOL", Question: "Does the role use COBOL?", RetiredAt: &retiredAt,
	})
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
	svc := NewService(st, answerer, &fakeCredentials{key: "sk-or-test"}, &fakeAlerter{}, &fakeProfiles{})

	if err := svc.RunTick(context.Background()); err != nil {
		t.Fatalf("RunTick: %v", err)
	}
	if len(answerer.calls) != 1 || !slices.Equal(answerer.calls[0], []string{"Does the role use Go?"}) {
		t.Fatalf("questions sent = %v, want exactly [%q] (retired option never asked)", answerer.calls, "Does the role use Go?")
	}
}

func TestProcess_NoPicksSkipsJevCallAndStillWritesScore(t *testing.T) {
	st := newFakeStore()
	job := dto.Job{ID: "job-nopicks", Title: "Backend Engineer", ContentFingerprint: "fp-nopicks", Source: "greenhouse"}
	cfg := dto.SearchConfig{UserID: "user-1", NotifyThreshold: 70}
	st.SeedJob(job, []dto.SearchConfig{cfg})
	st.SeedEffect(dto.AnswerEffect{ID: "effect-nopicks", JobID: job.ID, Fingerprint: job.ContentFingerprint, Attempts: 1})

	svc := NewService(st, &fakeAnswerer{t: t, forbidden: true}, &fakeCredentials{key: "sk-or-test"}, &fakeAlerter{}, &fakeProfiles{})

	if err := svc.RunTick(context.Background()); err != nil {
		t.Fatalf("RunTick: %v", err)
	}
	if len(st.Completed) != 1 || len(st.Completed[0].Scores) != 1 {
		t.Fatalf("completed effects = %+v, want 1 effect with a score written from the prior", st.Completed)
	}
}

func TestProcess_AlertsOnlyOnFirstDiscoveryAboveThreshold(t *testing.T) {
	st := newFakeStore()
	job := dto.Job{ID: "job-3", Title: "Backend Engineer", ContentFingerprint: "fp-3", Source: "greenhouse"}
	cfg := dto.SearchConfig{UserID: "user-1", NotifyThreshold: 50, Preferences: dto.Preferences{
		Picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "manual"}},
	}}
	st.SeedJob(job, []dto.SearchConfig{cfg})
	st.SeedAnswers(job.ID, job.ContentFingerprint, "typesafe/jev-1.13", map[string]dto.Answer{
		questionHash("Does the role use Go?"):             {PYes: 0.9, PNo: 0.05, PNotStated: 0.05},
		questionHash("Does the role use Rust?"):           {PYes: 0.05, PNo: 0.9, PNotStated: 0.05},
		questionHash("Is this primarily a backend role?"): {PYes: 0.9, PNo: 0.05, PNotStated: 0.05},
	})
	st.SeedEffect(dto.AnswerEffect{ID: "effect-3", JobID: job.ID, Fingerprint: job.ContentFingerprint, Attempts: 1, FirstDiscovery: true})

	alerter := &fakeAlerter{}
	svc := NewService(st, &fakeAnswerer{t: t, forbidden: true}, &fakeCredentials{key: "sk-or-test"}, alerter,
		&fakeProfiles{emails: map[string]string{"user-1": "user@example.com"}})

	if err := svc.RunTick(context.Background()); err != nil {
		t.Fatalf("RunTick: %v", err)
	}
	if len(alerter.notified) != 1 || alerter.notified[0] != "user@example.com" {
		t.Fatalf("notified = %v, want [user@example.com]", alerter.notified)
	}
}

type flakyClaimStore struct {
	*fakeStore
	failsLeft int
	completed chan struct{}
}

func (f *flakyClaimStore) ClaimAnswerEffect(ctx context.Context) (dto.AnswerEffect, error) {
	if f.failsLeft > 0 {
		f.failsLeft--
		return dto.AnswerEffect{}, errors.New("claim boom")
	}
	return f.fakeStore.ClaimAnswerEffect(ctx)
}

func (f *flakyClaimStore) CompleteAnswerEffect(ctx context.Context, effect dto.AnswerEffect, answers map[string]dto.Answer, scores []dto.JobScore) ([]string, error) {
	saved, err := f.fakeStore.CompleteAnswerEffect(ctx, effect, answers, scores)
	f.completed <- struct{}{}
	return saved, err
}

func TestRun_ReturnsWhenContextCancelled(t *testing.T) {
	svc := NewService(newFakeStore(), &fakeAnswerer{t: t, forbidden: true}, &fakeCredentials{}, &fakeAlerter{}, &fakeProfiles{})

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
	inner := newFakeStore()
	inner.SeedJob(job, []dto.SearchConfig{cfg})
	inner.SeedAnswers(job.ID, job.ContentFingerprint, "typesafe/jev-1.13", map[string]dto.Answer{
		questionHash("Does the role use Go?"):             {PYes: 0.9, PNo: 0.05, PNotStated: 0.05},
		questionHash("Does the role use Rust?"):           {PYes: 0.1, PNo: 0.85, PNotStated: 0.05},
		questionHash("Is this primarily a backend role?"): {PYes: 0.9, PNo: 0.05, PNotStated: 0.05},
	})
	inner.SeedEffect(dto.AnswerEffect{ID: "effect-4", JobID: job.ID, Fingerprint: job.ContentFingerprint, Attempts: 1})

	st := &flakyClaimStore{fakeStore: inner, failsLeft: 1, completed: make(chan struct{}, 1)}
	svc := NewService(st, &fakeAnswerer{t: t, forbidden: true}, &fakeCredentials{key: "sk-or-test"}, &fakeAlerter{}, &fakeProfiles{})

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
}

func TestRecompute_NeverCallsAnswererOrAlerter(t *testing.T) {
	st := newFakeStore()
	st.UpsertSearchConfig(dto.SearchConfig{
		UserID: "user-1", NotifyThreshold: 70,
		Preferences: dto.Preferences{Picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "manual"}}},
	})
	st.SeedScoringInputs("user-1", []store.ScoringInput{
		{Job: dto.Job{ID: "job-1"}, Answers: map[string]dto.Answer{
			questionHash("Does the role use Go?"): {PYes: 0.9, PNo: 0.05, PNotStated: 0.05},
		}},
	})

	alerter := &fakeAlerter{}
	svc := NewService(st, &fakeAnswerer{t: t, forbidden: true}, &fakeCredentials{}, alerter, &fakeProfiles{})

	result, err := svc.Recompute(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if result.Recomputed != 1 {
		t.Fatalf("recomputed = %d, want 1", result.Recomputed)
	}
	if len(st.Saved) != 1 || st.Saved[0].Score != 63 {
		t.Fatalf("saved scores = %+v, want one score of 63", st.Saved)
	}
	if len(alerter.notified) != 0 {
		t.Fatalf("alerter called on recompute: %v", alerter.notified)
	}
}

func TestRecompute_SalaryFloorReRanksWithoutAnswerer(t *testing.T) {
	st := newFakeStore()
	st.UpsertSearchConfig(dto.SearchConfig{
		UserID: "user-1", NotifyThreshold: 70,
		Preferences: dto.Preferences{SalaryFloor: &dto.Money{Amount: 55000, Currency: "GBP"}},
	})
	st.SeedScoringInputs("user-1", []store.ScoringInput{
		{Job: dto.Job{ID: "job-1", SalaryRaw: "£40k"}, Answers: map[string]dto.Answer{}},
	})

	svc := NewService(st, &fakeAnswerer{t: t, forbidden: true}, &fakeCredentials{}, &fakeAlerter{}, &fakeProfiles{})

	result, err := svc.Recompute(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if result.Recomputed != 1 {
		t.Fatalf("recomputed = %d, want 1", result.Recomputed)
	}
	if len(st.Saved) != 1 || st.Saved[0].Score != 30 {
		t.Fatalf("saved scores = %+v, want one score of 30 (salary below floor)", st.Saved)
	}
}

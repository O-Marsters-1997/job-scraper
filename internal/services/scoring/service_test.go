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

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

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

var (
	retiredAt    = time.Now().Add(-time.Hour)
	retiredCobol = dto.ScoringOption{
		ID: "tech:cobol", Dimension: dto.DimensionTech, Label: "COBOL", Question: "Does the role use COBOL?", RetiredAt: &retiredAt,
	}
	testJob = dto.Job{ID: "job-1", Title: "Backend Engineer", ContentFingerprint: "fp-1", Source: "greenhouse"}
)

func newFakeStore() *scoringtest.FakeStore {
	st := scoringtest.NewFakeStore()
	st.SeedOptions(bank)
	return st
}

func seedEffect(st *scoringtest.FakeStore, firstDiscovery bool, configs ...dto.SearchConfig) {
	st.SeedJob(testJob, configs)
	st.SeedEffect(dto.AnswerEffect{
		ID: "effect-1", JobID: testJob.ID, Fingerprint: testJob.ContentFingerprint, Attempts: 1, FirstDiscovery: firstDiscovery,
	})
}

func picking(userID string, optionIDs ...string) dto.SearchConfig {
	var picks []dto.Pick
	for _, id := range optionIDs {
		picks = append(picks, dto.Pick{OptionID: id, Stance: "nice", Source: "manual"})
	}
	return dto.SearchConfig{UserID: userID, NotifyThreshold: 70, Preferences: dto.Preferences{Picks: picks}}
}

func cachedAnswers() map[string]dto.Answer {
	return map[string]dto.Answer{
		scoring.QuestionHash("Does the role use Go?"):             {PYes: 0.9, PNo: 0.05, PNotStated: 0.05},
		scoring.QuestionHash("Does the role use Rust?"):           {PYes: 0.1, PNo: 0.85, PNotStated: 0.05},
		scoring.QuestionHash("Is this primarily a backend role?"): {PYes: 0.9, PNo: 0.05, PNotStated: 0.05},
	}
}

func runTick(t *testing.T, st scoring.Store, opts ...depsOpt) {
	t.Helper()
	svc := newService(t, st, append([]depsOpt{withKey("sk-or-test")}, opts...)...)
	if err := svc.RunTick(t.Context()); err != nil {
		t.Fatalf("RunTick() err = %v", err)
	}
}

type fakeAnswerer struct {
	calls [][]string
}

func (f *fakeAnswerer) Answer(_ context.Context, _ string, _ dto.Job, questions []string) (map[string]dto.Answer, dto.Usage, error) {
	f.calls = append(f.calls, questions)
	out := make(map[string]dto.Answer, len(questions))
	for _, q := range questions {
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

type fakeExtractor struct {
	calls int
	picks []dto.Pick
	err   error
}

func (f *fakeExtractor) Extract(context.Context, string, string, []dto.ScoringOption, []dto.DimensionSpec) ([]dto.Pick, error) {
	f.calls++
	return f.picks, f.err
}

type depsOpt func(*scoring.Deps)

func withAnswerer(a scoring.Answerer) depsOpt      { return func(d *scoring.Deps) { d.Answerer = a } }
func withAlerter(a scoring.Alerter) depsOpt        { return func(d *scoring.Deps) { d.Alerter = a } }
func withProfiles(p scoring.ProfileReader) depsOpt { return func(d *scoring.Deps) { d.Profiles = p } }
func withCandidates(c scoring.Reconsiderer) depsOpt {
	return func(d *scoring.Deps) { d.Candidates = c }
}
func withExtractor(e scoring.Extractor) depsOpt { return func(d *scoring.Deps) { d.Extractor = e } }
func withCredentials(c scoring.Credentials) depsOpt {
	return func(d *scoring.Deps) { d.Credentials = c }
}
func withTick(interval time.Duration) depsOpt {
	return func(d *scoring.Deps) { d.TickInterval = interval }
}
func withKey(key string) depsOpt { return withCredentials(&fakeCredentials{key: key}) }

func newDeps(t *testing.T, st scoring.Store, opts ...depsOpt) scoring.Deps {
	t.Helper()
	d := scoring.Deps{
		Store:       st,
		Answerer:    &fakeAnswerer{},
		Credentials: &fakeCredentials{},
		Alerter:     &fakeAlerter{},
		Profiles:    &fakeProfiles{},
		Candidates:  scoringtest.Reconsiders(),
		Extractor:   &fakeExtractor{},
	}
	for _, opt := range opts {
		opt(&d)
	}
	return d
}

func newService(t *testing.T, st scoring.Store, opts ...depsOpt) *scoring.Service {
	t.Helper()
	return scoring.NewService(newDeps(t, st, opts...))
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
		st.SeedAnswers(testJob.ID, testJob.ContentFingerprint, jev.Model, cachedAnswers())
		seedEffect(st, true, picking("user-1", "tech:go"))
		answerer := &fakeAnswerer{}

		runTick(t, st, withAnswerer(answerer))

		if len(answerer.calls) != 0 {
			t.Errorf("Answer calls = %v, want none (everything cached)", answerer.calls)
		}
		completed := st.Completed()
		if len(completed) != 1 || len(completed[0].Answers) != 0 {
			t.Errorf("completed effects = %+v, want 1 effect with no new answers", completed)
		}
	})

	questionCases := []struct {
		name    string
		options []dto.ScoringOption
		configs []dto.SearchConfig
		want    []string
	}{
		{
			name:    "missing questions are sent exactly",
			configs: []dto.SearchConfig{picking("user-1", "tech:go", "tech:rust", "role:backend")},
			want:    []string{"Does the role use Go?", "Does the role use Rust?", "Is this primarily a backend role?"},
		},
		{
			name:    "only picked questions are sent",
			configs: []dto.SearchConfig{picking("user-1", "tech:go")},
			want:    []string{"Does the role use Go?"},
		},
		{
			name:    "overlapping picks are unioned without duplicates",
			configs: []dto.SearchConfig{picking("user-1", "tech:go", "tech:rust"), picking("user-2", "tech:rust", "role:backend")},
			want:    []string{"Does the role use Go?", "Does the role use Rust?", "Is this primarily a backend role?"},
		},
		{
			name:    "a retired option pick is never sent",
			options: []dto.ScoringOption{retiredCobol},
			configs: []dto.SearchConfig{picking("user-1", "tech:go", "tech:cobol")},
			want:    []string{"Does the role use Go?"},
		},
	}
	for _, tt := range questionCases {
		t.Run(tt.name, func(t *testing.T) {
			st := newFakeStore()
			st.SeedOptions(append(slices.Clone(bank), tt.options...))
			seedEffect(st, false, tt.configs...)
			answerer := &fakeAnswerer{}

			runTick(t, st, withAnswerer(answerer))

			sortStrings := cmpopts.SortSlices(func(a, b string) bool { return a < b })
			if diff := cmp.Diff([][]string{tt.want}, answerer.calls, sortStrings); diff != "" {
				t.Errorf("Answer calls (-want +got):\n%s", diff)
			}
			completed := st.Completed()
			if len(completed) != 1 || len(completed[0].Answers) != len(tt.want) {
				t.Errorf("completed effects = %+v, want 1 effect with %d new answers", completed, len(tt.want))
			}
		})
	}

	t.Run("a successful answer emits a score call log", func(t *testing.T) {
		buf := captureScoreCallLogs(t)
		st := newFakeStore()
		seedEffect(st, false, picking("user-1", "tech:go"))

		runTick(t, st)

		lines := scoreCallLines(t, buf)
		if len(lines) != 1 {
			t.Fatalf("score.call lines = %d, want 1: %v", len(lines), lines)
		}
		if lines[0]["user_id"] != "user-1" || lines[0]["model"] != "typesafe/jev-1.13-test" || lines[0]["cost_usd"] != 0.0004 {
			t.Errorf("score.call line = %+v, want user_id=user-1 model=typesafe/jev-1.13-test cost_usd=0.0004", lines[0])
		}
	})

	t.Run("a failed answer emits no score call log", func(t *testing.T) {
		buf := captureScoreCallLogs(t)
		st := newFakeStore()
		seedEffect(st, false, picking("user-1", "tech:go"))

		runTick(t, st, withAnswerer(&failingAnswerer{}))

		if lines := scoreCallLines(t, buf); len(lines) != 0 {
			t.Errorf("score.call lines = %v, want none after a failed Answer call", lines)
		}
		if failed := st.Failed(); len(failed) != 1 {
			t.Errorf("failed effects = %d, want 1", len(failed))
		}
	})

	t.Run("no picks skips the jev call and still writes a score", func(t *testing.T) {
		st := newFakeStore()
		seedEffect(st, false, picking("user-1"))
		answerer := &fakeAnswerer{}

		runTick(t, st, withAnswerer(answerer))

		if len(answerer.calls) != 0 {
			t.Errorf("Answer calls = %v, want none", answerer.calls)
		}
		completed := st.Completed()
		if len(completed) != 1 || len(completed[0].Scores) != 1 {
			t.Errorf("completed effects = %+v, want 1 effect with a score written from the prior", completed)
		}
	})

	t.Run("a job failing every include filter is never sent to jev", func(t *testing.T) {
		st := newFakeStore()
		cfg := picking("user-1", "tech:go")
		cfg.RequiredTitleKeywords = []string{"designer"}
		seedEffect(st, false, cfg)
		answerer := &fakeAnswerer{}

		runTick(t, st, withAnswerer(answerer))

		if len(answerer.calls) != 0 {
			t.Errorf("Answer calls = %v, want none (job rejected by include filter)", answerer.calls)
		}
		if completed := st.Completed(); len(completed) != 1 {
			t.Errorf("completed effects = %d, want 1", len(completed))
		}
	})

	t.Run("alerts only on first discovery above threshold", func(t *testing.T) {
		st := newFakeStore()
		st.SeedAnswers(testJob.ID, testJob.ContentFingerprint, jev.Model, cachedAnswers())
		cfg := picking("user-1", "tech:go")
		cfg.NotifyThreshold = 50
		seedEffect(st, true, cfg)
		alerter := &fakeAlerter{}

		runTick(t, st, withAlerter(alerter), withProfiles(&fakeProfiles{emails: map[string]string{"user-1": "user@example.com"}}))

		if diff := cmp.Diff([]string{"user@example.com"}, alerter.notified); diff != "" {
			t.Errorf("notified (-want +got):\n%s", diff)
		}
	})

	t.Run("does not alert a user whose company is new", func(t *testing.T) {
		st := newFakeStore()
		st.SeedAnswers(testJob.ID, testJob.ContentFingerprint, jev.Model, cachedAnswers())
		cfg := picking("user-1", "tech:go")
		cfg.NotifyThreshold = 50
		cfg.CompanyIsNew = true
		seedEffect(st, true, cfg)
		alerter := &fakeAlerter{}

		runTick(t, st, withAlerter(alerter), withProfiles(&fakeProfiles{emails: map[string]string{"user-1": "user@example.com"}}))

		if len(alerter.notified) != 0 {
			t.Errorf("notified = %v, want none", alerter.notified)
		}
		if completed := st.Completed(); len(completed) != 1 || len(completed[0].Scores) != 1 {
			t.Errorf("completed = %+v, want the job still scored once", completed)
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
		svc := newService(t, newFakeStore())

		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan error, 1)
		go func() { done <- svc.Run(ctx) }()
		cancel()

		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Run() err = %v, want nil", err)
			}
		case <-time.After(time.Second):
			t.Error("Run did not return after context cancellation")
		}
	})

	t.Run("keeps ticking after a failed tick", func(t *testing.T) {
		inner := newFakeStore()
		inner.SeedAnswers(testJob.ID, testJob.ContentFingerprint, jev.Model, cachedAnswers())
		seedEffect(inner, false, picking("user-1", "tech:go"))
		st := &flakyClaimStore{FakeStore: inner, failsLeft: 1, completed: make(chan struct{}, 1)}
		svc := newService(t, st, withKey("sk-or-test"), withTick(time.Millisecond))

		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan error, 1)
		go func() { done <- svc.Run(ctx) }()

		select {
		case <-st.completed:
		case <-time.After(time.Second):
			t.Error("effect never completed after the failed tick")
		}

		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Run() err = %v, want nil", err)
			}
		case <-time.After(time.Second):
			t.Error("Run did not return after context cancellation")
		}
	})
}

func TestOptions(t *testing.T) {
	st := scoringtest.NewFakeStore()
	st.SeedOptions([]dto.ScoringOption{bank[0], retiredCobol})

	got, err := newService(t, st).Options(t.Context(), "user-1")
	if err != nil {
		t.Fatalf("Options() err = %v", err)
	}

	if len(got.Dimensions) != len(scoring.Dimensions) {
		t.Errorf("dimensions = %d, want %d", len(got.Dimensions), len(scoring.Dimensions))
	}
	want := []dto.ScoringOption{{ID: "tech:go", Dimension: dto.DimensionTech, Label: "Go"}}
	if diff := cmp.Diff(want, got.Options); diff != "" {
		t.Errorf("options, retired excluded and question text stripped (-want +got):\n%s", diff)
	}
}

func TestUpdateConfigTextExtraction(t *testing.T) {
	ctx := t.Context()
	goPick := dto.Pick{OptionID: "tech:go", Stance: "nice", Source: "text"}
	newExtractingService := func(t *testing.T, picks ...dto.Pick) (*scoring.Service, *fakeExtractor) {
		t.Helper()
		extractor := &fakeExtractor{picks: picks}
		return newService(t, newFakeStore(), withKey("sk-test"), withExtractor(extractor)), extractor
	}

	t.Run("unchanged text skips extraction", func(t *testing.T) {
		svc, extractor := newExtractingService(t, goPick)
		in := dto.ScoringConfigView{Preferences: dto.Preferences{PreferenceText: "I want to work with Go"}}

		for range 2 {
			if _, err := svc.UpdateConfig(ctx, "user-1", in); err != nil {
				t.Fatalf("UpdateConfig() err = %v", err)
			}
		}
		if extractor.calls != 1 {
			t.Errorf("extractor calls = %d, want 1 (the second save has unchanged text)", extractor.calls)
		}
	})

	t.Run("re-extraction replaces text picks and keeps manual ones", func(t *testing.T) {
		svc, extractor := newExtractingService(t, goPick)
		manual := dto.Pick{OptionID: "seniority:senior", Stance: "nice"}
		if _, err := svc.UpdateConfig(ctx, "user-1", dto.ScoringConfigView{
			Preferences: dto.Preferences{Picks: []dto.Pick{manual}, PreferenceText: "I know Go"},
		}); err != nil {
			t.Fatalf("UpdateConfig() err = %v", err)
		}

		extractor.picks = []dto.Pick{{OptionID: "domain:gambling", Stance: "avoid", Source: "text"}}
		got, err := svc.UpdateConfig(ctx, "user-1", dto.ScoringConfigView{
			Preferences: dto.Preferences{Picks: []dto.Pick{manual}, PreferenceText: "avoid gambling companies"},
		})
		if err != nil {
			t.Fatalf("UpdateConfig() err = %v", err)
		}

		want := []dto.Pick{
			{OptionID: "seniority:senior", Stance: "nice", Source: "manual"},
			{OptionID: "domain:gambling", Stance: "avoid", Source: "text"},
		}
		if diff := cmp.Diff(want, got.Preferences.Picks); diff != "" {
			t.Errorf("picks (-want +got):\n%s", diff)
		}
	})

	t.Run("a manual pick overrides a text pick on the same option", func(t *testing.T) {
		svc, _ := newExtractingService(t, dto.Pick{OptionID: "tech:kubernetes", Stance: "avoid", Source: "text"})

		got, err := svc.UpdateConfig(ctx, "user-1", dto.ScoringConfigView{
			Preferences: dto.Preferences{
				Picks:          []dto.Pick{{OptionID: "tech:kubernetes", Stance: "nice"}},
				PreferenceText: "avoid Kubernetes",
			},
		})
		if err != nil {
			t.Fatalf("UpdateConfig() err = %v", err)
		}

		want := []dto.Pick{
			{OptionID: "tech:kubernetes", Stance: "nice", Source: "manual"},
			{OptionID: "tech:kubernetes", Stance: "avoid", Source: "text", Overridden: true},
		}
		if diff := cmp.Diff(want, got.Preferences.Picks); diff != "" {
			t.Errorf("picks (-want +got):\n%s", diff)
		}
	})

	t.Run("a hallucinated option id is dropped, not rejected", func(t *testing.T) {
		svc, _ := newExtractingService(t, goPick, dto.Pick{OptionID: "tech:made-up", Stance: "nice", Source: "text"})

		got, err := svc.UpdateConfig(ctx, "user-1", dto.ScoringConfigView{
			Preferences: dto.Preferences{PreferenceText: "I like Go, and made-up-thing"},
		})
		if err != nil {
			t.Fatalf("UpdateConfig() err = %v", err)
		}
		if diff := cmp.Diff([]dto.Pick{goPick}, got.Preferences.Picks); diff != "" {
			t.Errorf("picks (-want +got):\n%s", diff)
		}
	})

	t.Run("no credential is unprocessable", func(t *testing.T) {
		svc := newService(t, newFakeStore())

		_, err := svc.UpdateConfig(ctx, "user-1", dto.ScoringConfigView{
			Preferences: dto.Preferences{PreferenceText: "I like Go"},
		})
		if !apperr.IsKind(err, apperr.KindUnprocessable) {
			t.Errorf("UpdateConfig() err = %v, want an unprocessable apperr", err)
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
		svc := newService(t, st, withAlerter(alerter))

		result, err := svc.Recompute(t.Context(), "user-1")
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

		svc := newService(t, st)

		if _, err := svc.Recompute(t.Context(), "user-1"); err != nil {
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

				svc := newService(t, st)
				result, err := svc.Recompute(t.Context(), "user-1")
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
				if tc.wantRows != nil {
					ignore := cmpopts.IgnoreFields(dto.ScoreRow{}, "Key", "Label", "Stance", "Overridden")
					if diff := cmp.Diff(tc.wantRows, saved[0].Rows, ignore); diff != "" {
						t.Errorf("rows (-want +got):\n%s", diff)
					}
				}
			})
		}
	})
}

func TestFillMissingAnswers(t *testing.T) {
	tests := []struct {
		name       string
		options    []dto.ScoringOption
		cfg        *dto.SearchConfig
		wantQueued int64
	}{
		{"queues for the currently picked options", nil, ptr(picking("user-1", "tech:go", "tech:rust")), 2},
		{"no picks queues nothing", nil, ptr(picking("user-1")), 0},
		{"no search config queues nothing", nil, nil, 0},
		{"a retired pick is never asked", []dto.ScoringOption{retiredCobol}, ptr(picking("user-1", "tech:cobol")), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := newFakeStore()
			st.SeedOptions(append(slices.Clone(bank), tt.options...))
			if tt.cfg != nil {
				st.SeedSearchConfig(*tt.cfg)
			}

			queued, err := newService(t, st).FillMissingAnswers(t.Context(), "user-1")
			if err != nil {
				t.Fatalf("FillMissingAnswers() err = %v", err)
			}
			if queued != tt.wantQueued {
				t.Errorf("FillMissingAnswers() = %d, want %d", queued, tt.wantQueued)
			}
		})
	}
}

func ptr[T any](v T) *T { return &v }

func TestAsk(t *testing.T) {
	ctx := t.Context()
	job := dto.Job{ID: "job-1", ContentFingerprint: "fp-1"}
	newSvc := func(t *testing.T, key string) (*scoring.Service, *fakeAnswerer) {
		t.Helper()
		st := newFakeStore()
		st.SeedJob(job, nil)
		answerer := &fakeAnswerer{}
		return newService(t, st, withAnswerer(answerer), withKey(key)), answerer
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

func TestAppendOverallFeedback(t *testing.T) {
	const userID = "user-1"

	t.Run("stores the trimmed reason with the current Picks and model", func(t *testing.T) {
		st := newFakeStore()
		st.SeedSearchConfig(picking(userID, "tech:go"))
		svc := scoring.NewService(newDeps(t, st))

		got, err := svc.AppendOverallFeedback(t.Context(), userID, dto.OverallFeedbackInput{Reason: "  too generous  "})
		if err != nil {
			t.Fatalf("AppendOverallFeedback() err = %v", err)
		}

		want := dto.ScoreFeedback{
			Kind: "overall", Reason: "too generous", Model: jev.Model,
			Picks: []dto.Pick{{OptionID: "tech:go", Stance: "nice", Source: "manual"}},
		}
		if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(dto.ScoreFeedback{}, "ID", "CreatedAt")); diff != "" {
			t.Errorf("AppendOverallFeedback() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("a user with no Search Config logs empty Picks", func(t *testing.T) {
		svc := scoring.NewService(newDeps(t, newFakeStore()))

		got, err := svc.AppendOverallFeedback(t.Context(), userID, dto.OverallFeedbackInput{Reason: "hm"})
		if err != nil {
			t.Fatalf("AppendOverallFeedback() err = %v", err)
		}
		if got.Picks == nil || len(got.Picks) != 0 {
			t.Errorf("AppendOverallFeedback() picks = %#v, want an empty non-nil slice", got.Picks)
		}
	})

	t.Run("blank reason is invalid and writes nothing", func(t *testing.T) {
		st := newFakeStore()
		svc := scoring.NewService(newDeps(t, st))

		_, err := svc.AppendOverallFeedback(t.Context(), userID, dto.OverallFeedbackInput{Reason: " \n"})
		if !apperr.IsKind(err, apperr.KindInvalid) {
			t.Fatalf("AppendOverallFeedback(blank) err = %v, want Invalid", err)
		}
		if got, _ := svc.ListFeedback(t.Context(), userID, dto.ScoreFeedbackQuery{}); len(got.Entries) != 0 {
			t.Errorf("ListFeedback() = %+v, want nothing written", got)
		}
	})
}

func seedScoredJob(st *scoringtest.FakeStore, userID string, optionIDs ...string) {
	st.SeedOptions(append([]dto.ScoringOption{retiredCobol}, bank...))
	st.SeedSearchConfig(picking(userID, optionIDs...))
	job := testJob
	job.CompanySlug = "acme"
	job.Description = "<p>Build Go services.</p>"
	st.SeedJob(job, nil)
	st.SeedAnswers(job.ID, job.ContentFingerprint, jev.Model, cachedAnswers())
	st.SeedJobScore(userID, job.ID, dto.JobScoreEvidence{
		Score: 72, Fingerprint: "score-fp", Model: "jev-old",
		Breakdown: []dto.ScoreRow{{Key: "tech:go", Label: "Go", Stance: "nice", Resolved: "yes", Effect: "meets"}},
	})
}

func TestAppendJobFeedback(t *testing.T) {
	const userID = "user-1"
	svc := func(st *scoringtest.FakeStore) *scoring.Service { return scoring.NewService(newDeps(t, st)) }

	t.Run("freezes the score, per-Option probabilities and Jev state", func(t *testing.T) {
		st := newFakeStore()
		seedScoredJob(st, userID, "tech:go", "tech:cobol", "tech:kubernetes")

		got, err := svc(st).AppendJobFeedback(t.Context(), userID, dto.JobFeedbackInput{JobID: testJob.ID, Direction: "lower", Reason: " too high "})
		if err != nil {
			t.Fatalf("AppendJobFeedback() err = %v", err)
		}

		score, direction, jobID := 72, "lower", testJob.ID
		want := dto.ScoreFeedback{
			Kind: "job", Direction: &direction, JobID: &jobID, Reason: "too high", Model: jev.Model,
			Picks: picking(userID, "tech:go", "tech:cobol", "tech:kubernetes").Preferences.Picks,
			Snapshot: dto.ScoreFeedbackSnapshot{
				Score:              &score,
				Breakdown:          []dto.ScoreRow{{Key: "tech:go", Label: "Go", Stance: "nice", Resolved: "yes", Effect: "meets"}},
				ScoreFingerprint:   "score-fp",
				ScoreModel:         "jev-old",
				ContentFingerprint: "fp-1",
				Options: []dto.FeedbackOption{
					{OptionID: "tech:go", Label: "Go", Question: "Does the role use Go?", Stance: "nice", Resolved: "yes", PYes: 0.9, PNo: 0.05, PNotStated: 0.05, Known: true},
					{OptionID: "tech:cobol", Label: "COBOL", Question: "Does the role use COBOL?", Stance: "nice", Resolved: "retired"},
					{OptionID: "tech:kubernetes", Label: "Kubernetes", Question: "Does the role use Kubernetes?", Stance: "nice", Resolved: "unknown"},
				},
				JevState: &dto.JevState{Title: "Backend Engineer", Company: "acme", Description: "Build Go services."},
			},
		}
		if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(dto.ScoreFeedback{}, "ID", "CreatedAt")); diff != "" {
			t.Errorf("AppendJobFeedback() mismatch (-want +got):\n%s", diff)
		}
	})

	tests := []struct {
		name string
		in   dto.JobFeedbackInput
		seed func(*scoringtest.FakeStore)
		kind apperr.Kind
	}{
		{name: "unknown job is not found", in: dto.JobFeedbackInput{JobID: "nope", Direction: "higher", Reason: "r"}, kind: apperr.KindNotFound},
		{
			name: "unscored job is unprocessable", in: dto.JobFeedbackInput{JobID: "job-2", Direction: "higher", Reason: "r"},
			seed: func(st *scoringtest.FakeStore) { st.SeedJob(dto.Job{ID: "job-2"}, nil) }, kind: apperr.KindUnprocessable,
		},
		{name: "bad direction is invalid", in: dto.JobFeedbackInput{JobID: testJob.ID, Direction: "sideways", Reason: "r"}, kind: apperr.KindInvalid},
		{name: "blank reason is invalid", in: dto.JobFeedbackInput{JobID: testJob.ID, Direction: "higher", Reason: " "}, kind: apperr.KindInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name+" and writes nothing", func(t *testing.T) {
			st := newFakeStore()
			seedScoredJob(st, userID, "tech:go")
			if tt.seed != nil {
				tt.seed(st)
			}

			_, err := svc(st).AppendJobFeedback(t.Context(), userID, tt.in)
			if !apperr.IsKind(err, tt.kind) {
				t.Fatalf("AppendJobFeedback(%+v) err = %v, want kind %v", tt.in, err, tt.kind)
			}
			if got, _ := svc(st).ListFeedback(t.Context(), userID, dto.ScoreFeedbackQuery{}); len(got.Entries) != 0 {
				t.Errorf("ListFeedback() = %+v, want nothing written", got)
			}
		})
	}
}

func TestListFeedback_Outdated(t *testing.T) {
	const userID = "user-1"
	st := newFakeStore()
	st.SeedSearchConfig(picking(userID, "tech:go"))
	svc := scoring.NewService(newDeps(t, st))
	for _, reason := range []string{"before", "also before"} {
		if _, err := svc.AppendOverallFeedback(t.Context(), userID, dto.OverallFeedbackInput{Reason: reason}); err != nil {
			t.Fatalf("AppendOverallFeedback(%q) err = %v", reason, err)
		}
	}
	st.SeedSearchConfig(picking(userID, "tech:cobol"))
	if _, err := svc.AppendOverallFeedback(t.Context(), userID, dto.OverallFeedbackInput{Reason: "after"}); err != nil {
		t.Fatalf("AppendOverallFeedback() err = %v", err)
	}

	hidden, err := svc.ListFeedback(t.Context(), userID, dto.ScoreFeedbackQuery{})
	if err != nil {
		t.Fatalf("ListFeedback() err = %v", err)
	}
	if hidden.Total != 1 || hidden.CurrentCount != 1 || hidden.OutdatedCount != 2 || len(hidden.Entries) != 1 {
		t.Errorf("ListFeedback() = %+v, want 1 current entry and 2 outdated counted", hidden)
	}

	shown, err := svc.ListFeedback(t.Context(), userID, dto.ScoreFeedbackQuery{Outdated: "true"})
	if err != nil {
		t.Fatalf("ListFeedback(outdated) err = %v", err)
	}
	if shown.Total != 3 || len(shown.Entries) != 3 || !shown.Entries[1].PicksChanged || shown.Entries[0].PicksChanged {
		t.Errorf("ListFeedback(outdated) = %+v, want all 3 entries, the two earlier ones tagged picks changed", shown)
	}
}

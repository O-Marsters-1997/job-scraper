package scoring_test

import (
	"context"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/scoringtest"
)

func buildModule(t *testing.T, st *scoringtest.FakeStore) *scoring.Module {
	t.Helper()
	return scoring.Build(scoring.Deps{
		Store:       st,
		Answerer:    &fakeAnswerer{t: t, forbidden: true},
		Credentials: &fakeCredentials{},
		Alerter:     &fakeAlerter{},
		Profiles:    &fakeProfiles{},
		Candidates:  scoringtest.Reconsiders(),
		Extractor:   scoringtest.Extracts(nil),
	})
}

func TestSearchConfig(t *testing.T) {
	st := newFakeStore()
	st.SeedSearchConfig(dto.SearchConfig{UserID: "user-1", NotifyThreshold: 70})
	m := buildModule(t, st)

	got, err := m.SearchConfig(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.UserID != "user-1" || got.NotifyThreshold != 70 {
		t.Fatalf("SearchConfig(...) = %+v, want userID user-1 with threshold 70", got)
	}
}

func TestOpsState(t *testing.T) {
	st := newFakeStore()
	st.SeedEffect(dto.AnswerEffect{ID: "effect-1", JobID: "job-1"})
	m := buildModule(t, st)

	got, err := m.OpsState(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.OutboxPending != 1 {
		t.Fatalf("OpsState(...) = %+v, want one pending effect", got)
	}
}

func TestJobsChanged(t *testing.T) {
	st := newFakeStore()
	st.SeedJob(dto.Job{ID: "job-1", ContentFingerprint: "fp-1"}, nil)
	m := buildModule(t, st)

	if err := m.JobsChanged(context.Background(), nil, []string{"job-1"}, true); err != nil {
		t.Fatal(err)
	}
	state, err := m.OpsState(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if state.OutboxPending != 1 {
		t.Fatalf("OpsState(...) = %+v, want one effect queued by JobsChanged", state)
	}
}

func TestJobsClosed(t *testing.T) {
	st := newFakeStore()
	st.SeedAnswers("job-1", "fp-1", "typesafe/jev-1.13", map[string]dto.Answer{"hash-1": {PYes: 0.9}})
	m := buildModule(t, st)

	if err := m.JobsClosed(context.Background(), nil, []string{"job-1"}); err != nil {
		t.Fatal(err)
	}
	answers, err := st.ListAnswers(context.Background(), "job-1", "fp-1", "typesafe/jev-1.13")
	if err != nil {
		t.Fatal(err)
	}
	if len(answers) != 0 {
		t.Fatalf("answers after JobsClosed = %+v, want none", answers)
	}
}

func TestCompanyTracked(t *testing.T) {
	st := newFakeStore()
	st.SeedJob(dto.Job{ID: "job-1", CompanyID: "company-1", ContentFingerprint: "fp-1"}, nil)
	m := buildModule(t, st)

	if err := m.CompanyTracked(context.Background(), nil, "user-1", "company-1"); err != nil {
		t.Fatal(err)
	}
	state, err := m.OpsState(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if state.OutboxPending != 1 {
		t.Fatalf("OpsState(...) = %+v, want one effect queued for the tracked company's job", state)
	}
}

func TestAddOption(t *testing.T) {
	st := newFakeStore()
	st.SeedOptions(nil)
	m := buildModule(t, st)

	if err := m.AddOption(context.Background(), "tech:zig", "tech", "Zig", "Does the role use Zig?"); err != nil {
		t.Fatal(err)
	}
	options, err := st.ListScoringOptions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(options) != 1 || options[0].Question != "Does the role use Zig?" {
		t.Fatalf("options = %+v, want one Zig option", options)
	}
}

func TestRewordOption(t *testing.T) {
	st := newFakeStore()
	m := buildModule(t, st)

	if err := m.RewordOption(context.Background(), "tech:go", "Does the role primarily use Go?"); err != nil {
		t.Fatal(err)
	}
	options, err := st.ListScoringOptions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if options[0].Question != "Does the role primarily use Go?" {
		t.Fatalf("question = %q, want reworded", options[0].Question)
	}
}

func TestRetireOption(t *testing.T) {
	st := newFakeStore()
	m := buildModule(t, st)

	if err := m.RetireOption(context.Background(), "tech:go"); err != nil {
		t.Fatal(err)
	}
	options, err := st.ListScoringOptions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if options[0].RetiredAt == nil {
		t.Fatalf("options = %+v, want retired_at set", options)
	}
}

package scoring_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
)

type recordingClassifier struct {
	kind  dto.CompanyKind
	err   error
	calls int
}

func (c *recordingClassifier) ClassifyRecruiter(context.Context, string, string, dto.Job) (dto.CompanyKind, dto.Usage, error) {
	c.calls++
	return c.kind, dto.Usage{}, c.err
}

type recordingKinds struct {
	suggested map[string]dto.CompanyKind
	err       error
}

func (k *recordingKinds) SuggestCompanyKind(_ context.Context, companyID string, kind dto.CompanyKind) (bool, error) {
	if k.err != nil {
		return false, k.err
	}
	if k.suggested == nil {
		k.suggested = map[string]dto.CompanyKind{}
	}
	k.suggested[companyID] = kind
	return true, nil
}

func withClassifier(c scoring.RecruiterClassifier, k scoring.CompanyKinds) depsOpt {
	return func(d *scoring.Deps) { d.Classifier, d.CompanyKinds = c, k }
}

func TestRunTickClassifiesCompany(t *testing.T) {
	seed := func(st interface {
		SeedJob(dto.Job, []dto.SearchConfig)
	}, kind dto.CompanyKind) {
		job := testJob
		job.CompanyID, job.CompanyKind = "company-1", kind
		st.SeedJob(job, []dto.SearchConfig{picking("user-1", "tech:go")})
	}
	run := func(t *testing.T, kind dto.CompanyKind, opts ...depsOpt) {
		t.Helper()
		st := newFakeStore()
		seed(st, kind)
		job := testJob
		st.SeedEffect(dto.AnswerEffect{ID: "effect-1", JobID: job.ID, Fingerprint: job.ContentFingerprint, Attempts: 1})
		runTick(t, st, opts...)
		if len(st.Completed()) != 1 {
			t.Errorf("completed effects = %d, want 1 (scoring must not fail)", len(st.Completed()))
		}
	}

	t.Run("an unclassified company is classified once and suggested", func(t *testing.T) {
		classifier, kinds := &recordingClassifier{kind: dto.KindLikelyAgency}, &recordingKinds{}
		run(t, dto.KindUnclassified, withClassifier(classifier, kinds))
		if classifier.calls != 1 || kinds.suggested["company-1"] != dto.KindLikelyAgency {
			t.Errorf("calls = %d, suggested = %v, want 1 call suggesting likely_agency", classifier.calls, kinds.suggested)
		}
	})

	t.Run("a classified company is never sent again", func(t *testing.T) {
		classifier, kinds := &recordingClassifier{kind: dto.KindLikelyAgency}, &recordingKinds{}
		run(t, dto.KindLikelyDirect, withClassifier(classifier, kinds))
		if classifier.calls != 0 {
			t.Errorf("classifier calls = %d, want 0", classifier.calls)
		}
	})

	t.Run("a classifier failure leaves the company unclassified", func(t *testing.T) {
		classifier, kinds := &recordingClassifier{err: errors.New("jev boom")}, &recordingKinds{}
		run(t, dto.KindUnclassified, withClassifier(classifier, kinds))
		if len(kinds.suggested) != 0 {
			t.Errorf("suggested = %v, want none", kinds.suggested)
		}
	})

	t.Run("no Jev key skips classification", func(t *testing.T) {
		classifier, kinds := &recordingClassifier{kind: dto.KindLikelyAgency}, &recordingKinds{}
		run(t, dto.KindUnclassified, withKey(""), withClassifier(classifier, kinds))
		if classifier.calls != 0 {
			t.Errorf("classifier calls = %d, want 0", classifier.calls)
		}
	})

	t.Run("a suggest failure does not fail scoring", func(t *testing.T) {
		classifier, kinds := &recordingClassifier{kind: dto.KindLikelyAgency}, &recordingKinds{err: errors.New("db down")}
		run(t, dto.KindUnclassified, withClassifier(classifier, kinds))
	})
}

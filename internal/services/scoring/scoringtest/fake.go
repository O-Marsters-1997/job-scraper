// Package scoringtest is the scoring context's test double: a map-backed
// fake of scoring.Store, proven against the real store by RunStoreContract
// (ADR 0012).
package scoringtest

import (
	"context"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/store"
)

// CompletedEffect is one recorded CompleteAnswerEffect call.
type CompletedEffect struct {
	Effect  dto.AnswerEffect
	Answers map[string]dto.Answer
	Scores  []dto.JobScore
}

// QueuedMissing is one recorded QueueMissingAnswers call.
type QueuedMissing struct {
	UserID string
	Hashes []string
	Model  string
}

type FakeStore struct {
	mu sync.Mutex

	effects []dto.AnswerEffect
	jobs    map[string]dto.Job
	configs map[string][]dto.SearchConfig
	answers map[string]map[string]dto.Answer
	inputs  map[string][]store.ScoringInput
	options []dto.ScoringOption
	search  map[string]dto.SearchConfig
	scored  map[string]bool
	pushes  map[string][]dto.PushSubscriptionInput

	feedback    map[string][]dto.ScoreFeedback
	feedbackSeq int
	evidence    map[string]dto.JobScoreEvidence

	failed        []dto.ScoringFailure
	completed     []CompletedEffect
	recomputed    []dto.JobScore
	queuedMissing []QueuedMissing
}

func NewFakeStore() *FakeStore {
	return &FakeStore{
		jobs:    make(map[string]dto.Job),
		configs: make(map[string][]dto.SearchConfig),
		answers: make(map[string]map[string]dto.Answer),
		inputs:  make(map[string][]store.ScoringInput),
		search:  make(map[string]dto.SearchConfig),
		pushes:  make(map[string][]dto.PushSubscriptionInput),
		scored:  make(map[string]bool),

		feedback: make(map[string][]dto.ScoreFeedback),
		evidence: make(map[string]dto.JobScoreEvidence),
	}
}

func answerKey(jobID, fingerprint, model string) string {
	return jobID + "|" + fingerprint + "|" + model
}

func scoredKey(jobID, userID string) string {
	return jobID + "|" + userID
}

func (f *FakeStore) SeedEffect(effect dto.AnswerEffect) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.effects = append(f.effects, effect)
}

func (f *FakeStore) SeedJob(job dto.Job, configs []dto.SearchConfig) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.jobs[job.ID] = job
	f.configs[job.ID] = configs
}

func (f *FakeStore) SeedAnswers(jobID, fingerprint, model string, answers map[string]dto.Answer) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.answers[answerKey(jobID, fingerprint, model)] = answers
}

func (f *FakeStore) SeedScoringInputs(userID string, inputs []store.ScoringInput) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.inputs[userID] = inputs
}

func (f *FakeStore) SeedOptions(options []dto.ScoringOption) {
	f.mu.Lock()
	defer f.mu.Unlock()
	options = append([]dto.ScoringOption{}, options...)
	f.options = options
}

func (f *FakeStore) SeedSearchConfig(cfg dto.SearchConfig) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.search[cfg.UserID] = cfg
}

func (f *FakeStore) Failed() []dto.ScoringFailure {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]dto.ScoringFailure{}, f.failed...)
}

func (f *FakeStore) Completed() []CompletedEffect {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]CompletedEffect{}, f.completed...)
}

func (f *FakeStore) Recomputed() []dto.JobScore {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]dto.JobScore{}, f.recomputed...)
}

func (f *FakeStore) QueuedMissing() []QueuedMissing {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]QueuedMissing{}, f.queuedMissing...)
}

func (f *FakeStore) ClaimAnswerEffect(context.Context) (dto.AnswerEffect, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.effects) == 0 {
		return dto.AnswerEffect{}, data.ErrNotFound
	}
	e := f.effects[0]
	f.effects = f.effects[1:]
	return e, nil
}

func (f *FakeStore) FailAnswerEffect(_ context.Context, _ string, _ int, failure dto.ScoringFailure) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failed = append(f.failed, failure)
	return nil
}

func (f *FakeStore) GetJobForScoring(_ context.Context, jobID string) (dto.Job, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	job, ok := f.jobs[jobID]
	if !ok {
		return dto.Job{}, data.ErrNotFound
	}
	return job, nil
}

func (f *FakeStore) ListInterestedConfigs(_ context.Context, jobID string) ([]dto.SearchConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.configs[jobID], nil
}

func (f *FakeStore) ListScoringOptions(context.Context) ([]dto.ScoringOption, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]dto.ScoringOption, len(f.options))
	copy(out, f.options)
	return out, nil
}

func (f *FakeStore) ListAnswers(_ context.Context, jobID, fingerprint, model string) (map[string]dto.Answer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := answerKey(jobID, fingerprint, model)
	out := make(map[string]dto.Answer, len(f.answers[key]))
	maps.Copy(out, f.answers[key])
	return out, nil
}

func (f *FakeStore) SaveAnswers(_ context.Context, jobID, fingerprint, model string, answers map[string]dto.Answer) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := answerKey(jobID, fingerprint, model)
	if f.answers[key] == nil {
		f.answers[key] = make(map[string]dto.Answer, len(answers))
	}
	for hash, a := range answers {
		if _, exists := f.answers[key][hash]; !exists {
			f.answers[key][hash] = a
		}
	}
	return nil
}

func (f *FakeStore) markScored(jobID string, scores []dto.JobScore) {
	for _, sc := range scores {
		f.scored[scoredKey(jobID, sc.UserID)] = true
	}
}

func (f *FakeStore) CompleteAnswerEffect(_ context.Context, effect dto.AnswerEffect, answers map[string]dto.Answer, scores []dto.JobScore) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.completed = append(f.completed, CompletedEffect{Effect: effect, Answers: answers, Scores: scores})
	f.markScored(effect.JobID, scores)
	saved := make([]string, len(scores))
	for i, s := range scores {
		saved[i] = s.UserID
	}
	return saved, nil
}

func (f *FakeStore) GetSearchConfig(_ context.Context, userID string) (dto.SearchConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	cfg, ok := f.search[userID]
	if !ok {
		return dto.SearchConfig{}, data.ErrNotFound
	}
	return cfg, nil
}

func (f *FakeStore) ListIncludeFilterConfigs(_ context.Context) ([]dto.SearchConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []dto.SearchConfig
	for _, cfg := range f.search {
		if len(cfg.RequiredLocations) > 0 || len(cfg.RequiredTitleKeywords) > 0 {
			out = append(out, cfg)
		}
	}
	return out, nil
}

func (f *FakeStore) UpsertSearchConfig(_ context.Context, cfg dto.SearchConfig) (dto.SearchConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.search[cfg.UserID] = cfg
	return cfg, nil
}

func (f *FakeStore) ListScoringInputs(_ context.Context, userID, _ string) ([]store.ScoringInput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.inputs[userID], nil
}

func (f *FakeStore) SaveScores(_ context.Context, scores []dto.JobScore) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.recomputed = append(f.recomputed, scores...)
	for _, sc := range scores {
		f.markScored(sc.JobID, []dto.JobScore{sc})
	}
	return nil
}

// QueueMissingAnswers queues every hash unconditionally: the fake models no
// scored-job or cached-answer state to compute a real gap against.
func (f *FakeStore) QueueMissingAnswers(_ context.Context, userID string, hashes []string, model string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.queuedMissing = append(f.queuedMissing, QueuedMissing{UserID: userID, Hashes: hashes, Model: model})
	return int64(len(hashes)), nil
}

func (f *FakeStore) OpsState(context.Context) (dto.OpsState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return dto.OpsState{OutboxPending: int64(len(f.effects))}, nil
}

func (f *FakeStore) GetScoringStatus(_ context.Context, userID string) (dto.ScoringStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var pending int64
	for _, e := range f.effects {
		if f.scored[scoredKey(e.JobID, userID)] {
			pending++
		}
	}
	return dto.ScoringStatus{Pending: pending}, nil
}

func (f *FakeStore) JobsChanged(_ context.Context, _ pgx.Tx, jobIDs []string, firstDiscovery bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, jobID := range jobIDs {
		for key := range f.answers {
			if strings.HasPrefix(key, jobID+"|") {
				delete(f.answers, key)
			}
		}
		job := f.jobs[jobID]
		f.effects = append(f.effects, dto.AnswerEffect{JobID: jobID, Fingerprint: job.ContentFingerprint, FirstDiscovery: firstDiscovery})
	}
	return nil
}

func (f *FakeStore) JobsClosed(_ context.Context, _ pgx.Tx, jobIDs []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, jobID := range jobIDs {
		for key := range f.answers {
			if strings.HasPrefix(key, jobID+"|") {
				delete(f.answers, key)
			}
		}
	}
	return nil
}

func (f *FakeStore) CompanyTracked(_ context.Context, _ pgx.Tx, _, companyID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for jobID, job := range f.jobs {
		if job.CompanyID == companyID {
			f.effects = append(f.effects, dto.AnswerEffect{JobID: jobID, Fingerprint: job.ContentFingerprint})
		}
	}
	return nil
}

func (f *FakeStore) AddScoringOption(_ context.Context, id, dimension, label, question string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.options = append(f.options, dto.ScoringOption{ID: id, Dimension: dto.Dimension(dimension), Label: label, Question: question})
	return nil
}

func (f *FakeStore) RewordScoringOption(_ context.Context, id, question string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, o := range f.options {
		if o.ID == id {
			f.options[i].Question = question
			return nil
		}
	}
	return data.ErrNotFound
}

func (f *FakeStore) RetireScoringOption(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, o := range f.options {
		if o.ID != id {
			continue
		}
		if o.RetiredAt != nil {
			return data.ErrNotFound
		}
		now := time.Now()
		f.options[i].RetiredAt = &now
		return nil
	}
	return data.ErrNotFound
}

func (f *FakeStore) ListCompanyAnswers(_ context.Context, companyIDs []string, model string) (map[string][]map[string]dto.Answer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[string][]map[string]dto.Answer)
	for _, job := range f.jobs {
		if !slices.Contains(companyIDs, job.CompanyID) {
			continue
		}
		answers := make(map[string]dto.Answer)
		maps.Copy(answers, f.answers[answerKey(job.ID, job.ContentFingerprint, model)])
		out[job.CompanyID] = append(out[job.CompanyID], answers)
	}
	return out, nil
}

var _ scoring.Store = (*FakeStore)(nil)

func (f *FakeStore) UpsertPushSubscription(_ context.Context, userID string, sub dto.PushSubscriptionInput) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deletePush(sub.Endpoint)
	f.pushes[userID] = append(f.pushes[userID], sub)
	return nil
}

func (f *FakeStore) ListPushSubscriptions(_ context.Context, userID string) ([]dto.PushSubscriptionInput, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.pushes[userID]), nil
}

func (f *FakeStore) DeletePushSubscription(_ context.Context, userID, endpoint string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pushes[userID] = slices.DeleteFunc(f.pushes[userID], func(s dto.PushSubscriptionInput) bool { return s.Endpoint == endpoint })
	return nil
}

func (f *FakeStore) deletePush(endpoint string) {
	for uid, subs := range f.pushes {
		f.pushes[uid] = slices.DeleteFunc(subs, func(s dto.PushSubscriptionInput) bool { return s.Endpoint == endpoint })
	}
}

func (f *FakeStore) InsertScoreFeedback(_ context.Context, userID string, entry dto.ScoreFeedback) (dto.ScoreFeedback, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.feedbackSeq++
	entry.ID = "feedback-" + strconv.Itoa(f.feedbackSeq)
	entry.CreatedAt = time.Now()
	f.feedback[userID] = append(f.feedback[userID], entry)
	return entry, nil
}

func (f *FakeStore) matchingFeedback(userID, kind string) []dto.ScoreFeedback {
	entries := slices.Clone(f.feedback[userID])
	slices.Reverse(entries)
	return slices.DeleteFunc(entries, func(e dto.ScoreFeedback) bool { return kind != "" && e.Kind != kind })
}

func (f *FakeStore) ListScoreFeedback(_ context.Context, userID, kind string, limit, offset int) ([]dto.ScoreFeedback, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	entries := f.matchingFeedback(userID, kind)
	entries = entries[min(offset, len(entries)):]
	return entries[:min(limit, len(entries))], nil
}

func (f *FakeStore) CountScoreFeedback(_ context.Context, userID, kind string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.matchingFeedback(userID, kind)), nil
}

func (f *FakeStore) DeleteScoreFeedback(_ context.Context, userID, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	entries := f.feedback[userID]
	i := slices.IndexFunc(entries, func(e dto.ScoreFeedback) bool { return e.ID == id })
	if i < 0 {
		return data.ErrNotFound
	}
	f.feedback[userID] = slices.Delete(entries, i, i+1)
	return nil
}

func (f *FakeStore) ClearScoreFeedback(_ context.Context, userID string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := int64(len(f.feedback[userID]))
	delete(f.feedback, userID)
	return n, nil
}

// SeedJobScore stores the score GetJobScoreForFeedback returns for the pair.
func (f *FakeStore) SeedJobScore(userID, jobID string, evidence dto.JobScoreEvidence) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.evidence[scoredKey(jobID, userID)] = evidence
}

func (f *FakeStore) GetJobScoreForFeedback(_ context.Context, userID, jobID string) (dto.JobScoreEvidence, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ev, ok := f.evidence[scoredKey(jobID, userID)]
	if !ok {
		return dto.JobScoreEvidence{}, data.ErrNotFound
	}
	return ev, nil
}

func (f *FakeStore) ListJobScoresForCollection(_ context.Context, userID string, jobIDs []string) ([]dto.CollectionJobScore, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []dto.CollectionJobScore
	for _, id := range jobIDs {
		job, ok := f.jobs[id]
		if !ok {
			continue
		}
		row := dto.CollectionJobScore{JobID: id, Title: job.Title, Company: job.CompanySlug}
		if ev, ok := f.evidence[scoredKey(id, userID)]; ok {
			row.Score, row.Breakdown = &ev.Score, ev.Breakdown
		}
		out = append(out, row)
	}
	return out, nil
}

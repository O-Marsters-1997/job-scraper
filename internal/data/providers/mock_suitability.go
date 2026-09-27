package providers

import (
	"context"
	"sync"

	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

// CompletedEffect records one MockSuitabilityProvider.CompleteAnswerEffect call.
type CompletedEffect struct {
	Effect  dto.AnswerEffect
	Answers map[string]dto.Answer
	Scores  []dto.JobScore
}

// MockSuitabilityProvider lives outside _test.go so it can be imported by tests in other packages.
type MockSuitabilityProvider struct {
	mu sync.Mutex

	effects []dto.AnswerEffect
	jobs    map[string]dto.Job
	configs map[string][]dto.SearchConfig
	answers map[string]map[string]dto.Answer
	inputs  map[string][]ScoringInput
	status  map[string]dto.ScoringStatus

	ClaimErr error

	Failed    []dto.ScoringFailure
	Completed []CompletedEffect
	Saved     []dto.JobScore
}

func NewMockSuitabilityProvider() *MockSuitabilityProvider {
	return &MockSuitabilityProvider{
		jobs:    make(map[string]dto.Job),
		configs: make(map[string][]dto.SearchConfig),
		answers: make(map[string]map[string]dto.Answer),
		inputs:  make(map[string][]ScoringInput),
		status:  make(map[string]dto.ScoringStatus),
	}
}

func answerKey(jobID, fingerprint, model string) string {
	return jobID + "|" + fingerprint + "|" + model
}

// SeedEffect queues effect to be returned by ClaimAnswerEffect, in order.
func (m *MockSuitabilityProvider) SeedEffect(effect dto.AnswerEffect) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.effects = append(m.effects, effect)
}

// SeedJob makes job available to GetJobForScoring and ListInterestedConfigs.
func (m *MockSuitabilityProvider) SeedJob(job dto.Job, configs []dto.SearchConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[job.ID] = job
	m.configs[job.ID] = configs
}

// SeedAnswers makes answers available to ListAnswers for (jobID, fingerprint, model).
func (m *MockSuitabilityProvider) SeedAnswers(jobID, fingerprint, model string, answers map[string]dto.Answer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.answers[answerKey(jobID, fingerprint, model)] = answers
}

// SeedScoringInputs makes inputs available to ListScoringInputs for userID.
func (m *MockSuitabilityProvider) SeedScoringInputs(userID string, inputs []ScoringInput) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inputs[userID] = inputs
}

func (m *MockSuitabilityProvider) ClaimAnswerEffect(_ context.Context) (dto.AnswerEffect, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ClaimErr != nil {
		return dto.AnswerEffect{}, m.ClaimErr
	}
	if len(m.effects) == 0 {
		return dto.AnswerEffect{}, data.ErrNotFound
	}
	e := m.effects[0]
	m.effects = m.effects[1:]
	return e, nil
}

func (m *MockSuitabilityProvider) FailAnswerEffect(_ context.Context, _ string, _ int, failure dto.ScoringFailure) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Failed = append(m.Failed, failure)
	return nil
}

func (m *MockSuitabilityProvider) GetJobForScoring(_ context.Context, jobID string) (dto.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[jobID]
	if !ok {
		return dto.Job{}, data.ErrNotFound
	}
	return job, nil
}

func (m *MockSuitabilityProvider) ListInterestedConfigs(_ context.Context, jobID string, _ bool) ([]dto.SearchConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.configs[jobID], nil
}

func (m *MockSuitabilityProvider) ListAnswers(_ context.Context, jobID, fingerprint, model string) (map[string]dto.Answer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := answerKey(jobID, fingerprint, model)
	out := make(map[string]dto.Answer, len(m.answers[key]))
	for h, a := range m.answers[key] {
		out[h] = a
	}
	return out, nil
}

func (m *MockSuitabilityProvider) CompleteAnswerEffect(_ context.Context, effect dto.AnswerEffect, answers map[string]dto.Answer, scores []dto.JobScore) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Completed = append(m.Completed, CompletedEffect{Effect: effect, Answers: answers, Scores: scores})
	saved := make([]string, len(scores))
	for i, s := range scores {
		saved[i] = s.UserID
	}
	return saved, nil
}

func (m *MockSuitabilityProvider) ListScoringInputs(_ context.Context, userID string) ([]ScoringInput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.inputs[userID], nil
}

func (m *MockSuitabilityProvider) SaveScores(_ context.Context, scores []dto.JobScore) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Saved = append(m.Saved, scores...)
	return nil
}

func (m *MockSuitabilityProvider) GetScoringStatus(_ context.Context, userID string) (dto.ScoringStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status[userID], nil
}

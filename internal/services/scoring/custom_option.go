package scoring

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
)

const draftSampleJobs = 5

func (s *Service) DraftCustomOption(ctx context.Context, userID string, in dto.CustomOptionDraftInput) (dto.CustomOptionDraft, error) {
	phrase, question := strings.TrimSpace(in.Phrase), strings.TrimSpace(in.Question)
	if phrase == "" {
		return dto.CustomOptionDraft{}, apperr.Invalid("phrase must not be blank")
	}

	draft := dto.CustomOptionDraft{Question: question, Dimension: in.Dimension}
	if question != "" && in.Dimension != "" && !slices.ContainsFunc(draftDimensions(), func(d dto.DimensionSpec) bool { return d.Key == in.Dimension }) {
		return dto.CustomOptionDraft{}, apperr.Invalid("dimension must be a non-ladder dimension")
	}
	apiKey, err := s.credentials.Get(ctx, userID, jev.Provider)
	if errors.Is(err, data.ErrNotFound) {
		return dto.CustomOptionDraft{}, apperr.Unprocessable("connect an OpenRouter credential to draft a question")
	}
	if err != nil {
		return dto.CustomOptionDraft{}, fmt.Errorf("scoring.DraftCustomOption: load credential: %w", err)
	}
	if question == "" {
		drafted, err := s.draftQuestion(ctx, apiKey, phrase)
		if err != nil {
			return dto.CustomOptionDraft{}, err
		}
		draft.Question, draft.Dimension = drafted.Question, drafted.Dimension
	}

	jobs, err := s.store.ListRecentScoredJobs(ctx, userID, draftSampleJobs)
	if err != nil {
		return dto.CustomOptionDraft{}, fmt.Errorf("scoring.DraftCustomOption: list recent jobs: %w", err)
	}
	draft.Answers = make([]dto.DraftAnswerOnJob, len(jobs))
	var g errgroup.Group
	q := draft.Question
	for i, job := range jobs {
		g.Go(func() error {
			answers, err := s.Ask(ctx, userID, job.ID, []string{q})
			if err != nil {
				return fmt.Errorf("scoring.DraftCustomOption: answer on job: %w", err)
			}
			draft.Answers[i] = dto.DraftAnswerOnJob{
				JobID: job.ID, Title: job.Title, Company: job.CompanySlug, Resolved: resolveAnswer(answers[q]),
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return dto.CustomOptionDraft{}, err
	}
	return draft, nil
}

func draftDimensions() []dto.DimensionSpec {
	return slices.DeleteFunc(slices.Clone(Dimensions), func(d dto.DimensionSpec) bool { return d.Kind == ladderKind })
}

func (s *Service) draftQuestion(ctx context.Context, apiKey, phrase string) (dto.DraftedQuestion, error) {
	dimensions := draftDimensions()
	drafted, err := s.drafter.Draft(ctx, apiKey, phrase, dimensions)
	if err != nil {
		return dto.DraftedQuestion{}, fmt.Errorf("scoring.DraftCustomOption: draft question: %w", err)
	}
	if strings.TrimSpace(drafted.Question) == "" || !slices.ContainsFunc(dimensions, func(d dto.DimensionSpec) bool { return d.Key == drafted.Dimension }) {
		return dto.DraftedQuestion{}, apperr.Upstream("the drafter returned an unusable question")
	}
	return drafted, nil
}

package scoring

import (
	"cmp"
	"context"
	"slices"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

const (
	previewJobs = 10
	previewTopN = 5
)

func (s *Service) Preview(ctx context.Context, userID string, in dto.ScoringPreviewInput) (dto.ScoringPreview, error) {
	pool, err := s.loadLabelPool(ctx, userID)
	if err != nil {
		return dto.ScoringPreview{}, err
	}
	picks, _, err := validatedPicks(pool.bank, in.Preferences.Picks, pool.cfg.Preferences.Picks)
	if err != nil {
		return dto.ScoringPreview{}, err
	}
	floor, err := validatedSalaryFloor(in.Preferences.SalaryFloor)
	if err != nil {
		return dto.ScoringPreview{}, err
	}
	unsaved := pool.cfg
	unsaved.Preferences = dto.Preferences{Picks: picks, SalaryFloor: floor, Scoring: pool.cfg.Preferences.Scoring}

	before := scoreInputs(userID, pool.cfg, pool.bank.byID, pool.inputs)
	after := scoreInputs(userID, unsaved, pool.bank.byID, pool.inputs)

	gradeOf := make(map[string]string, len(pool.grades))
	for _, g := range pool.grades {
		gradeOf[g.JobID] = g.Grade
	}
	out := dto.ScoringPreview{
		DismissedInTop5: dto.BeforeAfter{Before: dismissedInTop(before, gradeOf), After: dismissedInTop(after, gradeOf)},
		Jobs:            []dto.PreviewedJob{},
	}
	index := make(map[string]int, len(pool.inputs))
	for i, input := range pool.inputs {
		index[input.Job.ID] = i
	}
	for _, id := range previewJobIDs(pool, index) {
		i := index[id]
		job := pool.inputs[i].Job
		out.Jobs = append(out.Jobs, dto.PreviewedJob{
			JobID: id, Title: job.Title, Company: job.CompanySlug, Grade: gradeOf[id],
			Before: before[i].Score, After: after[i].Score, AfterBand: after[i].Band, AfterRows: after[i].Rows,
		})
	}
	return out, nil
}

func previewJobIDs(pool labelPool, index map[string]int) []string {
	var ids []string
	add := func(id string) {
		if _, ok := index[id]; ok && len(ids) < previewJobs && !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	grades := slices.Clone(pool.grades)
	slices.SortFunc(grades, func(a, b dto.Grade) int {
		return cmp.Or(b.UpdatedAt.Compare(a.UpdatedAt), cmp.Compare(a.JobID, b.JobID))
	})
	for _, g := range grades {
		add(g.JobID)
	}
	var implied []string
	for id, l := range pool.labels {
		if l.source != replaySourceGrade {
			implied = append(implied, id)
		}
	}
	slices.Sort(implied)
	for _, id := range implied {
		add(id)
	}
	for _, input := range pool.inputs {
		add(input.Job.ID)
	}
	return ids
}

func dismissedInTop(scores []dto.JobScore, gradeOf map[string]string) int {
	ranked := slices.Clone(scores)
	slices.SortFunc(ranked, func(a, b dto.JobScore) int {
		return cmp.Or(cmp.Compare(b.Score, a.Score), cmp.Compare(a.JobID, b.JobID))
	})
	top := ranked[:min(previewTopN, len(ranked))]
	return countWhere(top, func(sc dto.JobScore) bool { return gradeOf[sc.JobID] == replayGradeNegative })
}

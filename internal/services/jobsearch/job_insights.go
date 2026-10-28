package jobsearch

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

const scoreBandCount = 5

func (s *Service) Insights(ctx context.Context, userID string, q dto.JobInsightsQuery) (dto.JobInsights, error) {
	tz := cmp.Or(q.TZ, "UTC")
	if _, err := time.LoadLocation(tz); err != nil {
		return dto.JobInsights{}, apperr.Invalid("tz must be an IANA time zone name")
	}
	excluded, err := s.excludedCompanySlugs(ctx, userID)
	if err != nil {
		return dto.JobInsights{}, err
	}
	groups, err := s.store.JobInsightGroups(ctx, userID, dto.JobInsightsOptions{Now: time.Now(), TZ: tz, ExcludedCompanySlugs: excluded})
	if err != nil {
		return dto.JobInsights{}, err
	}
	return foldInsights(groups), nil
}

func foldInsights(groups []dto.JobInsightGroup) dto.JobInsights {
	out := dto.JobInsights{
		Daily:      []dto.JobInsightDay{},
		Sources:    []dto.JobInsightSource{},
		ScoreBands: make([]int, scoreBandCount),
		FoundBy:    []dto.JobInsightFinder{},
	}
	days := map[string]int{}
	type sourceTotal struct{ jobs, scored, scoreSum int }
	sources := map[string]*sourceTotal{}
	finders := map[string]int{}
	for _, g := range groups {
		out.Total += g.Jobs
		out.Scored += g.Scored
		days[g.Day] += g.Jobs
		if g.ScoreBand >= 0 {
			out.ScoreBands[g.ScoreBand] += g.Jobs
		}
		total := sources[g.Source]
		if total == nil {
			total = &sourceTotal{}
			sources[g.Source] = total
		}
		total.jobs += g.Jobs
		total.scored += g.Scored
		total.scoreSum += g.ScoreSum
		if g.FoundBy != "" {
			finders[g.FoundBy] += g.Jobs
		}
	}
	for day, count := range days {
		out.Daily = append(out.Daily, dto.JobInsightDay{Day: day, Count: count})
	}
	slices.SortFunc(out.Daily, func(a, b dto.JobInsightDay) int { return cmp.Compare(a.Day, b.Day) })
	for source, total := range sources {
		avg := 0
		if total.scored > 0 {
			avg = (2*total.scoreSum + total.scored) / (2 * total.scored)
		}
		out.Sources = append(out.Sources, dto.JobInsightSource{Source: source, Count: total.jobs, Scored: total.scored, AvgScore: avg})
	}
	slices.SortFunc(out.Sources, func(a, b dto.JobInsightSource) int {
		return cmp.Or(cmp.Compare(b.Count, a.Count), cmp.Compare(a.Source, b.Source))
	})
	for name, count := range finders {
		out.FoundBy = append(out.FoundBy, dto.JobInsightFinder{Name: name, Count: count})
	}
	slices.SortFunc(out.FoundBy, func(a, b dto.JobInsightFinder) int {
		return cmp.Or(cmp.Compare(b.Count, a.Count), cmp.Compare(a.Name, b.Name))
	})
	return out
}

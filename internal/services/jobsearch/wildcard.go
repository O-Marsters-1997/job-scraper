package jobsearch

import (
	"context"
	"hash/fnv"
	"math/rand/v2"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

const wildcardBlock = 10

// ListScored returns the user's scored jobs best first, with about one job in
// ten replaced by a flagged wildcard from the bottom half of the list. The
// choice is fixed for a (user, UTC day), so reloads see one order.
func (s *Service) ListScored(ctx context.Context, userID string) ([]dto.Job, error) {
	jobs, err := s.store.ListJobs(ctx, userID)
	if err != nil {
		return nil, err
	}
	return slotWildcards(jobs, userID, time.Now().UTC()), nil
}

func slotWildcards(jobs []dto.Job, userID string, day time.Time) []dto.Job {
	n := len(jobs)
	count := n / wildcardBlock
	if count == 0 {
		return jobs
	}

	seed := fnv.New64a()
	_, _ = seed.Write([]byte(userID + "|" + day.Format(time.DateOnly)))
	rng := rand.New(rand.NewPCG(seed.Sum64(), 0))

	bottom := (n + 1) / 2
	pool := rng.Perm(n - bottom)[:count]
	wildcards := make(map[int]dto.Job, count)
	taken := make(map[int]bool, count)
	for block, p := range pool {
		taken[bottom+p] = true
		job := jobs[bottom+p]
		job.Wildcard = true
		wildcards[block*wildcardBlock+rng.IntN(wildcardBlock)] = job
	}

	out := make([]dto.Job, 0, n)
	next := 0
	for i := range n {
		if job, ok := wildcards[i]; ok {
			out = append(out, job)
			continue
		}
		for taken[next] {
			next++
		}
		out = append(out, jobs[next])
		next++
	}
	return out
}

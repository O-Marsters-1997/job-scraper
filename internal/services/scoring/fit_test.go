package scoring

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func syntheticJobs(n int, rng *rand.Rand) []preparedJob {
	dims := []dto.Dimension{dto.DimensionRole, dto.DimensionDomain, dto.DimensionStage, dto.DimensionSize}
	jobs := make([]preparedJob, n)
	for i := range jobs {
		for _, d := range dims {
			yes := rng.Float64() < 0.5
			answer := dto.Answer{PNo: 1}
			if yes {
				answer = dto.Answer{PYes: 1}
			}
			jobs[i].picks = append(jobs[i].picks, evaluatedPick{dimension: d, key: string(d), stance: "nice", answer: answer, known: true})
		}
	}
	return jobs
}

func TestFitWeightsRecoversKnownWeights(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	truth := map[dto.Dimension]float64{dto.DimensionRole: 1, dto.DimensionDomain: 4, dto.DimensionStage: 2.5, dto.DimensionSize: 0.5}
	jobs := syntheticJobs(4000, rng)
	labels := make([]fitLabel, len(jobs))
	for i, j := range jobs {
		z := -3.5
		for k, x := range j.features()[:len(Dimensions)] {
			z += truth[Dimensions[k].Key] * x
		}
		labels[i] = fitLabel{job: i, positive: rng.Float64() < 1/(1+math.Exp(-z))}
	}

	got := fitWeights(jobs, labels)

	for dim, want := range truth {
		if math.Abs(got[dim]-want) > 0.6 {
			t.Errorf("fitWeights()[%s] = %.2f, want %.1f within 0.6", dim, got[dim], want)
		}
	}
}

func TestComputeUsesParams(t *testing.T) {
	pick := evaluatedPick{dimension: dto.DimensionTech, key: "tech:go", stance: "nice", answer: dto.Answer{PYes: 1}, known: true}
	miss := evaluatedPick{dimension: dto.DimensionRole, key: "role:backend", stance: "nice", answer: dto.Answer{PNo: 1}, known: true}
	picks := []evaluatedPick{pick, miss}

	base, _, _ := compute(picks, nil, "", nil, false, nil)
	heavy, _, _ := compute(picks, nil, "", nil, false, &dto.ScoringParams{Weights: map[dto.Dimension]float64{dto.DimensionTech: 10}})
	if heavy <= base {
		t.Errorf("score with tech weight 10 = %d, want above the default's %d", heavy, base)
	}

	_, band, _ := compute(picks, nil, "", nil, false, &dto.ScoringParams{Bands: &dto.BandCuts{Great: 10, Good: 5, Fair: 1}})
	if band != "great" {
		t.Errorf("band with Great from 10 = %q, want great", band)
	}
}

func TestCalibrateBands(t *testing.T) {
	scores := []int{90, 88, 70, 66, 30, 20}
	grades := []string{"great", "great", "ok", "ok", "no", "no"}

	got := calibrateBands(scores, grades)

	for i, sc := range scores {
		if !bandAgrees(grades[i], bandFor(sc, got)) {
			t.Errorf("calibrateBands() = %+v puts score %d in %s for a %s grade", got, sc, bandFor(sc, got), grades[i])
		}
	}
}

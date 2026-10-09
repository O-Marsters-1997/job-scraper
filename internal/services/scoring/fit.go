package scoring

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
)

const (
	minBandGrades   = 30
	minWeightLabels = 50
	refitEvery      = 15
	cvFolds         = 5
	fitIterations   = 1500
	fitRate         = 0.5
	fitPenalty      = 0.003
	minFittedWeight = 0.1
	bandStep        = 5
)

type fitLabel struct {
	job      int
	positive bool
}

type fitMetrics struct{ medianRankPct, concordance float64 }

func (m fitMetrics) atLeastAsGoodAs(champion fitMetrics) bool {
	return m.medianRankPct <= champion.medianRankPct && m.concordance >= champion.concordance
}

func (s *Service) Fit(ctx context.Context, userID string) (string, error) {
	pool, err := s.loadLabelPool(ctx, userID)
	if err != nil {
		return "", err
	}
	cfg := pool.cfg
	if cfg.UserID == "" {
		return "", apperr.Invalid("set scoring preferences before fitting")
	}
	if len(pool.grades) < minBandGrades {
		return fmt.Sprintf("%d grades, need %d: defaults apply\n", len(pool.grades), minBandGrades), nil
	}

	gradeOf := make(map[string]string, len(pool.grades))
	for _, g := range pool.grades {
		gradeOf[g.JobID] = g.Grade
	}
	jobs := make([]preparedJob, len(pool.inputs))
	var labels []fitLabel
	var graded []int
	for i, in := range pool.inputs {
		jobs[i] = prepareJob(cfg, in.Job, pool.bank.byID, in.Answers, in.Corrections, in.Favourite)
		if l, ok := pool.labels[in.Job.ID]; ok {
			labels = append(labels, fitLabel{job: i, positive: l.positive})
		}
		if _, ok := gradeOf[in.Job.ID]; ok {
			graded = append(graded, i)
		}
	}

	next := dto.ScoringParams{FittedAt: time.Now().UTC(), GradeCount: len(pool.grades)}
	champion := cfg.Preferences.Scoring
	if champion != nil {
		next.Weights, next.Bands = champion.Weights, champion.Bands
	}
	var report strings.Builder

	positives := countWhere(labels, func(l fitLabel) bool { return l.positive })
	negatives := len(labels) - positives
	if len(labels) < minWeightLabels || positives < replayMinNegatives || negatives < replayMinNegatives {
		fmt.Fprintf(&report, "Weights unchanged: %d labels (%d positive, %d negative), need %d with %d of each\n",
			len(labels), positives, negatives, minWeightLabels, replayMinNegatives)
	} else {
		fitted := func(train []fitLabel) *dto.ScoringParams {
			return &dto.ScoringParams{Weights: fitWeights(jobs, train)}
		}
		kept := func([]fitLabel) *dto.ScoringParams { return champion }
		got, want := crossValidate(jobs, labels, fitted), crossValidate(jobs, labels, kept)
		verdict := "accepted"
		if got.atLeastAsGoodAs(want) {
			next.Weights = fitWeights(jobs, labels)
		} else {
			verdict = "rejected"
			slog.WarnContext(ctx, "weights challenger rejected",
				slog.String(logger.KeyUserID, userID),
				slog.Float64("challenger_rank_pct", got.medianRankPct), slog.Float64("champion_rank_pct", want.medianRankPct),
				slog.Float64("challenger_concordance", got.concordance), slog.Float64("champion_concordance", want.concordance))
		}
		fmt.Fprintf(&report, "Weights %s: rank %.1f vs %.1f, concordance %.3f vs %.3f\n",
			verdict, got.medianRankPct, want.medianRankPct, got.concordance, want.concordance)
	}

	scores := make([]int, len(graded))
	grades := make([]string, len(graded))
	for i, j := range graded {
		scores[i], _, _ = jobs[j].compute(&next)
		grades[i] = gradeOf[pool.inputs[j].Job.ID]
	}
	cuts := calibrateBands(scores, grades)
	next.Bands = &cuts
	fmt.Fprintf(&report, "Bands: Great from %d, Good from %d, Fair from %d\n", cuts.Great, cuts.Good, cuts.Fair)

	cfg.Preferences.Scoring = &next
	if _, err := s.store.UpsertSearchConfig(ctx, cfg); err != nil {
		return "", fmt.Errorf("scoring.Fit: save params: %w", err)
	}
	if _, err := s.Recompute(ctx, userID); err != nil {
		return "", fmt.Errorf("scoring.Fit: recompute: %w", err)
	}
	return report.String(), nil
}

func (s *Service) refitIfDue(ctx context.Context, userID string) {
	if err := s.fitIfDue(ctx, userID); err != nil {
		slog.ErrorContext(ctx, "scoring refit failed", slog.String(logger.KeyUserID, userID), slog.Any(logger.KeyErr, err))
	}
}

func (s *Service) fitIfDue(ctx context.Context, userID string) error {
	grades, err := s.store.ListGrades(ctx, userID)
	if err != nil || len(grades) < minBandGrades {
		return err
	}
	cfg, err := s.searchConfigOrDefault(ctx, userID)
	if err != nil || cfg.UserID == "" {
		return err
	}
	fitted := 0
	if cfg.Preferences.Scoring != nil {
		fitted = cfg.Preferences.Scoring.GradeCount
	}
	if len(grades)-fitted < refitEvery {
		return nil
	}
	_, err = s.Fit(ctx, userID)
	return err
}

func (p preparedJob) features() []float64 {
	picks := p.picks
	if p.floor != nil {
		picks = append(slices.Clone(picks), salaryPick(*p.floor, p.salaryRaw))
	}
	x := make([]float64, len(Dimensions)+1)
	nice := make(map[dto.Dimension]*niceDimension)
	for _, pk := range picks {
		if pk.retired || !pk.known {
			continue
		}
		switch pk.stance {
		case "nice":
			d, ok := nice[pk.dimension]
			if !ok {
				d = &niceDimension{}
				nice[pk.dimension] = d
			}
			d.sumYes += pk.answer.PYes
			d.evidence = max(d.evidence, pk.answer.PYes+pk.answer.PNo)
		case "avoid":
			x[len(Dimensions)] += pk.answer.PYes
		}
	}
	for i, spec := range Dimensions {
		if d, ok := nice[spec.Key]; ok {
			x[i] = d.evidence * min(1, d.sumYes/float64(spec.Saturation))
		}
	}
	return x
}

func fitWeights(jobs []preparedJob, labels []fitLabel) map[dto.Dimension]float64 {
	cols := len(Dimensions) + 1
	x := make([][]float64, len(labels))
	for i, l := range labels {
		x[i] = jobs[l.job].features()
	}
	centre := make([]float64, cols)
	for i, d := range Dimensions {
		centre[i] = d.Weight
	}
	centre[cols-1] = -avoidWeight

	theta, bias := slices.Clone(centre), 0.0
	grad := make([]float64, cols)
	n := float64(len(labels))
	for range fitIterations {
		clear(grad)
		var gradBias float64
		for i, l := range labels {
			z := bias
			for j, v := range x[i] {
				z += theta[j] * v
			}
			miss := 1/(1+math.Exp(-z)) - boolToFloat(l.positive)
			gradBias += miss
			for j, v := range x[i] {
				grad[j] += miss * v
			}
		}
		bias -= fitRate * gradBias / n
		for j := range theta {
			theta[j] -= fitRate * (grad[j]/n + 2*fitPenalty*(theta[j]-centre[j]))
		}
	}

	weights := make(map[dto.Dimension]float64, len(Dimensions))
	for i, d := range Dimensions {
		weights[d.Key] = max(theta[i], minFittedWeight)
	}
	return weights
}

func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func crossValidate(jobs []preparedJob, labels []fitLabel, train func([]fitLabel) *dto.ScoringParams) fitMetrics {
	var positives, negatives []replayRow
	var rankPcts []float64
	for fold := range cvFolds {
		var trainSet, heldOut []fitLabel
		for i, l := range labels {
			if i%cvFolds == fold {
				heldOut = append(heldOut, l)
			} else {
				trainSet = append(trainSet, l)
			}
		}
		params := train(trainSet)
		scores := make([]int, len(jobs))
		for i, j := range jobs {
			scores[i], _, _ = j.compute(params)
		}
		for _, l := range heldOut {
			row := replayRow{Positive: l.positive, Score: scores[l.job]}
			if !l.positive {
				negatives = append(negatives, row)
				continue
			}
			positives = append(positives, row)
			rank := 1 + countWhere(scores, func(sc int) bool { return sc > row.Score })
			rankPcts = append(rankPcts, 100*float64(rank)/float64(len(scores)))
		}
	}
	return fitMetrics{medianRankPct: median(rankPcts), concordance: concordance(positives, negatives)}
}

func calibrateBands(scores []int, grades []string) dto.BandCuts {
	best, bestHits, bestDist := defaultBands, -1, math.MaxInt
	for fair := bandStep; fair <= 100; fair += bandStep {
		for good := fair + bandStep; good <= 100; good += bandStep {
			for great := good + bandStep; great <= 100; great += bandStep {
				cuts := dto.BandCuts{Great: great, Good: good, Fair: fair}
				hits := 0
				for i, sc := range scores {
					if bandAgrees(grades[i], bandFor(sc, cuts)) {
						hits++
					}
				}
				dist := absInt(great-defaultBands.Great) + absInt(good-defaultBands.Good) + absInt(fair-defaultBands.Fair)
				if hits > bestHits || hits == bestHits && dist < bestDist {
					best, bestHits, bestDist = cuts, hits, dist
				}
			}
		}
	}
	return best
}

func bandAgrees(grade, band string) bool {
	switch grade {
	case "great":
		return band == "great"
	case "ok":
		return band == "good" || band == "fair"
	}
	return band == "poor"
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

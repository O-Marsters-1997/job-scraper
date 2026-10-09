package scoring

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
	"github.com/ollymarsters/job-scraper/internal/services/scoring/store"
)

const (
	replayTopN          = 20
	replayMinNegatives  = 5
	replayLowScoredPct  = 50
	replaySourceGrade   = "grade"
	replayGradeNegative = "no"
)

type replayRow struct {
	JobID    string
	Title    string
	Company  string
	Positive bool
	Source   string
	Score    int
	Band     string
	Rank     int
	Reasons  []string
}

func (r replayRow) rankPct(total int) float64 { return 100 * float64(r.Rank) / float64(total) }

type replayReport struct {
	Total    int
	Scores   []int
	Labelled []replayRow
	Unscored int
	Params   *dto.ScoringParams
}

// Replay re-runs compute over userID's cached answers for every scored job
// and renders where their labelled jobs rank. It never calls Jev and writes
// nothing.
func (s *Service) Replay(ctx context.Context, userID string) (string, error) {
	report, err := s.replayReport(ctx, userID)
	if err != nil {
		return "", err
	}
	return renderReplay(report), nil
}

type labelPool struct {
	cfg    dto.SearchConfig
	bank   bank
	inputs []store.ScoringInput
	grades []dto.Grade
	labels map[string]replayLabelInfo
}

func (s *Service) loadLabelPool(ctx context.Context, userID string) (labelPool, error) {
	cfg, err := s.searchConfigOrDefault(ctx, userID)
	if err != nil {
		return labelPool{}, err
	}
	bk, err := s.loadBank(ctx)
	if err != nil {
		return labelPool{}, err
	}
	inputs, err := s.store.ListScoringInputs(ctx, userID, jev.Model)
	if err != nil {
		return labelPool{}, err
	}
	grades, err := s.store.ListGrades(ctx, userID)
	if err != nil {
		return labelPool{}, err
	}
	implied, err := s.store.ListImpliedPositives(ctx, userID)
	if err != nil {
		return labelPool{}, err
	}
	return labelPool{cfg: cfg, bank: bk, inputs: inputs, grades: grades, labels: replayLabels(grades, implied)}, nil
}

func (s *Service) replayReport(ctx context.Context, userID string) (replayReport, error) {
	pool, err := s.loadLabelPool(ctx, userID)
	if err != nil {
		return replayReport{}, err
	}
	cfg, labels := pool.cfg, pool.labels

	report := replayReport{Total: len(pool.inputs), Scores: make([]int, len(pool.inputs)), Params: cfg.Preferences.Scoring}
	var labelled []replayRow
	for i, in := range pool.inputs {
		scored := scoreJob(userID, cfg, in.Job, pool.bank.byID, in.Answers, in.Corrections, in.Favourite)
		score := scored.Score
		report.Scores[i] = score
		if l, ok := labels[in.Job.ID]; ok {
			labelled = append(labelled, replayRow{
				JobID: in.Job.ID, Title: in.Job.Title, Company: in.Job.CompanySlug,
				Positive: l.positive, Source: l.source, Score: score, Band: scored.Band, Reasons: l.reasons,
			})
		}
	}
	for i := range labelled {
		labelled[i].Rank = 1 + countWhere(report.Scores, func(sc int) bool { return sc > labelled[i].Score })
	}
	slices.SortFunc(labelled, func(a, b replayRow) int {
		return cmp.Or(cmp.Compare(a.Rank, b.Rank), cmp.Compare(a.JobID, b.JobID))
	})
	report.Labelled = labelled
	report.Unscored = len(labels) - len(labelled)
	return report, nil
}

type replayLabelInfo struct {
	positive bool
	source   string
	reasons  []string
}

func replayLabels(grades []dto.Grade, implied []dto.ImpliedLabel) map[string]replayLabelInfo {
	labels := make(map[string]replayLabelInfo, len(grades)+len(implied))
	for _, g := range grades {
		labels[g.JobID] = replayLabelInfo{positive: g.Grade != replayGradeNegative, source: replaySourceGrade, reasons: g.Reasons}
	}
	for _, l := range implied {
		if _, ok := labels[l.JobID]; !ok {
			labels[l.JobID] = replayLabelInfo{positive: true, source: l.Source}
		}
	}
	return labels
}

func countWhere[T any](xs []T, pred func(T) bool) int {
	n := 0
	for _, x := range xs {
		if pred(x) {
			n++
		}
	}
	return n
}

func renderReplay(r replayReport) string {
	var sb strings.Builder
	sb.WriteString("# Replay\n\n")
	writeReplayParams(&sb, r.Params)
	writeReplaySummary(&sb, r)
	if len(r.Labelled) == 0 {
		return sb.String()
	}
	sb.WriteString("\n## Labelled jobs\n\n| Job | Company | Label | Source | Score | Band | Rank |\n|---|---|---|---|---|---|---|\n")
	for _, row := range r.Labelled {
		fmt.Fprintf(&sb, "| %s | %s | %s | %s | %d | %s | %d |\n", cell(row.Title), cell(row.Company), replayLabel(row), row.Source, row.Score, row.Band, row.Rank)
	}
	writeReplayReasons(&sb, r.Labelled)
	return sb.String()
}

func writeReplayReasons(sb *strings.Builder, rows []replayRow) {
	positives, negatives := map[string]int{}, map[string]int{}
	for _, row := range rows {
		counts := negatives
		if row.Positive {
			counts = positives
		}
		for _, reason := range row.Reasons {
			counts[reason]++
		}
	}
	if len(positives)+len(negatives) == 0 {
		return
	}
	sb.WriteString("\n## Labels by reason\n\n| Reason | Negatives | Positives |\n|---|---|---|\n")
	for _, reason := range gradeReasons {
		if n, p := negatives[reason], positives[reason]; n+p > 0 {
			fmt.Fprintf(sb, "| %s | %d | %d |\n", reason, n, p)
		}
	}
}

func replayLabel(row replayRow) string {
	if row.Positive {
		return "positive"
	}
	return "negative"
}

func writeReplaySummary(sb *strings.Builder, r replayReport) {
	var positives, negatives []replayRow
	for _, row := range r.Labelled {
		if row.Positive {
			positives = append(positives, row)
		} else {
			negatives = append(negatives, row)
		}
	}
	fmt.Fprintf(sb, "%d scored jobs · %d positives · %d negatives", r.Total, len(positives), len(negatives))
	if r.Unscored > 0 {
		fmt.Fprintf(sb, " · %d labelled jobs without a score excluded", r.Unscored)
	}
	sb.WriteString("\n\n")

	if len(positives) == 0 {
		sb.WriteString("No positives yet: grade a job great or ok, apply, or keep a tailored CV.\n")
		return
	}

	pcts := make([]float64, len(positives))
	for i, p := range positives {
		pcts[i] = p.rankPct(r.Total)
	}
	medianPct := median(pcts)
	inTop := countWhere(positives, func(p replayRow) bool { return p.Rank <= replayTopN })
	fmt.Fprintf(sb, "- Positive rank: median rank percentile %.0f (lower is better), %d of %d in the top %d\n", medianPct, inTop, len(positives), replayTopN)

	var low []string
	for _, p := range positives {
		if p.rankPct(r.Total) > replayLowScoredPct {
			low = append(low, fmt.Sprintf("%s (%s, score %d)", p.Title, p.Company, p.Score))
		}
	}
	if len(low) == 0 {
		sb.WriteString("- Low-scored positives: none\n")
	} else {
		fmt.Fprintf(sb, "- Low-scored positives: %d in the bottom half: %s\n", len(low), strings.Join(low, "; "))
	}

	if len(negatives) >= replayMinNegatives {
		fmt.Fprintf(sb, "- Concordance: %.0f%% of positive/negative pairs score the positive higher\n", 100*concordance(positives, negatives))
	}

	top := slices.Max(r.Scores)
	distinct := len(slices.Compact(slices.Sorted(slices.Values(r.Scores))))
	fmt.Fprintf(sb, "- Distinct scores: %d, with %d jobs tied at the top score of %d\n", distinct, countWhere(r.Scores, func(s int) bool { return s == top }), top)
}

func median(xs []float64) float64 {
	slices.Sort(xs)
	mid := len(xs) / 2
	if len(xs)%2 == 0 {
		return (xs[mid-1] + xs[mid]) / 2
	}
	return xs[mid]
}

func concordance(positives, negatives []replayRow) float64 {
	var wins float64
	for _, p := range positives {
		for _, n := range negatives {
			switch {
			case p.Score > n.Score:
				wins++
			case p.Score == n.Score:
				wins += 0.5
			}
		}
	}
	return wins / float64(len(positives)*len(negatives))
}

func writeReplayParams(sb *strings.Builder, p *dto.ScoringParams) {
	if p == nil {
		sb.WriteString("Scoring: default weights and Bands\n\n")
		return
	}
	fmt.Fprintf(sb, "Scoring: fitted %s from %d grades\n", p.FittedAt.Format(time.DateOnly), p.GradeCount)
	if len(p.Weights) == 0 {
		sb.WriteString("- Weights: defaults\n")
	} else {
		parts := make([]string, 0, len(p.Weights))
		for _, d := range Dimensions {
			if w, ok := p.Weights[d.Key]; ok {
				parts = append(parts, fmt.Sprintf("%s %.2f", d.Key, w))
			}
		}
		fmt.Fprintf(sb, "- Weights: fitted: %s\n", strings.Join(parts, ", "))
	}
	b := bandsFor(p)
	fmt.Fprintf(sb, "- Bands: Great from %d, Good from %d, Fair from %d\n\n", b.Great, b.Good, b.Fair)
}

package scoring

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

type packPick struct {
	Label    string
	Stance   string
	Question string
}

func packPicks(picks []dto.Pick, b bank) []packPick {
	out := make([]packPick, 0, len(picks))
	for _, p := range dedupeBySource(picks) {
		opt, ok := b.byID[p.OptionID]
		if !ok {
			continue
		}
		label := opt.Label
		if opt.RetiredAt != nil {
			label += " (retired)"
		}
		stance := p.Stance
		if p.Weight > 0 {
			stance = fmt.Sprintf("weight %d", p.Weight)
		}
		out = append(out, packPick{Label: label, Stance: stance, Question: opt.Question})
	}
	return out
}

func renderPack(entries []dto.ScoreFeedback, picks []packPick, replay replayReport, outdatedOmitted int, includeOutdated bool) string {
	var overall, jobs, collections []dto.ScoreFeedback
	for _, e := range entries {
		switch e.Kind {
		case feedbackKindOverall:
			overall = append(overall, e)
		case feedbackKindJob:
			jobs = append(jobs, e)
		case feedbackKindCollection:
			collections = append(collections, e)
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Feedback Pack\n\n%d overall · %d Job entries · %d Collection entries", len(overall), len(jobs), len(collections))
	if !includeOutdated {
		fmt.Fprintf(&sb, " · %d outdated omitted (--include-outdated)", outdatedOmitted)
	}
	sb.WriteString("\n\n")

	sb.WriteString("## How Suitability is computed\n\n")
	sb.WriteString("Each Job is scored 0-100 from the user's Picks and Jev's cached answers to each Option's question.\n")
	fmt.Fprintf(&sb, "An answer resolves to yes, no or unknown for the checklist rows: the top of P(yes), P(no) and P(not stated) wins when it reaches %.1f, otherwise unknown. The score itself uses the probabilities.\n\n", resolveThreshold)
	fmt.Fprintf(&sb, "Score = round(100 * (sum(weight * (evidence * credit + %[1]g * missing)) + %[1]g * prior) / (sum(weight * (evidence + missing)) + avoid cost + prior)), with missing = alpha * (1 - evidence), alpha = %[2]g and prior = %[3]g.\n\n", 0.5, missingAlpha, prior)
	fmt.Fprintf(&sb, "- Per dimension with nice or ok Picks, credit = min(1, sum of strength * P(yes) over its Picks / saturation), with strength %g for nice and %g for ok, and evidence = the largest P(yes) + P(no) over its Picks. Dimension weights and saturation:\n", pickStrength["nice"], pickStrength["ok"])
	for _, d := range Dimensions {
		fmt.Fprintf(&sb, "  - %s: weight %.0f, saturation %d%s\n", d.Key, d.Weight, d.Saturation, gateNote(d))
	}
	fmt.Fprintf(&sb, "- An avoid Pick adds %d * P(yes) to the denominator and nothing to the numerator.\n", avoidWeight)
	fmt.Fprintf(&sb, "- A Gate caps the score at %d: every Pick in a Gate dimension resolves no and an option the user did not pick resolves yes. A salary below the floor also gates.\n", gateCap)
	fmt.Fprintf(&sb, "- seniority is a ladder instead: Junior 1, Mid 2, Senior 3, Lead/Staff 4, Principal/Head 5. The user's tier weights (1-100) give a point (weighted mean level) and a tolerance (weighted spread, at least %g). The job's level is sum(level * P(yes)) / sum(P(yes)) over the tier answers, evidence = min(1, sum(P(yes))), and credit = exp(-d^2 / 2) with d = |job level - point| / tolerance. It gates when evidence reaches %.1f and d exceeds %g.\n", ladderToleranceFloor, resolveThreshold, ladderGateSpread)
	sb.WriteString("- A block Pick resolving yes forces the score to 0.\n")
	fmt.Fprintf(&sb, "- Bands: Great from %d, Good from %d, Fair from %d, Poor below.\n", bandGreatMin, bandGoodMin, bandFairMin)
	sb.WriteString("- A picked dimension's missing evidence counts as a coin flip, pulling the score toward 50. A dimension with no Picks counts on neither side.\n\n")

	sb.WriteString("Current Picks:\n\n")
	if len(picks) == 0 {
		sb.WriteString("(none)\n\n")
	} else {
		sb.WriteString("| Option | Stance | Question |\n|---|---|---|\n")
		for _, p := range picks {
			fmt.Fprintf(&sb, "| %s | %s | %s |\n", cell(p.Label), p.Stance, cell(p.Question))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Levers you may change\n\n")
	fmt.Fprintf(&sb, "- Dimension weights and saturation, avoidWeight (%d), prior (%.0f), alpha (%g), gateCap (%d)\n", avoidWeight, prior, missingAlpha, gateCap)
	fmt.Fprintf(&sb, "- resolveThreshold (%.1f)\n", resolveThreshold)
	sb.WriteString("- Question wording\n")
	sb.WriteString("- New or retired Options\n")
	sb.WriteString("- The user's Picks\n\n")

	sb.WriteString("## Overall feedback\n\n")
	if len(overall) == 0 {
		sb.WriteString("(none)\n\n")
	}
	for _, e := range overall {
		if tag := driftTag(e); tag != "" {
			fmt.Fprintf(&sb, "[%s]\n\n", tag)
		}
		writeReason(&sb, e.Reason)
	}

	sb.WriteString("## Replay\n\n")
	writeReplaySummary(&sb, replay)
	sb.WriteString("\n")

	sb.WriteString("## Job entries\n\n")
	if len(jobs) == 0 {
		sb.WriteString("(none)\n")
	}
	for _, e := range jobs {
		writeJobEntry(&sb, e)
	}

	sb.WriteString("## Collection entries\n\n")
	if len(collections) == 0 {
		sb.WriteString("(none)\n")
	}
	for _, e := range collections {
		writeCollectionEntry(&sb, e)
	}
	return sb.String()
}

func driftTag(e dto.ScoreFeedback) string {
	var tags []string
	if e.PicksChanged {
		tags = append(tags, "picks changed")
	}
	if e.ModelChanged {
		tags = append(tags, "model changed")
	}
	return strings.Join(tags, ", ")
}

func writeReason(sb *strings.Builder, reason string) {
	for _, line := range strings.Split(reason, "\n") {
		fmt.Fprintf(sb, "> %s\n", line)
	}
	sb.WriteString("\n")
}

func writeJobEntry(sb *strings.Builder, e dto.ScoreFeedback) {
	snap := e.Snapshot
	var state dto.JevState
	if snap.JevState != nil {
		state = *snap.JevState
	}
	var score int
	if snap.Score != nil {
		score = *snap.Score
	}
	direction := ""
	if e.Direction != nil {
		direction = *e.Direction
	}
	heading := fmt.Sprintf("%s, %s: score %d, should be %s", state.Title, state.Company, score, direction)
	if tag := driftTag(e); tag != "" {
		heading += " [" + tag + "]"
	}
	fmt.Fprintf(sb, "### %s\n\n", heading)
	writeReason(sb, e.Reason)
	fmt.Fprintf(sb, "- Model: %s\n- Score model: %s\n- Score fingerprint: %s\n- Content fingerprint: %s\n\n",
		e.Model, snap.ScoreModel, snap.ScoreFingerprint, snap.ContentFingerprint)

	if len(snap.Options) == 0 {
		sb.WriteString("No Options were picked.\n\n")
	} else {
		sb.WriteString("| Option | Stance | Resolved | P(yes) | P(no) | P(not stated) | Confidence |\n|---|---|---|---|---|---|---|\n")
		for _, o := range snap.Options {
			if !o.Known {
				fmt.Fprintf(sb, "| %s | %s | %s | no cached answer | | | |\n", cell(o.Label), o.Stance, o.Resolved)
				continue
			}
			fmt.Fprintf(sb, "| %s | %s | %s | %.2f | %.2f | %.2f | %.2f |\n",
				cell(o.Label), o.Stance, o.Resolved, o.PYes, o.PNo, o.PNotStated, o.Confidence)
		}
		sb.WriteString("\n")
	}

	stateJSON, _ := json.MarshalIndent(state, "", "  ")
	fence := strings.Repeat("`", max(3, longestRun(string(stateJSON), '`')+1))
	fmt.Fprintf(sb, "Job state sent to Jev:\n\n%sjson\n%s\n%s\n\n", fence, stateJSON, fence)
}

func writeCollectionEntry(sb *strings.Builder, e dto.ScoreFeedback) {
	snap := e.Snapshot
	fmt.Fprintf(sb, "### Ranking of %d Jobs\n\n", len(snap.Ranking))
	writeReason(sb, e.Reason)

	filters := make([]string, 0, len(snap.Filters))
	for k, v := range snap.Filters {
		filters = append(filters, k+"="+v)
	}
	slices.Sort(filters)
	if len(filters) == 0 {
		filters = append(filters, "(none)")
	}
	fmt.Fprintf(sb, "- Model: %s\n- Filters: %s\n\n", e.Model, strings.Join(filters, ", "))

	for _, r := range snap.Ranking {
		score := "no score"
		if r.Score != nil {
			score = strconv.Itoa(*r.Score)
		}
		fmt.Fprintf(sb, "%d. %s · %s — %s", r.Rank, score, r.Title, r.Company)
		if len(r.Effects) > 0 {
			fmt.Fprintf(sb, " · %s", strings.Join(r.Effects, ", "))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
}

func longestRun(s string, r rune) int {
	longest, run := 0, 0
	for _, c := range s {
		if c != r {
			run = 0
			continue
		}
		run++
		longest = max(longest, run)
	}
	return longest
}

var cellEscaper = strings.NewReplacer("|", `\|`, "\n", " ")

func cell(s string) string { return cellEscaper.Replace(s) }

func gateNote(d dto.DimensionSpec) string {
	if d.Gate {
		return ", Gate"
	}
	return ""
}

package scoring

import (
	"encoding/json"
	"fmt"
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
		out = append(out, packPick{Label: label, Stance: p.Stance, Question: opt.Question})
	}
	return out
}

func renderPack(entries []dto.ScoreFeedback, picks []packPick, outdatedOmitted int, includeOutdated bool) string {
	var overall, jobs []dto.ScoreFeedback
	for _, e := range entries {
		switch e.Kind {
		case feedbackKindOverall:
			overall = append(overall, e)
		case feedbackKindJob:
			jobs = append(jobs, e)
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Feedback Pack\n\n%d overall · %d Job entries", len(overall), len(jobs))
	if !includeOutdated {
		fmt.Fprintf(&sb, " · %d outdated omitted (--include-outdated)", outdatedOmitted)
	}
	sb.WriteString("\n\n")

	sb.WriteString("## How Suitability is computed\n\n")
	sb.WriteString("Each Job is scored 0-100 from the user's Picks and Jev's cached answers to each Option's question.\n")
	fmt.Fprintf(&sb, "An answer resolves to yes, no or unknown: the top of P(yes), P(no) and P(not stated) wins when it reaches %.1f, otherwise unknown.\n\n", resolveThreshold)
	fmt.Fprintf(&sb, "Score = round(100 * (met + %.1f * %d) / (evaluable + %d)).\n\n", 0.5, priorK, priorK)
	fmt.Fprintf(&sb, "- A nice Pick counts per dimension: %d to evaluable once any of the dimension's nice Picks resolves yes or no, and %d to met if one resolves yes.\n", niceWeight, niceWeight)
	fmt.Fprintf(&sb, "- An avoid Pick resolving yes adds %d to evaluable and nothing to met.\n", avoidWeight)
	sb.WriteString("- A block Pick resolving yes forces the score to 0.\n")
	sb.WriteString("- Unknown counts on neither side.\n\n")

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
	fmt.Fprintf(&sb, "- Weights: niceWeight (%d), avoidWeight (%d), priorK (%d)\n", niceWeight, avoidWeight, priorK)
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

	sb.WriteString("## Job entries\n\n")
	if len(jobs) == 0 {
		sb.WriteString("(none)\n")
	}
	for _, e := range jobs {
		writeJobEntry(&sb, e)
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

package scoring

import (
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

func renderPack(entries []dto.ScoreFeedback, picks []packPick) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Feedback Pack\n\n%d overall entries\n\n", len(entries))

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
	if len(entries) == 0 {
		sb.WriteString("(none)\n")
	}
	for _, e := range entries {
		for _, line := range strings.Split(e.Reason, "\n") {
			fmt.Fprintf(&sb, "> %s\n", line)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

var cellEscaper = strings.NewReplacer("|", `\|`, "\n", " ")

func cell(s string) string { return cellEscaper.Replace(s) }

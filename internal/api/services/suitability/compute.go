package suitability

import (
	"crypto/sha256"
	"encoding/hex"
	"math"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

const (
	niceWeight       = 1
	avoidWeight      = 2
	priorK           = 3
	resolveThreshold = 0.6
)

type evaluatedPick struct {
	dimension dto.Dimension
	key       string
	label     string
	stance    string
	answer    dto.Answer
	known     bool
}

func resolveAnswer(a dto.Answer) string {
	top, value := a.PYes, "yes"
	if a.PNo > top {
		top, value = a.PNo, "no"
	}
	if a.PNotStated > top {
		top, value = a.PNotStated, "unknown"
	}
	if top < resolveThreshold {
		return "unknown"
	}
	return value
}

func compute(picks []evaluatedPick, salaryRaw string, floor *dto.Money) (int, []dto.ScoreRow, bool) {
	if floor != nil {
		picks = append(picks, salaryPick(*floor, salaryRaw))
	}
	rows := make([]dto.ScoreRow, 0, len(picks))
	type dimState struct{ known, matched bool }
	dims := make(map[dto.Dimension]*dimState)
	var met, evaluable float64
	var hidden bool

	for _, p := range picks {
		resolved := "unknown"
		if p.known {
			resolved = resolveAnswer(p.answer)
		}
		row := dto.ScoreRow{Key: p.key, Label: p.label, Stance: p.stance, Resolved: resolved}

		switch p.stance {
		case "nice":
			d, ok := dims[p.dimension]
			if !ok {
				d = &dimState{}
				dims[p.dimension] = d
			}
			switch resolved {
			case "yes":
				d.known, d.matched = true, true
				row.Effect = "meets"
			case "no":
				d.known = true
				row.Effect = "misses"
			default:
				row.Effect = "unknown"
			}
		case "avoid", "block":
			switch resolved {
			case "yes":
				evaluable += avoidWeight
				row.Effect = "misses"
				if p.stance == "block" {
					hidden = true
				}
			case "no":
				row.Effect = "neutral"
			default:
				row.Effect = "unknown"
			}
		default:
			row.Effect = "unknown"
		}
		rows = append(rows, row)
	}

	for _, d := range dims {
		if !d.known {
			continue
		}
		evaluable += niceWeight
		if d.matched {
			met += niceWeight
		}
	}

	score := int(math.Round(100 * (met + 0.5*priorK) / (evaluable + priorK)))
	return score, rows, hidden
}

func countUnknown(rows []dto.ScoreRow) int {
	n := 0
	for _, r := range rows {
		if r.Effect == "unknown" {
			n++
		}
	}
	return n
}

func questionHash(question string) string {
	sum := sha256.Sum256([]byte(question))
	return hex.EncodeToString(sum[:])
}

package scoring

import (
	"crypto/sha256"
	"encoding/hex"
	"math"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

const (
	avoidWeight      = 2
	prior            = 1.0
	resolveThreshold = 0.6
	gateCap          = 44
	bandGreatMin     = 80
	bandGoodMin      = 65
	bandFairMin      = 45
	favouriteBeta    = 0.4
	favouriteKey     = "company:favourite"
)

var pickStrength = map[string]float64{"nice": 1, "ok": 0.5}

type evaluatedPick struct {
	dimension dto.Dimension
	key       string
	label     string
	stance    string
	answer    dto.Answer
	known     bool
	retired   bool
	corrected bool
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

type niceDimension struct {
	sumYes, evidence float64
}

type gateDimension struct {
	picked, picksNo int
	pickedRows      []int
	unpickedYes     bool
}

func compute(picks, unpicked []evaluatedPick, salaryRaw string, floor *dto.Money, favourite bool) (int, string, []dto.ScoreRow) {
	if floor != nil {
		picks = append(picks, salaryPick(*floor, salaryRaw))
	}
	rows := make([]dto.ScoreRow, 0, len(picks))
	nice := make(map[dto.Dimension]*niceDimension)
	gates := make(map[dto.Dimension]*gateDimension)
	var avoidCost float64
	var blocked, gated bool

	for _, p := range picks {
		if p.retired {
			rows = append(rows, dto.ScoreRow{Key: p.key, Label: p.label, Stance: p.stance, Resolved: "retired", Effect: "retired"})
			continue
		}

		resolved := "unknown"
		if p.known {
			resolved = resolveAnswer(p.answer)
		}
		row := dto.ScoreRow{Key: p.key, Label: p.label, Stance: p.stance, Resolved: resolved, Corrected: p.corrected}

		switch p.stance {
		case "nice", "ok":
			d, ok := nice[p.dimension]
			if !ok {
				d = &niceDimension{}
				nice[p.dimension] = d
			}
			if p.known {
				d.sumYes += pickStrength[p.stance] * p.answer.PYes
				d.evidence = max(d.evidence, p.answer.PYes+p.answer.PNo)
			}
			switch resolved {
			case "yes":
				row.Effect = "meets"
			case "no":
				row.Effect = "misses"
			default:
				row.Effect = "unknown"
			}
			if dimensionSpecs[p.dimension].Gate {
				g, ok := gates[p.dimension]
				if !ok {
					g = &gateDimension{}
					gates[p.dimension] = g
				}
				g.picked++
				g.pickedRows = append(g.pickedRows, len(rows))
				if resolved == "no" {
					g.picksNo++
				}
			}
		case "avoid":
			if p.known {
				avoidCost += avoidWeight * p.answer.PYes
			}
			switch resolved {
			case "yes":
				row.Effect = "misses"
				if p.key == salaryKey {
					row.Effect = "gated"
					gated = true
				}
			case "no":
				row.Effect = "neutral"
			default:
				row.Effect = "unknown"
			}
		case "block":
			switch resolved {
			case "yes":
				row.Effect = "blocked"
				blocked = true
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

	for _, u := range unpicked {
		if g, ok := gates[u.dimension]; ok && u.known && resolveAnswer(u.answer) == "yes" {
			g.unpickedYes = true
		}
	}
	for _, g := range gates {
		if g.picksNo == g.picked && g.unpickedYes {
			gated = true
			for _, i := range g.pickedRows {
				rows[i].Effect = "gated"
			}
		}
	}

	var weighted, covered float64
	for dim, d := range nice {
		spec := dimensionSpecs[dim]
		credit := min(1, d.sumYes/float64(spec.Saturation))
		weighted += spec.Weight * d.evidence
		covered += spec.Weight * d.evidence * credit
	}

	score := int(math.Round(100 * (covered + 0.5*prior) / (weighted + avoidCost + prior)))
	if favourite && !blocked {
		score = favouriteLift(score)
		rows = append(rows, dto.ScoreRow{Key: favouriteKey, Label: "Favourite company", Resolved: "yes", Effect: "favourite"})
	}
	if gated {
		score = min(score, gateCap)
	}
	if blocked {
		score = 0
	}
	return score, bandFor(score), rows
}

func favouriteLift(score int) int {
	p := min(max(float64(score)/100, 0.01), 0.99)
	logit := math.Log(p/(1-p)) + favouriteBeta
	return max(score, int(math.Round(100/(1+math.Exp(-logit)))))
}

func bandFor(score int) string {
	switch {
	case score >= bandGreatMin:
		return "great"
	case score >= bandGoodMin:
		return "good"
	case score >= bandFairMin:
		return "fair"
	}
	return "poor"
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

// QuestionHash is the key option_answers caches a bank question's answer
// under; tests use it to seed and assert against that same key.
func QuestionHash(question string) string {
	sum := sha256.Sum256([]byte(question))
	return hex.EncodeToString(sum[:])
}

func pickedQuestionHashes(picks []dto.Pick, byID map[string]dto.ScoringOption) map[string]string {
	hashes := make(map[string]string, len(picks))
	gateDims := make(map[dto.Dimension]bool)
	for _, p := range picks {
		opt, ok := byID[p.OptionID]
		if !ok || opt.RetiredAt != nil {
			continue
		}
		hashes[QuestionHash(opt.Question)] = opt.Question
		gateDims[opt.Dimension] = dimensionSpecs[opt.Dimension].Gate
	}
	for _, opt := range byID {
		if opt.RetiredAt == nil && gateDims[opt.Dimension] {
			hashes[QuestionHash(opt.Question)] = opt.Question
		}
	}
	return hashes
}

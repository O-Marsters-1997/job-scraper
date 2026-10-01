package checks

import (
	"fmt"
	"slices"
	"strings"
)

const (
	checkParse = "parse"

	bulletKeyRunes = 24
)

// ParseInput is the draft PDF as a text extractor reads it.
type ParseInput struct {
	Lines    []string
	Headings []string
}

var foldText = strings.NewReplacer(
	"ﬀ", "ff", "ﬁ", "fi", "ﬂ", "fl", "ﬃ", "ffi", "ﬄ", "ffl", "ﬅ", "st", "ﬆ", "st",
	"•", "", "●", "", "▪", "", "◦", "", "■", "", "○", "", "·", "",
)

func foldLine(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(foldText.Replace(s))), " ")
}

// Parse warns when the draft's PDF layout would confuse a CV parser. A nil
// Parse emits nothing.
func Parse(d Draft) []Finding {
	if d.Parse == nil {
		return nil
	}
	lines := make([]string, len(d.Parse.Lines))
	for i, l := range d.Parse.Lines {
		lines[i] = foldLine(l)
	}

	var out []Finding
	if bulletsOutOfOrder(d, strings.Join(lines, " ")) {
		out = append(out, Finding{
			Check: checkParse, Severity: Info,
			Message: "bullets are read out of order, which usually means columns or a table; use a single-column layout",
		})
	}
	for _, h := range d.Parse.Headings {
		want := foldLine(h)
		if want != "" && !slices.Contains(lines, want) {
			out = append(out, Finding{
				Check: checkParse, Severity: Info,
				Message: fmt.Sprintf("the heading %q does not sit on its own line; avoid tables and side-by-side text around headings", h),
			})
		}
	}
	return out
}

// bulletsOutOfOrder compares only each bullet's opening words, since a bullet
// split across columns is interleaved with its neighbour after the first line.
func bulletsOutOfOrder(d Draft, text string) bool {
	last := -1
	for _, p := range d.Positions {
		for _, b := range p.Bullets {
			key := []rune(foldLine(b.Text))
			if len(key) == 0 {
				continue
			}
			key = key[:min(len(key), bulletKeyRunes)]
			at := strings.Index(text, string(key))
			if at < 0 {
				continue
			}
			if at < last {
				return true
			}
			last = at
		}
	}
	return false
}

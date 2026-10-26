// Package linefit counts the lines a CV bullet wraps to, matching headless
// Chromium laying the web editor's text out in the metric-compatible fonts.
package linefit

import (
	"embed"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

//go:embed fonts/*.ttf
var fontFS embed.FS

const pxPerPt = 4.0 / 3

// serifHint and sansHint mirror SERIF_HINT and SANS_HINT in frontend/src/lib/docFonts.ts.
var (
	serifHint = regexp.MustCompile(`(?i)serif|times|georgia|garamond|palatino|book|roman`)
	sansHint  = regexp.MustCompile(`(?i)sans`)
)

// Style is a bullet's dominant run style. Font is the Doc's family name
// (Calibri, Arial, Times New Roman, Cambria), mapped as frontend/src/lib/docFonts.ts does.
type Style struct {
	Font   string
	Size   float64
	Bold   bool
	Italic bool
}

type face struct {
	f    *sfnt.Font
	upem fixed.Int26_6
}

func loadFace(name string) face {
	data, err := fontFS.ReadFile("fonts/" + name + ".ttf")
	if err != nil {
		panic(fmt.Sprintf("read embedded font %s: %v", name, err))
	}
	f, err := sfnt.Parse(data)
	if err != nil {
		panic(fmt.Sprintf("parse embedded font %s: %v", name, err))
	}
	return face{f: f, upem: fixed.Int26_6(f.UnitsPerEm()) << 6}
}

func family(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "calibri", "carlito":
		return "carlito"
	case "arial", "arimo":
		return "arimo"
	case "times new roman", "tinos":
		return "tinos"
	case "cambria", "caladea":
		return "caladea"
	}
	if serifHint.MatchString(name) && !sansHint.MatchString(name) {
		return "tinos"
	}
	return "arimo"
}

// Width is the text width of a block: page width less side margins and indentStart.
func Width(p dto.LayoutPage, b dto.LayoutBlock) float64 {
	return p.Width - p.MarginLeft - p.MarginRight - b.IndentStart
}

// Lines counts the lines text wraps to at width pt. It floors the px font size
// to 1/64 px, as Chromium's 26.6 fixed point does.
func Lines(text string, s Style, width float64) int {
	weight, slant := "400", "normal"
	if s.Bold {
		weight = "700"
	}
	if s.Italic {
		slant = "italic"
	}
	f := loadFace(family(s.Font) + "-latin-" + weight + "-" + slant)
	sizePx := math.Floor(s.Size*pxPerPt*64) / 64
	m := measurer{face: f, scale: sizePx / float64(f.f.UnitsPerEm())}
	avail := width * pxPerPt

	lines, line := 1, ""
	for _, seg := range segments(text) {
		if m.width(line+seg) <= avail {
			line += seg
			continue
		}
		if line != "" {
			lines++
			line = ""
		}
		if m.width(seg) <= avail {
			line = seg
			continue
		}
		for _, r := range seg {
			if m.width(line+string(r)) > avail && line != "" {
				lines++
				line = ""
			}
			line += string(r)
		}
	}
	return lines
}

type measurer struct {
	face  face
	scale float64
	buf   sfnt.Buffer
}

// width is the advance sum plus GPOS kerning in px; trailing spaces hang
// and do not count.
func (m *measurer) width(s string) float64 {
	var total fixed.Int26_6
	var prev sfnt.GlyphIndex
	for i, r := range []rune(strings.TrimRight(s, " ")) {
		g, err := m.face.f.GlyphIndex(&m.buf, r)
		if err != nil {
			panic(fmt.Sprintf("glyph index %q: %v", r, err))
		}
		adv, err := m.face.f.GlyphAdvance(&m.buf, g, m.face.upem, font.HintingNone)
		if err != nil {
			panic(fmt.Sprintf("glyph advance %q: %v", r, err))
		}
		total += adv
		if i > 0 {
			k, err := m.face.f.Kern(&m.buf, prev, g, m.face.upem, font.HintingNone)
			if err != nil && !errors.Is(err, sfnt.ErrNotFound) {
				panic(fmt.Sprintf("kern: %v", err))
			}
			total += k
		}
		prev = g
	}
	return float64(total) / 64 * m.scale
}

// segments splits text after each space run and after a hyphen that follows
// a letter and precedes neither a digit nor a space: Chromium's break opportunities.
func segments(text string) []string {
	var out []string
	rs := []rune(text)
	start := 0
	for i, r := range rs {
		next := rune(0)
		if i+1 < len(rs) {
			next = rs[i+1]
		}
		brk := r == ' ' && next != ' ' ||
			r == '-' && i > 0 && unicode.IsLetter(rs[i-1]) && next != 0 && !unicode.IsDigit(next) && next != ' '
		if brk {
			out = append(out, string(rs[start:i+1]))
			start = i + 1
		}
	}
	if start < len(rs) {
		out = append(out, string(rs[start:]))
	}
	return out
}

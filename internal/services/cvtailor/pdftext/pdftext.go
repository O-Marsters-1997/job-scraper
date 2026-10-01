package pdftext

import (
	"bytes"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/ledongthuc/pdf"
)

// Lines returns the text of every page as rows in the order a PDF text
// extractor reads them, top to bottom. Text sharing a baseline is one line,
// so columns and table cells merge into the row they sit on.
func Lines(data []byte) (lines []string, err error) {
	defer func() {
		if r := recover(); r != nil {
			lines, err = nil, fmt.Errorf("malformed PDF: %v", r)
		}
	}()
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	for n := 1; n <= r.NumPage(); n++ {
		rows, err := r.Page(n).GetTextByRow()
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			parts := make([]string, len(row.Content))
			for i, t := range row.Content {
				parts[i] = t.S
			}
			if line := strings.Join(strings.Fields(strings.Join(parts, " ")), " "); line != "" {
				lines = append(lines, line)
			}
		}
	}
	return lines, nil
}

const glyphGap = 0.5

const sameLine = 2.0

// Run is one string of text drawn at a single position, size and font.
type Run struct {
	Page int
	X, Y float64
	Size float64
	Font string
	Text string
}

// Runs returns every text run of the PDF in reading order: page by page,
// top to bottom, then left to right.
func Runs(data []byte) (runs []Run, err error) {
	defer func() {
		if r := recover(); r != nil {
			runs, err = nil, fmt.Errorf("malformed PDF: %v", r)
		}
	}()
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	for n := 1; n <= r.NumPage(); n++ {
		start := len(runs)
		var prevEnd float64
		for _, t := range r.Page(n).Content().Text {
			if last := len(runs) - 1; last >= start {
				p := &runs[last]
				if p.Font == t.Font && p.Size == t.FontSize && p.Y == t.Y && math.Abs(t.X-prevEnd) < glyphGap {
					p.Text += t.S
					prevEnd = t.X + t.W
					continue
				}
			}
			runs = append(runs, Run{Page: n, X: t.X, Y: t.Y, Size: t.FontSize, Font: t.Font, Text: t.S})
			prevEnd = t.X + t.W
		}
		page := runs[start:]
		sort.SliceStable(page, func(i, j int) bool { return page[i].Y > page[j].Y })
		for lo := 0; lo < len(page); {
			hi := lo + 1
			for hi < len(page) && page[lo].Y-page[hi].Y <= sameLine {
				hi++
			}
			row := page[lo:hi]
			sort.SliceStable(row, func(i, j int) bool { return row[i].X < row[j].X })
			lo = hi
		}
	}
	return runs, nil
}

// Match reports whether a and b draw the same text in the same fonts, with
// every position and size within tol points. When they differ, diff names the
// page, line and text of the first mismatching run.
func Match(a, b []Run, tol float64) (ok bool, diff string) {
	line, lastY, lastPage := 0, 0.0, 0
	for i := 0; i < max(len(a), len(b)); i++ {
		var x, y Run
		ref := b[min(i, len(b)-1)]
		if i < len(a) {
			x, ref = a[i], a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if ref.Page != lastPage {
			line = 1
		} else if math.Abs(ref.Y-lastY) > tol {
			line++
		}
		lastPage, lastY = ref.Page, ref.Y
		where := fmt.Sprintf("page %d, line %d, %q", ref.Page, line, ref.Text)
		switch {
		case i >= len(a):
			return false, where + ": extra text only in the second PDF"
		case i >= len(b):
			return false, where + ": missing from the second PDF"
		case x.Page != y.Page || x.Text != y.Text:
			return false, fmt.Sprintf("%s: text reflowed, second PDF has %q on page %d", where, y.Text, y.Page)
		case x.Font != y.Font:
			return false, fmt.Sprintf("%s: font changed from %s to %s", where, x.Font, y.Font)
		case math.Abs(x.Size-y.Size) > tol:
			return false, fmt.Sprintf("%s: font size changed from %g to %g", where, x.Size, y.Size)
		case math.Abs(x.X-y.X) > tol || math.Abs(x.Y-y.Y) > tol:
			return false, fmt.Sprintf("%s: moved from (%.1f, %.1f) to (%.1f, %.1f)", where, x.X, x.Y, y.X, y.Y)
		}
	}
	return true, ""
}

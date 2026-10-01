package pdftext

import (
	"bytes"
	"fmt"
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

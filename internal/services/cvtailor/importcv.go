package cvtailor

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/docparse"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

var (
	dateToken = `(?:[A-Za-z]{3,9}\.?\s+\d{4}|\d{1,2}/\d{4}|\d{4})`
	dateRange = regexp.MustCompile(`(?i)\(?\s*(` + dateToken + `)\s*(?:-|–|—|to)\s*(present|current|now|` + dateToken + `)\s*\)?`)
	titleSeps = []string{" at ", " @ ", " | ", " – ", " — ", " - ", ", "}
	dateForms = []string{"January 2006", "Jan 2006", "Jan. 2006", "1/2006", "01/2006", "2006"}
)

type DocFetcher interface {
	GetDocument(ctx context.Context, userID, docID, tabID string) (json.RawMessage, error)
}

func (s *Service) PreviewImport(ctx context.Context, userID string, in dto.ImportPreviewInput) (dto.ImportPreview, error) {
	if in.DocID == "" || in.TabID == "" {
		return dto.ImportPreview{}, apperr.Invalid("docId and tabId are required")
	}
	raw, err := s.docs.GetDocument(ctx, userID, in.DocID, in.TabID)
	if err != nil {
		return dto.ImportPreview{}, err
	}
	ds, err := docparse.Parse(raw)
	if err != nil {
		return dto.ImportPreview{}, apperr.Unprocessable("could not read the CV tab")
	}
	existing, err := s.store.ListPositions(ctx, userID)
	if err != nil {
		return dto.ImportPreview{}, err
	}
	known := make(map[string]struct{}, len(existing))
	for _, p := range existing {
		known[strings.ToLower(p.Employer)] = struct{}{}
	}

	positions := positionsFromStructure(ds)
	for i := range positions {
		_, positions[i].EmployerExists = known[strings.ToLower(positions[i].Employer)]
	}
	return dto.ImportPreview{Positions: positions}, nil
}

func (s *Service) ImportPositions(ctx context.Context, userID string, in dto.ImportInput) ([]dto.Position, error) {
	if len(in.Positions) == 0 {
		return nil, apperr.Invalid("nothing to import")
	}
	positions := make([]dto.ImportPosition, len(in.Positions))
	for i, p := range in.Positions {
		valid, err := validatePosition(dto.PositionInput{
			Employer: p.Employer, Title: p.Title, StartDate: p.StartDate, EndDate: p.EndDate,
		})
		if err != nil {
			return nil, err
		}
		positions[i] = dto.ImportPosition{Employer: valid.Employer, Title: valid.Title, StartDate: valid.StartDate, EndDate: valid.EndDate}
		for _, text := range p.Achievements {
			if text = strings.TrimSpace(text); text != "" {
				positions[i].Achievements = append(positions[i].Achievements, text)
			}
		}
	}
	return s.store.ImportPositions(ctx, userID, positions)
}

func positionsFromStructure(ds docparse.DocStructure) []dto.ImportPosition {
	byHeading := map[int]*dto.ImportPosition{}
	var order []int
	for _, slot := range ds.Slots {
		if slot.HeadingIndex < 0 {
			continue
		}
		p, ok := byHeading[slot.HeadingIndex]
		if !ok {
			parsed := parseHeading(ds.Headings[slot.HeadingIndex].Text)
			p = &parsed
			byHeading[slot.HeadingIndex] = p
			order = append(order, slot.HeadingIndex)
		}
		p.Achievements = append(p.Achievements, slot.Text)
	}
	out := make([]dto.ImportPosition, 0, len(order))
	for _, idx := range order {
		out = append(out, *byHeading[idx])
	}
	return out
}

func parseHeading(text string) dto.ImportPosition {
	p := dto.ImportPosition{Achievements: []string{}}
	if m := dateRange.FindStringSubmatchIndex(text); m != nil {
		p.StartDate = parseDate(text[m[2]:m[3]])
		p.EndDate = parseDate(text[m[4]:m[5]])
		text = text[:m[0]] + " " + text[m[1]:]
	}
	text = strings.Trim(strings.TrimSpace(text), "-–—|,() ")
	for _, sep := range titleSeps {
		if title, employer, ok := strings.Cut(text, sep); ok {
			p.Title, p.Employer = strings.TrimSpace(title), strings.TrimSpace(employer)
			return p
		}
	}
	p.Title = text
	return p
}

func parseDate(s string) *string {
	s = strings.TrimSpace(s)
	for _, layout := range dateForms {
		if t, err := time.Parse(layout, s); err == nil {
			d := t.Format(time.DateOnly)
			return &d
		}
	}
	return nil
}

package cvtailor

import (
	"context"
	"strings"
	"unicode"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/docparse"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

func (s *Service) Headings(ctx context.Context, userID string, q dto.CVTabQuery) ([]dto.CVHeading, error) {
	positions, err := s.store.ListPositions(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.headings(ctx, userID, q, positions)
}

func (s *Service) headings(ctx context.Context, userID string, q dto.CVTabQuery, positions []dto.Position) ([]dto.CVHeading, error) {
	doc, err := loadTab(ctx, s.docs, userID, q.DocID, q.TabID)
	if err != nil {
		return nil, err
	}
	saved, err := s.store.ListHeadingMappings(ctx, userID, q.DocID, q.TabID)
	if err != nil {
		return nil, err
	}
	savedByText := make(map[string]*string, len(saved))
	for _, m := range saved {
		savedByText[m.HeadingText] = m.PositionID
	}

	out := []dto.CVHeading{}
	for _, h := range roleHeadings(doc) {
		heading := dto.CVHeading{Text: h.Text, SlotCount: h.slots}
		if pid, ok := savedByText[h.Text]; ok {
			heading.PositionID, heading.Confirmed = pid, true
		} else {
			heading.PositionID = autoMatch(h.Text, positions)
		}
		out = append(out, heading)
	}
	return out, nil
}

func (s *Service) SaveHeadings(ctx context.Context, userID string, in dto.HeadingMappingsInput) ([]dto.HeadingMapping, error) {
	positions, err := s.store.ListPositions(ctx, userID)
	if err != nil {
		return nil, err
	}
	owned := make(map[string]bool, len(positions))
	for _, p := range positions {
		owned[p.ID] = true
	}
	for i, m := range in.Mappings {
		m.HeadingText = strings.TrimSpace(m.HeadingText)
		in.Mappings[i] = m
		if m.HeadingText == "" {
			return nil, apperr.Invalid("headingText is required")
		}
		if m.PositionID != nil && !owned[*m.PositionID] {
			return nil, apperr.Invalid("unknown position")
		}
	}
	if err := s.store.SaveHeadingMappings(ctx, userID, in.DocID, in.TabID, in.Mappings); err != nil {
		return nil, err
	}
	return in.Mappings, nil
}

type roleHeading struct {
	docparse.Heading
	slots int
}

func roleHeadings(doc docparse.DocStructure) []roleHeading {
	counts := make(map[int]int, len(doc.Headings))
	for _, sl := range doc.Slots {
		counts[sl.HeadingIndex]++
	}
	var out []roleHeading
	for i, h := range doc.Headings {
		if n := counts[i]; n > 0 {
			out = append(out, roleHeading{Heading: h, slots: n})
		}
	}
	return out
}

func autoMatch(heading string, positions []dto.Position) *string {
	h := normalise(heading)
	var byEmployer []dto.Position
	for _, p := range positions {
		if e := normalise(p.Employer); e != "" && containsWords(h, e) {
			byEmployer = append(byEmployer, p)
		}
	}
	switch len(byEmployer) {
	case 0:
		return nil
	case 1:
		return &byEmployer[0].ID
	}
	var byTitle []dto.Position
	for _, p := range byEmployer {
		if t := normalise(p.Title); t != "" && containsWords(h, t) {
			byTitle = append(byTitle, p)
		}
	}
	if len(byTitle) == 1 {
		return &byTitle[0].ID
	}
	return nil
}

func normalise(s string) string {
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	return strings.Join(fields, " ")
}

func containsWords(haystack, needle string) bool {
	return strings.Contains(" "+haystack+" ", " "+needle+" ")
}

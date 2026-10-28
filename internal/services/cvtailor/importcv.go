package cvtailor

import (
	"context"
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/docparse"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

const dateExpr = `(?:[A-Za-z]{3,9}\.?\s+\d{4}|\d{1,2}/\d{4}|\d{4})`

var (
	dateRange = regexp.MustCompile(`(?i)\(?\s*(` + dateExpr + `)\s*(?:-|–|—|to)\s*(present|current|now|` + dateExpr + `)\s*\)?`)
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
	ds, err := loadTab(ctx, s.docs, userID, in.DocID, in.TabID)
	if err != nil {
		return dto.ImportPreview{}, err
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
	bank, err := s.store.ListBankSkills(ctx, userID)
	if err != nil {
		return dto.ImportPreview{}, err
	}
	return dto.ImportPreview{Positions: positions, Skills: skillsFromStructure(ds, bank)}, nil
}

func skillsFromStructure(ds docparse.DocStructure, bank []dto.BankSkill) []dto.ImportSkill {
	seen := make(map[string]struct{}, len(bank))
	inBank := make(map[string]bool, len(bank))
	for _, b := range bank {
		seen[normalizeText(b.Name)] = struct{}{}
		inBank[normalizeText(b.Name)] = true
	}
	out := []dto.ImportSkill{}
	if ds.Skills == nil {
		return out
	}
	for _, line := range ds.Skills.Lines {
		for _, item := range line.Items {
			name := strings.TrimSpace(item)
			key := normalizeText(name)
			if key == "" {
				continue
			}
			_, exists := seen[key]
			if exists && !inBank[key] {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, dto.ImportSkill{Name: name, Category: strings.TrimSpace(line.Label), Exists: exists})
		}
	}
	return out
}

func (s *Service) ImportPositions(ctx context.Context, userID string, in dto.ImportInput) ([]dto.Position, error) {
	if len(in.Positions) == 0 && len(in.Skills) == 0 {
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
	skills := make([]dto.BankSkillInput, len(in.Skills))
	for i, sk := range in.Skills {
		valid, err := validateBankSkill(dto.BankSkillInput{Name: sk.Name, Category: sk.Category})
		if err != nil {
			return nil, err
		}
		skills[i] = valid
	}
	if err := s.importBankSkills(ctx, userID, skills); err != nil {
		return nil, err
	}
	fresh, touched, err := s.addToExistingRoles(ctx, userID, positions)
	if err != nil {
		return nil, err
	}

	out := []dto.Position{}
	if len(fresh) > 0 {
		if out, err = s.store.ImportPositions(ctx, userID, fresh); err != nil {
			return nil, err
		}
	}
	if len(touched) == 0 {
		return out, nil
	}
	all, err := s.store.ListPositions(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, p := range all {
		if touched[p.ID] {
			out = append(out, p)
		}
	}
	return out, nil
}

func (s *Service) importBankSkills(ctx context.Context, userID string, skills []dto.BankSkillInput) error {
	if len(skills) == 0 {
		return nil
	}
	bank, err := s.store.ListBankSkills(ctx, userID)
	if err != nil {
		return err
	}
	have := make(map[string]struct{}, len(bank)+len(skills))
	for _, b := range bank {
		have[normalizeText(b.Name)] = struct{}{}
	}
	for _, sk := range skills {
		key := normalizeText(sk.Name)
		if _, ok := have[key]; ok {
			continue
		}
		if _, err := s.store.CreateBankSkill(ctx, userID, sk); err != nil {
			return err
		}
		have[key] = struct{}{}
	}
	return nil
}

func (s *Service) addToExistingRoles(ctx context.Context, userID string, positions []dto.ImportPosition) (fresh []dto.ImportPosition, touched map[string]bool, err error) {
	existing, err := s.store.ListPositions(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	idOfRole := make(map[string]string, len(existing))
	byID := make(map[string]dto.Position, len(existing))
	have := make(map[string]map[string]bool, len(existing))
	for _, p := range existing {
		idOfRole[roleKey(p.Employer, p.Title)] = p.ID
		byID[p.ID] = p
		have[p.ID] = make(map[string]bool, len(p.Achievements))
		for _, a := range p.Achievements {
			have[p.ID][normalizeText(a.Text)] = true
		}
	}

	touched = map[string]bool{}
	for _, p := range positions {
		id, ok := idOfRole[roleKey(p.Employer, p.Title)]
		if !ok {
			if id, ok = repairableRole(existing, p); ok {
				old := byID[id]
				if _, err := s.store.UpdatePosition(ctx, userID, dto.PositionInput{
					ID: id, Employer: p.Employer, Title: p.Title, StartDate: old.StartDate, EndDate: old.EndDate,
				}); err != nil {
					return nil, nil, err
				}
				idOfRole[roleKey(p.Employer, p.Title)] = id
				touched[id] = true
			}
		}
		if !ok {
			fresh = append(fresh, p)
			continue
		}
		for _, text := range p.Achievements {
			key := normalizeText(text)
			if have[id][key] {
				continue
			}
			if _, err := s.store.CreateAchievement(ctx, userID, dto.AchievementInput{PositionID: id, Text: text}); err != nil {
				return nil, nil, err
			}
			have[id][key] = true
			touched[id] = true
		}
	}
	return fresh, touched, nil
}

func repairableRole(existing []dto.Position, in dto.ImportPosition) (string, bool) {
	if in.StartDate == nil {
		return "", false
	}
	want := words(in.Employer + " " + in.Title)
	if len(want) == 0 {
		return "", false
	}
	var id string
	n := 0
	for _, p := range existing {
		if !sameDate(p.StartDate, in.StartDate) || !sameDate(p.EndDate, in.EndDate) {
			continue
		}
		have := words(p.Employer + " " + p.Title)
		if slices.ContainsFunc(want, func(w string) bool { return !slices.Contains(have, w) }) {
			continue
		}
		id = p.ID
		n++
	}
	return id, n == 1
}

func sameDate(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func words(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

func roleKey(employer, title string) string {
	return normalizeText(employer) + "|" + normalizeText(title)
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
	title, rest, ok := cutAtFirstSep(text)
	p.Title = title
	if ok {
		p.Employer, _, _ = cutAtFirstSep(rest)
	}
	return p
}

func cutAtFirstSep(text string) (before, after string, ok bool) {
	at, width := -1, 0
	for _, sep := range titleSeps {
		if i := strings.Index(text, sep); i >= 0 && (at < 0 || i < at) {
			at, width = i, len(sep)
		}
	}
	if at < 0 {
		return text, "", false
	}
	return strings.TrimSpace(text[:at]), strings.TrimSpace(text[at+width:]), true
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

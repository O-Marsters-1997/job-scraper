package docparse

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf16"
)

var (
	profileHeading = regexp.MustCompile(`(?i)\b(profile|summary|about)\b`)
	skillsHeading  = regexp.MustCompile(`(?i)\b(skills|technologies)\b`)
	skillSplit     = regexp.MustCompile(`\s*[,;|•·]\s*`)
	separators     = []string{" | ", " • ", " · ", "; ", ", "}
	emailRe        = regexp.MustCompile(`[\w.+-]+@[\w-]+(?:\.[\w-]+)+`)
	phoneRe        = regexp.MustCompile(`\+?\d[\d\s().-]{7,}\d`)
)

// Heading is a heading paragraph. Index is its start index in the Doc body.
type Heading struct {
	Text  string
	Level int
	Index int
}

// Slot is one editable paragraph. HeadingIndex points into DocStructure.Headings
// (-1 before the first heading); StartIndex and EndIndex are Doc body indices.
type Slot struct {
	ID           string
	HeadingIndex int
	Text         string
	StartIndex   int
	EndIndex     int
}

// SkillLine is one paragraph of the skills section, or a run of unlabelled
// list paragraphs. ItemsStart..End is the Doc range holding the items, after
// the label and before the paragraph's newline.
type SkillLine struct {
	Label      string
	Items      []string
	Separator  string
	ItemsStart int
	End        int
}

// SkillsSlot is the skills section. It spans StartIndex to EndIndex and, for a
// prose layout, records the Separator the items were joined with. Items is the
// concatenation of every line's Items.
type SkillsSlot struct {
	Lines      []SkillLine
	Items      []string
	List       bool
	Separator  string
	StartIndex int
	EndIndex   int
}

// Contact records where an email address or phone number appears in the Tab.
type Contact struct {
	InBody         bool
	InHeaderFooter bool
}

// DocStructure is the parsed shape of one Docs Tab. Profile and Skills are nil
// when the CV has no such section.
type DocStructure struct {
	Contact  Contact
	Headings []Heading
	Slots    []Slot
	Profile  *Slot
	Skills   *SkillsSlot
}

type tab struct {
	DocumentTab struct {
		Body struct {
			Content []element `json:"content"`
		} `json:"body"`
		Headers map[string]segment `json:"headers"`
		Footers map[string]segment `json:"footers"`
	} `json:"documentTab"`
}

type segment struct {
	Content []element `json:"content"`
}

type element struct {
	StartIndex int        `json:"startIndex"`
	EndIndex   int        `json:"endIndex"`
	Paragraph  *paragraph `json:"paragraph"`
	Table      *struct {
		TableRows []struct {
			TableCells []struct {
				Content []element `json:"content"`
			} `json:"tableCells"`
		} `json:"tableRows"`
	} `json:"table"`
}

type paragraph struct {
	Elements []struct {
		TextRun *struct {
			Content string `json:"content"`
		} `json:"textRun"`
	} `json:"elements"`
	ParagraphStyle struct {
		NamedStyleType string `json:"namedStyleType"`
	} `json:"paragraphStyle"`
	Bullet *json.RawMessage `json:"bullet"`
}

type para struct {
	text       string
	lead       int
	level      int
	list       bool
	start, end int
}

// Parse turns the JSON of one Docs API Tab (a Tab object with documentTab.body)
// into a DocStructure.
func Parse(tabJSON []byte) (DocStructure, error) {
	var t tab
	if err := json.Unmarshal(tabJSON, &t); err != nil {
		return DocStructure{}, fmt.Errorf("docparse.Parse: %w", err)
	}
	var paras []para
	flatten(t.DocumentTab.Body.Content, &paras)
	ds := build(paras)
	ds.Contact = Contact{InBody: hasContact(paras), InHeaderFooter: hasContact(segmentParas(t))}
	return ds, nil
}

func segmentParas(t tab) []para {
	var paras []para
	for _, segs := range []map[string]segment{t.DocumentTab.Headers, t.DocumentTab.Footers} {
		for _, seg := range segs {
			flatten(seg.Content, &paras)
		}
	}
	return paras
}

func hasContact(paras []para) bool {
	return slices.ContainsFunc(paras, func(p para) bool {
		return emailRe.MatchString(p.text) || slices.ContainsFunc(phoneRe.FindAllString(p.text, -1), isPhone)
	})
}

func isPhone(s string) bool {
	digits := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			digits++
		}
	}
	return digits >= 9
}

func flatten(content []element, out *[]para) {
	for _, el := range content {
		switch {
		case el.Paragraph != nil:
			var sb strings.Builder
			for _, pe := range el.Paragraph.Elements {
				if pe.TextRun != nil {
					sb.WriteString(pe.TextRun.Content)
				}
			}
			raw := sb.String()
			*out = append(*out, para{
				text:  strings.TrimSpace(raw),
				lead:  utf16Len(raw[:len(raw)-len(strings.TrimLeftFunc(raw, unicode.IsSpace))]),
				level: headingLevel(el.Paragraph.ParagraphStyle.NamedStyleType),
				list:  el.Paragraph.Bullet != nil,
				start: el.StartIndex,
				end:   el.EndIndex,
			})
		case el.Table != nil:
			for _, row := range el.Table.TableRows {
				for _, cell := range row.TableCells {
					flatten(cell.Content, out)
				}
			}
		}
	}
}

func headingLevel(style string) int {
	var n int
	if _, err := fmt.Sscanf(style, "HEADING_%d", &n); err != nil {
		return 0
	}
	return n
}

type section int

const (
	sectionRole section = iota
	sectionProfile
	sectionSkills
)

func build(paras []para) DocStructure {
	var ds DocStructure
	current := sectionRole
	headingIdx := -1
	var skillParas []para

	for _, p := range paras {
		if p.text == "" {
			continue
		}
		if p.level > 0 {
			ds.Headings = append(ds.Headings, Heading{Text: p.text, Level: p.level, Index: p.start})
			headingIdx = len(ds.Headings) - 1
			switch {
			case ds.Profile == nil && profileHeading.MatchString(p.text):
				current = sectionProfile
			case ds.Skills == nil && skillsHeading.MatchString(p.text):
				current = sectionSkills
			default:
				current = sectionRole
			}
			continue
		}
		switch current {
		case sectionProfile:
			if ds.Profile == nil && !p.list {
				ds.Profile = &Slot{ID: "profile", HeadingIndex: headingIdx, Text: p.text, StartIndex: p.start, EndIndex: p.end}
			}
		case sectionSkills:
			skillParas = append(skillParas, p)
		case sectionRole:
			if p.list {
				ds.Slots = append(ds.Slots, Slot{
					ID:           fmt.Sprintf("s%d", len(ds.Slots)),
					HeadingIndex: headingIdx,
					Text:         p.text,
					StartIndex:   p.start,
					EndIndex:     p.end,
				})
			}
		}
	}
	ds.Skills = buildSkills(skillParas)
	return ds
}

func buildSkills(paras []para) *SkillsSlot {
	if len(paras) == 0 {
		return nil
	}
	s := &SkillsSlot{
		List:       paras[0].list,
		StartIndex: paras[0].start,
		EndIndex:   paras[len(paras)-1].end,
	}
	if !s.List {
		for _, p := range paras {
			if s.Separator = detectSeparator(p.text); s.Separator != "" {
				break
			}
		}
	}
	for i := 0; i < len(paras); i++ {
		if !isPlainListItem(paras[i]) {
			s.Lines = append(s.Lines, proseLine(paras[i]))
			continue
		}
		run := []para{paras[i]}
		for i+1 < len(paras) && isPlainListItem(paras[i+1]) {
			i++
			run = append(run, paras[i])
		}
		s.Lines = append(s.Lines, listLine(run))
	}
	for _, l := range s.Lines {
		s.Items = append(s.Items, l.Items...)
	}
	return s
}

func isPlainListItem(p para) bool {
	_, _, labelled := splitLabel(p.text)
	return p.list && !labelled
}

func listLine(run []para) SkillLine {
	l := SkillLine{Separator: "\n", ItemsStart: run[0].start + run[0].lead, End: run[len(run)-1].end - 1}
	for _, p := range run {
		l.Items = append(l.Items, p.text)
	}
	return l
}

func proseLine(p para) SkillLine {
	label, body, labelled := splitLabel(p.text)
	l := SkillLine{Label: label, Separator: detectSeparator(body), ItemsStart: p.start + p.lead, End: p.end - 1}
	if labelled {
		l.ItemsStart += utf16Len(p.text[:len(p.text)-len(body)])
	}
	for _, item := range skillSplit.Split(body, -1) {
		if item != "" {
			l.Items = append(l.Items, item)
		}
	}
	return l
}

func splitLabel(text string) (label, body string, labelled bool) {
	before, after, found := strings.Cut(text, ":")
	label = strings.TrimSpace(before)
	if !found || label == "" || skillSplit.MatchString(label) {
		return "", text, false
	}
	return label, strings.TrimLeftFunc(after, unicode.IsSpace), true
}

func utf16Len(s string) int {
	return len(utf16.Encode([]rune(s)))
}

func detectSeparator(text string) string {
	for _, sep := range separators {
		if strings.Contains(text, sep) {
			return sep
		}
	}
	return ""
}

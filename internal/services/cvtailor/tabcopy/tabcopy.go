package tabcopy

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

var ErrUnsupported = errors.New("tabcopy: tab has content that can't be rebuilt")

type tab struct {
	DocumentTab struct {
		Body struct {
			Content []element `json:"content"`
		} `json:"body"`
		DocumentStyle docStyle `json:"documentStyle"`
		NamedStyles   struct {
			Styles []namedStyle `json:"styles"`
		} `json:"namedStyles"`
		Lists   map[string]list            `json:"lists"`
		Headers map[string]json.RawMessage `json:"headers"`
		Footers map[string]json.RawMessage `json:"footers"`
	} `json:"documentTab"`
}

type element struct {
	StartIndex   int              `json:"startIndex"`
	EndIndex     int              `json:"endIndex"`
	Paragraph    *paragraph       `json:"paragraph"`
	SectionBreak *json.RawMessage `json:"sectionBreak"`
}

type paragraph struct {
	Elements       []paragraphElement `json:"elements"`
	ParagraphStyle style              `json:"paragraphStyle"`
	Bullet         *struct {
		ListID       string `json:"listId"`
		NestingLevel int    `json:"nestingLevel"`
	} `json:"bullet"`
}

type paragraphElement struct {
	StartIndex int `json:"startIndex"`
	EndIndex   int `json:"endIndex"`
	TextRun    *struct {
		Content   string `json:"content"`
		TextStyle style  `json:"textStyle"`
	} `json:"textRun"`
}

type namedStyle struct {
	NamedStyleType string `json:"namedStyleType"`
	TextStyle      style  `json:"textStyle"`
	ParagraphStyle style  `json:"paragraphStyle"`
}

type list struct {
	ListProperties struct {
		NestingLevels []nestingLevel `json:"nestingLevels"`
	} `json:"listProperties"`
}

type docStyle struct {
	PageSize     json.RawMessage `json:"pageSize"`
	MarginTop    json.RawMessage `json:"marginTop"`
	MarginBottom json.RawMessage `json:"marginBottom"`
	MarginLeft   json.RawMessage `json:"marginLeft"`
	MarginRight  json.RawMessage `json:"marginRight"`
}

type nestingLevel struct {
	GlyphType       string          `json:"glyphType"`
	IndentFirstLine json.RawMessage `json:"indentFirstLine"`
	IndentStart     json.RawMessage `json:"indentStart"`
}

type style map[string]json.RawMessage

const (
	normalText   = "NORMAL_TEXT"
	bulletPreset = "BULLET_DISC_CIRCLE_SQUARE"
	numberPreset = "NUMBERED_DECIMAL_ALPHA_ROMAN"
	glyphDecimal = "DECIMAL"
	maxColumns   = 1
)

// Requests returns the batchUpdate requests that rebuild src's body in the
// Tab tabID with every style explicit: the Docs API can neither copy a Tab nor
// set its named styles.
func Requests(src json.RawMessage, tabID string) ([]json.RawMessage, error) {
	var t tab
	if err := json.Unmarshal(src, &t); err != nil {
		return nil, fmt.Errorf("decoding tab: %w", err)
	}
	dt := t.DocumentTab
	if len(dt.Headers) > 0 || len(dt.Footers) > 0 {
		return nil, fmt.Errorf("%w: headers or footers", ErrUnsupported)
	}

	named := map[string]namedStyle{}
	for _, s := range dt.NamedStyles.Styles {
		named[s.NamedStyleType] = s
	}

	var text strings.Builder
	var bullets, paragraphs, runs []json.RawMessage
	var run bulletRun

	for _, el := range dt.Body.Content {
		switch {
		case el.SectionBreak != nil:
			var sb struct {
				SectionStyle struct {
					ColumnProperties []json.RawMessage `json:"columnProperties"`
				} `json:"sectionStyle"`
			}
			if err := json.Unmarshal(*el.SectionBreak, &sb); err != nil {
				return nil, fmt.Errorf("decoding section break: %w", err)
			}
			if el.StartIndex != 0 {
				return nil, fmt.Errorf("%w: section break at index %d", ErrUnsupported, el.StartIndex)
			}
			if len(sb.SectionStyle.ColumnProperties) > maxColumns {
				return nil, fmt.Errorf("%w: multiple columns", ErrUnsupported)
			}
			continue
		case el.Paragraph == nil:
			return nil, fmt.Errorf("%w: non-paragraph content at index %d", ErrUnsupported, el.StartIndex)
		}
		p := el.Paragraph

		for _, pe := range p.Elements {
			if pe.TextRun == nil {
				return nil, fmt.Errorf("%w: inline element at index %d", ErrUnsupported, pe.StartIndex)
			}
			text.WriteString(pe.TextRun.Content)
		}

		namedType := p.ParagraphStyle.namedType()
		pstyle := merge(named[normalText].ParagraphStyle, named[namedType].ParagraphStyle)
		if p.Bullet != nil {
			lvl := levelOf(dt.Lists, p.Bullet.ListID, p.Bullet.NestingLevel)
			pstyle = merge(pstyle, style{"indentStart": lvl.IndentStart, "indentFirstLine": lvl.IndentFirstLine})
			if p.Bullet.ListID == run.listID && el.StartIndex == run.end {
				run.end = el.EndIndex
			} else {
				bullets = run.appendRequest(bullets, tabID, dt.Lists)
				run = bulletRun{listID: p.Bullet.ListID, start: el.StartIndex, end: el.EndIndex}
			}
		}
		pstyle = merge(pstyle, p.ParagraphStyle)
		delete(pstyle, "namedStyleType")
		delete(pstyle, "headingId")
		if r := updateStyle("updateParagraphStyle", "paragraphStyle", pstyle, el.StartIndex, el.EndIndex, tabID); r != nil {
			paragraphs = append(paragraphs, r)
		}

		for _, pe := range p.Elements {
			tstyle := merge(named[normalText].TextStyle, named[namedType].TextStyle, pe.TextRun.TextStyle)
			if r := updateStyle("updateTextStyle", "textStyle", tstyle, pe.StartIndex, pe.EndIndex, tabID); r != nil {
				runs = append(runs, r)
			}
		}
	}
	bullets = run.appendRequest(bullets, tabID, dt.Lists)

	// The target Tab already ends in a newline, so the source's final one is not inserted.
	body := strings.TrimSuffix(text.String(), "\n")
	out := []json.RawMessage{documentStyle(dt.DocumentStyle, tabID)}
	if body != "" {
		out = append(out, mustRequest("insertText", map[string]any{
			"location": map[string]any{"index": 1, "tabId": tabID},
			"text":     body,
		}))
	}
	out = append(out, bullets...)
	out = append(out, paragraphs...)
	out = append(out, runs...)
	return out, nil
}

func (s style) namedType() string {
	var n string
	if raw, ok := s["namedStyleType"]; ok {
		_ = json.Unmarshal(raw, &n)
	}
	if n == "" {
		return normalText
	}
	return n
}

func levelOf(ls map[string]list, id string, level int) nestingLevel {
	levels := ls[id].ListProperties.NestingLevels
	if level < len(levels) {
		return levels[level]
	}
	return nestingLevel{}
}

type bulletRun struct {
	listID     string
	start, end int
}

func (r bulletRun) appendRequest(reqs []json.RawMessage, tabID string, ls map[string]list) []json.RawMessage {
	if r.listID == "" {
		return reqs
	}
	preset := bulletPreset
	if lvls := ls[r.listID].ListProperties.NestingLevels; len(lvls) > 0 && lvls[0].GlyphType == glyphDecimal {
		preset = numberPreset
	}
	return append(reqs, mustRequest("createParagraphBullets", map[string]any{
		"range":        map[string]any{"startIndex": r.start, "endIndex": r.end, "tabId": tabID},
		"bulletPreset": preset,
	}))
}

func documentStyle(ds docStyle, tabID string) json.RawMessage {
	s := merge(style{"pageSize": ds.PageSize, "marginTop": ds.MarginTop, "marginBottom": ds.MarginBottom,
		"marginLeft": ds.MarginLeft, "marginRight": ds.MarginRight})
	return mustRequest("updateDocumentStyle", map[string]any{
		"documentStyle": s,
		"fields":        fields(s),
		"tabId":         tabID,
	})
}

func updateStyle(kind, key string, s style, start, end int, tabID string) json.RawMessage {
	if len(s) == 0 {
		return nil
	}
	return mustRequest(kind, map[string]any{
		"range":  map[string]any{"startIndex": start, "endIndex": end, "tabId": tabID},
		key:      s,
		"fields": fields(s),
	})
}

func merge(layers ...style) style {
	out := style{}
	for _, l := range layers {
		for k, v := range l {
			if len(v) > 0 {
				out[k] = v
			}
		}
	}
	return out
}

func fields(s style) string {
	keys := make([]string, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return strings.Join(keys, ",")
}

func mustRequest(kind string, body map[string]any) json.RawMessage {
	b, err := json.Marshal(map[string]any{kind: body})
	if err != nil {
		panic(fmt.Sprintf("tabcopy: marshalling %s: %v", kind, err))
	}
	return b
}

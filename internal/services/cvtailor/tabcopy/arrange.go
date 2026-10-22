package tabcopy

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

var ErrAmbiguousRole = errors.New("tabcopy: a position is mapped to more than one heading")

type object map[string]json.RawMessage

type block struct {
	from, to int
	position string
}

type bodyParagraph struct {
	heading bool
	bullet  bool
	text    string
}

// Arrange returns src with its role blocks (a heading through the paragraph before the next
// one) in the order of order, a list of position ids; roleOf maps heading text to position id.
// Blocks of positions left out are dropped, and unmapped blocks keep their place.
func Arrange(src json.RawMessage, roleOf map[string]string, order []string, roles map[string]Role) (json.RawMessage, map[string]string, error) {
	var tab, doc, body object
	var content []object
	if err := json.Unmarshal(src, &tab); err != nil {
		return nil, nil, fmt.Errorf("decoding tab: %w", err)
	}
	if err := json.Unmarshal(tab["documentTab"], &doc); err != nil {
		return nil, nil, fmt.Errorf("decoding tab: %w", err)
	}
	if err := json.Unmarshal(doc["body"], &body); err != nil {
		return nil, nil, fmt.Errorf("decoding tab: %w", err)
	}
	if err := json.Unmarshal(body["content"], &content); err != nil {
		return nil, nil, fmt.Errorf("decoding tab: %w", err)
	}

	lead := 0
	for lead < len(content) && content[lead]["paragraph"] == nil {
		lead++
	}
	paras := make([]bodyParagraph, 0, len(content)-lead)
	for _, el := range content[lead:] {
		if el["paragraph"] == nil {
			return nil, nil, fmt.Errorf("%w: non-paragraph content", ErrUnsupported)
		}
		p, err := paragraphText(el["paragraph"])
		if err != nil {
			return nil, nil, err
		}
		paras = append(paras, p)
	}

	blocks := blocksOf(paras, roleOf)
	byPosition := map[string]block{}
	tpl := -1
	for i, bl := range blocks {
		if bl.position == "" {
			continue
		}
		if _, dup := byPosition[bl.position]; dup {
			return nil, nil, fmt.Errorf("%w: %q", ErrAmbiguousRole, bl.position)
		}
		byPosition[bl.position] = bl
		if tpl < 0 {
			tpl = i
		}
	}
	fresh := 0
	for _, id := range order {
		if _, ok := byPosition[id]; !ok {
			fresh++
		}
	}
	if fresh > 0 {
		if tpl < 0 {
			return nil, nil, fmt.Errorf("%w: no role to copy", ErrUnsupported)
		}
		last := len(blocks) - 1
		for blocks[last].position == "" {
			last--
		}
		blocks = slices.Insert(blocks, last+1, slices.Repeat([]block{{position: "+"}}, fresh)...)
	}

	out := append([]object{}, content[:lead]...)
	queue := order
	next := 0
	if lead < len(content) {
		next = intOf(content[lead]["startIndex"])
	}
	headings := map[string]string{}
	for _, slot := range blocks {
		bl := slot
		if slot.position != "" {
			if len(queue) == 0 {
				continue
			}
			var ok bool
			if bl, ok = byPosition[queue[0]]; !ok {
				mapped := mappedAt(paras, blocks[tpl], roleOf)
				to, known := roles[queue[0]]
				if !known {
					return nil, nil, fmt.Errorf("tabcopy: no block or role for position %q", queue[0])
				}
				copied, heading, end, err := cloneRole(content[lead:], paras, blocks[tpl], mapped, roles[roleOf[paras[mapped].text]], to, next)
				if err != nil {
					return nil, nil, err
				}
				out, next, headings[queue[0]] = append(out, copied...), end, heading
				queue = queue[1:]
				continue
			}
			queue = queue[1:]
		}
		for _, el := range content[lead+bl.from : lead+bl.to] {
			by := next - intOf(el["startIndex"])
			next = intOf(el["endIndex"]) + by
			out = append(out, shifted(el, by))
		}
	}

	body["content"] = mustMarshal(out)
	doc["body"] = mustMarshal(body)
	tab["documentTab"] = mustMarshal(doc)
	arranged, err := json.Marshal(tab)
	return arranged, headings, err
}

func mappedAt(paras []bodyParagraph, bl block, roleOf map[string]string) int {
	for i := bl.from; i < bl.to; i++ {
		if _, ok := roleOf[paras[i].text]; ok && paras[i].heading {
			return i
		}
	}
	return bl.from
}

func paragraphText(raw json.RawMessage) (bodyParagraph, error) {
	var p struct {
		Elements []struct {
			TextRun *struct{ Content string } `json:"textRun"`
		} `json:"elements"`
		ParagraphStyle struct {
			NamedStyleType string `json:"namedStyleType"`
		} `json:"paragraphStyle"`
		Bullet *struct{} `json:"bullet"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return bodyParagraph{}, fmt.Errorf("decoding paragraph: %w", err)
	}
	var text strings.Builder
	for _, pe := range p.Elements {
		if pe.TextRun != nil {
			text.WriteString(pe.TextRun.Content)
		}
	}
	return bodyParagraph{
		heading: strings.HasPrefix(p.ParagraphStyle.NamedStyleType, "HEADING_"),
		bullet:  p.Bullet != nil,
		text:    strings.TrimSpace(text.String()),
	}, nil
}

func blocksOf(paras []bodyParagraph, roleOf map[string]string) []block {
	var blocks []block
	prevHeading := false
	for i, p := range paras {
		if p.text == "" {
			continue
		}
		if p.heading && !prevHeading || len(blocks) == 0 {
			blocks = append(blocks, block{from: i})
		}
		prevHeading = p.heading
		if pos, ok := roleOf[p.text]; ok && p.heading {
			blocks[len(blocks)-1].position = pos
		}
	}
	for i := range blocks {
		blocks[i].to = len(paras)
		if i+1 < len(blocks) {
			blocks[i].to = blocks[i+1].from
		}
	}
	if len(blocks) > 0 {
		blocks[0].from = 0
	}
	return blocks
}

// Clear returns the requests that empty the Tab described by cur.
func Clear(cur json.RawMessage, tabID string) ([]json.RawMessage, error) {
	dt, _, err := decode(cur)
	if err != nil {
		return nil, err
	}
	end := 1
	if n := len(dt.Body.Content); n > 0 {
		end = dt.Body.Content[n-1].EndIndex
	}
	var out []json.RawMessage
	if end-1 > 1 {
		out = append(out, mustRequest("deleteContentRange", map[string]any{
			"range": map[string]any{"startIndex": 1, "endIndex": end - 1, "tabId": tabID},
		}))
	}
	return append(out, mustRequest("deleteParagraphBullets", map[string]any{
		"range": map[string]any{"startIndex": 1, "endIndex": 2, "tabId": tabID},
	})), nil
}

func shifted(el object, by int) object {
	out := maps.Clone(el)
	shiftIndices(out, by)
	var p object
	var elems []object
	if json.Unmarshal(el["paragraph"], &p) != nil || json.Unmarshal(p["elements"], &elems) != nil {
		return out
	}
	moved := make([]object, len(elems))
	for i, pe := range elems {
		moved[i] = maps.Clone(pe)
		shiftIndices(moved[i], by)
	}
	p["elements"] = mustMarshal(moved)
	out["paragraph"] = mustMarshal(p)
	return out
}

func shiftIndices(o object, by int) {
	for _, k := range []string{"startIndex", "endIndex"} {
		if raw, ok := o[k]; ok {
			o[k] = mustMarshal(intOf(raw) + by)
		}
	}
}

func intOf(raw json.RawMessage) int {
	var n int
	_ = json.Unmarshal(raw, &n)
	return n
}

func mustMarshal(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("tabcopy: marshalling: %v", err))
	}
	return b
}

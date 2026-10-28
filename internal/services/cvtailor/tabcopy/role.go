package tabcopy

import (
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"strings"
	"time"
	"unicode/utf16"
)

type Role struct {
	Title, Employer string
	Start, End      *string
}

const monthPattern = `(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\.? +`

var (
	dateRange   = regexp.MustCompile(`((?:` + monthPattern + `)?\d{4})( *[–—-] *)((?:` + monthPattern + `)?\d{4}|[A-Za-z]+)`)
	leadingWord = regexp.MustCompile(`^[A-Za-z]+`)
)

func dateLike(sample string, iso *string, present string) string {
	if iso == nil || len(*iso) < 10 {
		return present
	}
	d, err := time.Parse("2006-01-02", (*iso)[:10])
	if err != nil {
		return present
	}
	switch month := leadingWord.FindString(sample); {
	case month == "":
		return fmt.Sprint(d.Year())
	case len(month) > 3:
		return fmt.Sprintf("%s %d", d.Month(), d.Year())
	default:
		return fmt.Sprintf("%s %d", d.Month().String()[:3], d.Year())
	}
}

type edit struct {
	from, to int
	text     string
}

func dateEdit(text string, role Role) (edit, bool) {
	loc := dateRange.FindStringSubmatchIndex(text)
	if loc == nil {
		return edit{}, false
	}
	start, sep, end := text[loc[2]:loc[3]], text[loc[4]:loc[5]], text[loc[6]:loc[7]]
	present, endSample := end, end
	if strings.ContainsAny(end, "0123456789") {
		present = "Present"
	} else {
		endSample = start
	}
	return edit{loc[0], loc[1], dateLike(start, role.Start, present) + sep + dateLike(endSample, role.End, present)}, true
}

func restyleDates(text string, to Role) string {
	e, ok := dateEdit(text, to)
	if !ok {
		return text
	}
	return text[:e.from] + e.text + text[e.to:]
}

func fieldEnd(s string, from int) int {
	if i := strings.IndexAny(s[from:], "|,\t\n"); i >= 0 {
		return from + i
	}
	return len(s)
}

func trimmedEnd(s string, from, to int) int {
	return from + len(strings.TrimRight(s[from:to], " "))
}

func fieldEdits(whole string, to Role) []edit {
	sepAt := fieldEnd(whole, 0)
	titleEnd := trimmedEnd(whole, 0, sepAt)
	tailFrom := titleEnd
	edits := []edit{{0, titleEnd, to.Title + ", " + to.Employer}}
	if sepAt < len(whole) && (whole[sepAt] == '|' || whole[sepAt] == ',') {
		start := sepAt + 1
		for start < len(whole) && whole[start] == ' ' {
			start++
		}
		end := trimmedEnd(whole, start, fieldEnd(whole, start))
		edits = []edit{{0, titleEnd, to.Title}, {start, end, to.Employer}}
		tailFrom = end
	}
	if d, ok := dateEdit(whole[tailFrom:], to); ok {
		d.from, d.to = d.from+tailFrom, d.to+tailFrom
		edits = append(edits, d)
	}
	return edits
}

func applyEdits(texts []string, edits []edit) []string {
	total := 0
	for _, t := range texts {
		total += len(t)
	}
	out := make([]string, len(texts))
	runStart := 0
	for i, t := range texts {
		runEnd := runStart + len(t)
		var b strings.Builder
		cursor := runStart
		for _, e := range edits {
			from, to := min(max(e.from, runStart), runEnd), min(max(e.to, runStart), runEnd)
			owns := runStart <= e.from && (e.from < runEnd || e.from == total && i == len(texts)-1)
			if from >= to && !owns {
				continue
			}
			b.WriteString(t[cursor-runStart : max(cursor, from)-runStart])
			if owns {
				b.WriteString(e.text)
			}
			cursor = max(cursor, to)
		}
		b.WriteString(t[cursor-runStart:])
		out[i] = b.String()
		runStart = runEnd
	}
	return out
}

func utf16Len(s string) int { return len(utf16.Encode([]rune(s))) }

func contentOf(r object) string {
	var tr struct{ Content string }
	_ = json.Unmarshal(r["textRun"], &tr)
	return tr.Content
}

func withContent(r object, content string) object {
	var tr object
	_ = json.Unmarshal(r["textRun"], &tr)
	tr["content"] = mustMarshal(content)
	out := maps.Clone(r)
	out["textRun"] = mustMarshal(tr)
	return out
}

func mapContent(runs []object, f func(string) string) []object {
	out := make([]object, len(runs))
	for i, r := range runs {
		out[i] = withContent(r, f(contentOf(r)))
	}
	return out
}

func rewrite(el object, start int, f func([]object) []object) (object, int, error) {
	var p object
	var runs []object
	if err := json.Unmarshal(el["paragraph"], &p); err != nil {
		return nil, 0, fmt.Errorf("decoding paragraph: %w", err)
	}
	if err := json.Unmarshal(p["elements"], &runs); err != nil {
		return nil, 0, fmt.Errorf("decoding paragraph: %w", err)
	}
	runs = f(runs)
	at := start
	for _, r := range runs {
		r["startIndex"] = mustMarshal(at)
		at += utf16Len(contentOf(r))
		r["endIndex"] = mustMarshal(at)
	}
	p["elements"] = mustMarshal(runs)
	return object{"startIndex": mustMarshal(start), "endIndex": mustMarshal(at), "paragraph": mustMarshal(p)}, at, nil
}

func swap(s, from, to string) string {
	if from == "" {
		return s
	}
	return strings.ReplaceAll(s, from, to)
}

func restyle(runs []object, named, last bool, from, to Role) []object {
	switch {
	case named:
		return mapContent(runs, func(s string) string {
			return restyleDates(swap(swap(s, from.Title, to.Title), from.Employer, to.Employer), to)
		})
	case !last:
		return mapContent(runs[:1], func(string) string { return to.Title + "\n" })
	}
	texts := make([]string, len(runs))
	for i, r := range runs {
		texts[i] = contentOf(r)
	}
	edited := applyEdits(texts, fieldEdits(strings.Join(texts, ""), to))
	out := make([]object, len(runs))
	for i, r := range runs {
		out[i] = withContent(r, edited[i])
	}
	return out
}

func cloneRole(content []object, paras []bodyParagraph, tpl block, mapped int, from, to Role, next int) ([]object, string, int, error) {
	first := mapped
	for first > tpl.from && paras[first-1].heading && paras[first-1].text != "" {
		first--
	}
	bullet := -1
	for i := mapped + 1; i < tpl.to && bullet < 0; i++ {
		if paras[i].bullet {
			bullet = i
		}
	}
	if bullet < 0 {
		return nil, "", 0, fmt.Errorf("%w: no bullet to copy", ErrUnsupported)
	}
	var lines []string
	for _, p := range paras[first : mapped+1] {
		lines = append(lines, p.text)
	}
	joined := strings.Join(lines, "\n")
	named := from.Title != "" && strings.Contains(joined, from.Title) || from.Employer != "" && strings.Contains(joined, from.Employer)

	var out []object
	for i := first; i <= mapped; i++ {
		el, end, err := rewrite(content[i], next, func(runs []object) []object {
			return restyle(runs, named, i == mapped, from, to)
		})
		if err != nil {
			return nil, "", 0, err
		}
		next = end
		out = append(out, el)
	}
	heading, err := paragraphText(out[len(out)-1]["paragraph"])
	if err != nil {
		return nil, "", 0, err
	}
	el, end, err := rewrite(content[bullet], next, func(runs []object) []object { return runs })
	if err != nil {
		return nil, "", 0, err
	}
	return append(out, el), heading.text, end, nil
}

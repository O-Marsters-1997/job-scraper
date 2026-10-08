package checks

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	checkGrounding = "grounding"
	CheckSkills    = "skills"
)

var numberRe = regexp.MustCompile(`\d+(?:[.,]\d+)*(?:%|[xX×]|[kKmMbB]\b)?`)

// Grounding blocks numbers, technologies and proper nouns, and skills that the
// Bank does not support, and reports job skills with no source as info gaps.
func Grounding(d Draft) []Finding {
	var out []Finding
	for _, p := range d.Positions {
		scope := p.Achievements
		for _, b := range p.Bullets {
			out = append(out, groundSlot(b, b.Cited, scope)...)
		}
	}
	if d.Profile != nil {
		out = append(out, groundSlot(*d.Profile, d.Bank, d.Bank)...)
	}
	return append(out, groundSkills(d)...)
}

func groundSlot(s Slot, numberSource, termSource []string) []Finding {
	if s.Text == s.BaseText {
		return nil
	}
	var out []Finding
	numbers := normalise(strings.Join(numberSource, " "))
	for _, n := range numberRe.FindAllString(s.Text, -1) {
		if !containsNumber(numbers, n) {
			out = append(out, Finding{
				Check: checkGrounding, Severity: Block, SlotID: s.ID,
				Message: fmt.Sprintf("number %q does not appear in the cited Achievements", n),
			})
		}
	}
	terms := normalise(strings.Join(termSource, " "))
	seen := map[string]bool{}
	for _, t := range properTerms(s.Text) {
		key := strings.ToLower(t)
		if seen[key] {
			continue
		}
		seen[key] = true
		if !containsWord(terms, key) {
			out = append(out, Finding{
				Check: checkGrounding, Severity: Block, SlotID: s.ID,
				Message: fmt.Sprintf("%q does not appear in the Achievements it may draw on", t),
			})
		}
	}
	return out
}

func groundSkills(d Draft) []Finding {
	bank := normalise(strings.Join(d.Bank, " "))
	baseText := normalise(strings.Join(d.BaseText, " "))
	base := map[string]bool{}
	for _, line := range d.BaseSkills {
		for _, s := range line.Items {
			base[strings.ToLower(strings.TrimSpace(s))] = true
		}
	}
	bankSkill := map[string]bool{}
	for _, name := range d.BankSkills {
		bankSkill[strings.ToLower(strings.TrimSpace(name))] = true
	}
	sourced := func(skill string) bool {
		key := strings.ToLower(strings.TrimSpace(skill))
		return base[key] || bankSkill[key] || containsWord(bank, key) || containsWord(baseText, key)
	}

	var out []Finding
	for _, line := range d.Skills {
		for _, s := range line.Items {
			if !sourced(s) {
				out = append(out, Finding{
					Check: CheckSkills, Severity: Block,
					Message: fmt.Sprintf("skill %q is in neither the base skills nor the Bank", s),
				})
			}
		}
	}
	return out
}

func normalise(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, ",", ""))
}

func containsNumber(haystack, n string) bool {
	n = normalise(n)
	for _, m := range numberRe.FindAllString(haystack, -1) {
		if m == n {
			return true
		}
	}
	return false
}

func isTermRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

// containsWord matches needle on term boundaries, so "java" does not match
// "javascript" but "c++" and "node.js" still match.
func containsWord(haystack, needle string) bool {
	needle = strings.TrimSpace(needle)
	if needle == "" {
		return true
	}
	for from := 0; ; {
		i := strings.Index(haystack[from:], needle)
		if i < 0 {
			return false
		}
		start, end := from+i, from+i+len(needle)
		prev, _ := utf8.DecodeLastRuneInString(haystack[:start])
		next, _ := utf8.DecodeRuneInString(haystack[end:])
		before := start == 0 || !isTermRune(prev)
		after := end == len(haystack) || !isTermRune(next)
		if before && after {
			return true
		}
		from = start + 1
	}
}

var wordRe = regexp.MustCompile(`[\p{L}\p{N}][\p{L}\p{N}.+#/&'-]*`)

// properTerms returns technology-looking tokens: capitalised words that are
// not at the start of a sentence, and words with internal capitals, digits or
// symbols (PostgreSQL, Node.js, C++).
func properTerms(text string) []string {
	var out []string
	for _, loc := range wordRe.FindAllStringIndex(text, -1) {
		w := strings.TrimRight(text[loc[0]:loc[1]], ".'-/&")
		if w == "" || startsWithDigit(w) {
			continue
		}
		if isDistinctive(w) || (isCapitalised(w) && !atSentenceStart(text, loc[0])) {
			out = append(out, w)
		}
	}
	return out
}

func startsWithDigit(w string) bool {
	return unicode.IsDigit([]rune(w)[0])
}

func isCapitalised(w string) bool {
	return unicode.IsUpper([]rune(w)[0])
}

func isDistinctive(w string) bool {
	rs := []rune(w)
	for i, r := range rs {
		if i > 0 && unicode.IsUpper(r) {
			return true
		}
		if r == '+' || r == '#' || (i > 0 && (r == '.' || unicode.IsDigit(r))) {
			return true
		}
	}
	return false
}

func atSentenceStart(text string, idx int) bool {
	before := strings.TrimRightFunc(text[:idx], unicode.IsSpace)
	if before == "" {
		return true
	}
	last, _ := utf8.DecodeLastRuneInString(before)
	return strings.ContainsRune(".!?:•-", last)
}

package checks

import (
	"fmt"
	"slices"
	"strings"
)

// SkillLines blocks a skills edit that changes the base CV's lines: their
// count, order or labels, or the items in any line.
func SkillLines(d Draft) []Finding {
	if d.LegacySkills || len(d.Skills) == 0 || len(d.BaseSkills) == 0 {
		return nil
	}
	if len(d.Skills) != len(d.BaseSkills) {
		return []Finding{skillLinesBlock("skills has %d lines, the CV has %d", len(d.Skills), len(d.BaseSkills))}
	}
	var out []Finding
	for i, base := range d.BaseSkills {
		line := d.Skills[i]
		if strings.TrimSpace(line.Label) != base.Label {
			out = append(out, skillLinesBlock("skills line %d is labelled %q, the CV labels it %q", i+1, line.Label, base.Label))
			continue
		}
		if !sameItems(line.Items, base.Items) {
			out = append(out, skillLinesBlock("skills line %q must keep exactly its items, reordered: want %q, got %q", base.Label, base.Items, line.Items))
		}
	}
	return out
}

func skillLinesBlock(format string, args ...any) Finding {
	return Finding{Check: CheckSkills, Severity: Block, Message: fmt.Sprintf(format, args...)}
}

func sameItems(a, b []string) bool {
	return slices.Equal(sortedKeys(a), sortedKeys(b))
}

func sortedKeys(items []string) []string {
	keys := make([]string, len(items))
	for i, it := range items {
		keys[i] = strings.ToLower(strings.TrimSpace(it))
	}
	slices.Sort(keys)
	return keys
}

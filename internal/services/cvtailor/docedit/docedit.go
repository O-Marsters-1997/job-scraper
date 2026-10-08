package docedit

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/docparse"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
)

const defaultSkillSeparator = ", "

var (
	ErrUnknownPosition = errors.New("docedit: edit names a position with no slots")
	ErrTooManyBullets  = errors.New("docedit: more bullets than slots")
	ErrSkillLines      = errors.New("docedit: skills do not match the document's skill lines")
)

type Range struct {
	StartIndex int `json:"startIndex"`
	EndIndex   int `json:"endIndex"`
}

type Location struct {
	Index int `json:"index"`
}

type DeleteContentRange struct {
	Range Range `json:"range"`
}

type InsertText struct {
	Location Location `json:"location"`
	Text     string   `json:"text"`
}

// Request is one element of a batchUpdate requests array; the Docs API
// requires exactly one field set.
type Request struct {
	DeleteContentRange *DeleteContentRange `json:"deleteContentRange,omitempty"`
	InsertText         *InsertText         `json:"insertText,omitempty"`
}

type PositionSlots map[string][]string

type edit struct {
	start int
	reqs  []Request
}

// Requests orders batchUpdate requests by descending document index so
// earlier indices stay valid as the Docs API applies them.
func Requests(ds docparse.DocStructure, positions PositionSlots, edits cvedit.EditSet) ([]Request, error) {
	slots := make(map[string]docparse.Slot, len(ds.Slots))
	for _, s := range ds.Slots {
		slots[s.ID] = s
	}

	var all []edit
	for _, pe := range edits.Positions {
		ids, ok := positions[pe.PositionID]
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrUnknownPosition, pe.PositionID)
		}
		if len(pe.Bullets) > len(ids) {
			return nil, fmt.Errorf("%w: position %q has %d bullets for %d slots", ErrTooManyBullets, pe.PositionID, len(pe.Bullets), len(ids))
		}
		for i, id := range ids {
			slot, ok := slots[id]
			if !ok {
				return nil, fmt.Errorf("%w: slot %q not in document", ErrUnknownPosition, id)
			}
			if i < len(pe.Bullets) {
				if pe.Bullets[i].Text == slot.Text {
					continue
				}
				all = append(all, replace(slot.StartIndex, slot.EndIndex-1, pe.Bullets[i].Text))
				continue
			}
			all = append(all, remove(slot.StartIndex, slot.EndIndex))
		}
	}

	if ds.Profile != nil && edits.Profile != nil {
		all = append(all, replace(ds.Profile.StartIndex, ds.Profile.EndIndex-1, *edits.Profile))
	}
	if ds.Skills != nil && len(edits.Skills) > 0 {
		skillEdits, err := skillRequests(*ds.Skills, edits)
		if err != nil {
			return nil, err
		}
		all = append(all, skillEdits...)
	}

	slices.SortStableFunc(all, func(a, b edit) int { return b.start - a.start })
	var reqs []Request
	for _, e := range all {
		reqs = append(reqs, e.reqs...)
	}
	return reqs, nil
}

func skillRequests(s docparse.SkillsSlot, edits cvedit.EditSet) ([]edit, error) {
	if edits.LegacySkills {
		return []edit{replace(s.StartIndex, s.EndIndex-1, joinItems(s.Separator, s.List, cvedit.FlatSkills(edits.Skills)))}, nil
	}
	if len(edits.Skills) != len(s.Lines) {
		return nil, fmt.Errorf("%w: %d edited lines for %d in the document", ErrSkillLines, len(edits.Skills), len(s.Lines))
	}
	var out []edit
	for i, line := range s.Lines {
		group := edits.Skills[i]
		if strings.TrimSpace(group.Label) != line.Label {
			return nil, fmt.Errorf("%w: line %d is labelled %q, the document has %q", ErrSkillLines, i+1, group.Label, line.Label)
		}
		if slices.Equal(group.Items, line.Items) {
			continue
		}
		sep := line.Separator
		if sep == "" {
			sep = s.Separator
		}
		out = append(out, replace(line.ItemsStart, line.End, joinItems(sep, false, group.Items)))
	}
	return out, nil
}

func joinItems(separator string, list bool, items []string) string {
	if list {
		return strings.Join(items, "\n")
	}
	if separator == "" {
		separator = defaultSkillSeparator
	}
	return strings.Join(items, separator)
}

func remove(start, end int) edit {
	return edit{start: start, reqs: []Request{deleteRange(start, end)}}
}

func replace(start, end int, text string) edit {
	reqs := []Request{}
	if end > start {
		reqs = append(reqs, deleteRange(start, end))
	}
	reqs = append(reqs, Request{InsertText: &InsertText{Location: Location{Index: start}, Text: text}})
	return edit{start: start, reqs: reqs}
}

func deleteRange(start, end int) Request {
	return Request{DeleteContentRange: &DeleteContentRange{Range: Range{StartIndex: start, EndIndex: end}}}
}

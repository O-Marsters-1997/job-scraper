package cvtailor

import (
	"maps"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
)

func TestLeanRule_Replacements(t *testing.T) {
	t.Parallel()
	bank := []dto.Position{
		{ID: "p1", Achievements: []dto.Achievement{
			{ID: "a1", PositionID: "p1", Text: "Ran on-call"},
			{ID: "a2", PositionID: "p1", Text: "Cut p99 latency"},
			{ID: "a3", PositionID: "p1", Text: "Wrote docs"},
			{ID: "a4", PositionID: "p1", Text: "Mentored two engineers"},
		}},
		{ID: "p2", Achievements: []dto.Achievement{
			{ID: "b1", PositionID: "p2", Text: "Managed a team"},
			{ID: "b2", PositionID: "p2", Text: "Shipped a billing system"},
		}},
	}
	lean := func(id string, l float64) map[string]dto.Answer {
		for _, p := range bank {
			for _, a := range p.Achievements {
				if a.ID == id {
					return map[string]dto.Answer{achievementQuestion(a.Text): {PYes: max(l, 0), PNo: max(-l, 0)}}
				}
			}
		}
		return nil
	}
	merge := func(ms ...map[string]dto.Answer) map[string]dto.Answer {
		out := map[string]dto.Answer{}
		for _, m := range ms {
			maps.Copy(out, m)
		}
		return out
	}
	bullets := func(texts ...string) []dto.DraftBullet {
		out := make([]dto.DraftBullet, len(texts))
		for i, text := range texts {
			out[i] = dto.DraftBullet{Text: text, AchievementIDs: []string{}}
		}
		return out
	}
	one := func(texts ...string) dto.DraftContent {
		return dto.DraftContent{Positions: []dto.DraftPosition{{PositionID: "p1", Bullets: bullets(texts...)}}}
	}

	tests := []struct {
		name    string
		content dto.DraftContent
		answers map[string]dto.Answer
		lines   map[string]int
		want    []Replacement
	}{
		{
			name:    "replaces a low-lean bullet with the best unused achievement of its position",
			content: one("Ran on-call", "Wrote docs"),
			answers: merge(lean("a1", -0.5), lean("a3", 0.1), lean("a2", 0.6)),
			want:    []Replacement{{PositionID: "p1", SlotIndex: 0, AchievementIDs: []string{"a2"}}},
		},
		{
			name:    "keeps a bullet above the threshold",
			content: one("Ran on-call"),
			answers: merge(lean("a1", -0.1), lean("a2", 0.9)),
		},
		{
			name:    "keeps a bullet when no unused achievement beats it",
			content: one("Ran on-call"),
			answers: merge(lean("a1", -0.5), lean("a2", -0.5), lean("a3", -0.7)),
		},
		{
			name:    "never picks an achievement already in the CV",
			content: one("Ran on-call", "Cut p99 latency"),
			answers: merge(lean("a1", -0.5), lean("a2", 0.9), lean("a3", 0.2)),
			want:    []Replacement{{PositionID: "p1", SlotIndex: 0, AchievementIDs: []string{"a3"}}},
		},
		{
			name:    "leaves an unmapped position untouched",
			content: one("Ran on-call"),
			answers: merge(lean("a1", -0.5), lean("b1", -0.9), lean("b2", 0.9)),
		},
		{
			name:    "leaves a bullet that matches no achievement",
			content: one("Something the bank never saw"),
			answers: merge(lean("a2", 0.9)),
		},
		{
			name:    "two low bullets cannot take the same achievement",
			content: one("Ran on-call", "Wrote docs"),
			answers: merge(lean("a1", -0.5), lean("a3", -0.4), lean("a2", 0.6)),
			want:    []Replacement{{PositionID: "p1", SlotIndex: 0, AchievementIDs: []string{"a2"}}},
		},
		{
			name:    "a low 2-line bullet becomes two 1-line achievements in lean order",
			content: one("Ran on-call", "Wrote docs"),
			answers: merge(lean("a1", -0.5), lean("a3", 0.1), lean("a2", 0.6), lean("a4", 0.3)),
			lines:   map[string]int{"Ran on-call": 2},
			want:    []Replacement{{PositionID: "p1", SlotIndex: 0, AchievementIDs: []string{"a2", "a4"}}},
		},
		{
			name:    "a swap that leaves a 2-line gap keeps the original",
			content: one("Ran on-call"),
			answers: merge(lean("a1", -0.5), lean("a2", 0.6)),
			lines:   map[string]int{"Ran on-call": 3},
		},
		{
			name:    "a swap that leaves a 1-line gap goes ahead",
			content: one("Ran on-call"),
			answers: merge(lean("a1", -0.5), lean("a2", 0.6)),
			lines:   map[string]int{"Ran on-call": 2},
			want:    []Replacement{{PositionID: "p1", SlotIndex: 0, AchievementIDs: []string{"a2"}}},
		},
		{
			name:    "skips an achievement longer than the freed lines and takes a shorter one",
			content: one("Ran on-call"),
			answers: merge(lean("a1", -0.5), lean("a2", 0.9), lean("a3", 0.4)),
			lines:   map[string]int{"Cut p99 latency": 2},
			want:    []Replacement{{PositionID: "p1", SlotIndex: 0, AchievementIDs: []string{"a3"}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			textOf := map[string]string{}
			for _, p := range bank {
				for _, a := range p.Achievements {
					textOf[a.ID] = a.Text
				}
			}
			lines := func(_, text string) int { return max(tt.lines[text], 1) }
			got := leanRule{}.Replacements(RuleInput{Positions: bank, Content: tt.content, Answers: tt.answers, Lines: lines})
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Replacements() mismatch (-want +got):\n%s", diff)
			}

			for _, cp := range tt.content.Positions {
				before, after := 0, 0
				replaced := map[int][]string{}
				for _, r := range got {
					if r.PositionID == cp.PositionID {
						replaced[r.SlotIndex] = r.AchievementIDs
					}
				}
				for i, b := range cp.Bullets {
					before += lines(cp.PositionID, b.Text)
					if ids, ok := replaced[i]; ok {
						for _, id := range ids {
							after += lines(cp.PositionID, textOf[id])
						}
						continue
					}
					after += lines(cp.PositionID, b.Text)
				}
				if after > before {
					t.Errorf("role %s uses %d lines after Replacements(), source has %d", cp.PositionID, after, before)
				}
			}
		})
	}
}

func TestPlan_withReplacements(t *testing.T) {
	t.Parallel()
	newPlan := func() plan {
		return plan{positions: []planned{{
			Position: cvedit.Position{ID: "p1", SlotTexts: []string{"one", "two", "three"}},
			slotIDs:  []string{"s0", "s1", "s2"},
			texts:    map[string]string{},
		}}}
	}
	bank := []dto.Position{{ID: "p1", Achievements: []dto.Achievement{
		{ID: "a", Text: "A"}, {ID: "b", Text: "B"}, {ID: "c", Text: "C"}, {ID: "d", Text: "D"},
	}}}

	t.Run("kept bullets keep their slots and split extras follow the replaced slot", func(t *testing.T) {
		t.Parallel()
		got := newPlan().withReplacements(bank, []Replacement{{PositionID: "p1", SlotIndex: 1, AchievementIDs: []string{"a", "b"}}}).Positions[0]
		wantBullets := []cvedit.Bullet{
			{Keep: true, Text: "one", SlotID: "s0"},
			{Text: "A", AchievementIDs: []string{"a"}, SlotID: "s1"},
			{Keep: true, Text: "three", SlotID: "s2"},
		}
		if diff := cmp.Diff(wantBullets, got.Bullets); diff != "" {
			t.Errorf("withReplacements() bullets (-want +got):\n%s", diff)
		}
		wantAdded := []cvedit.AddedBullet{{SlotID: "new-p1-0", After: "s1", Text: "B", AchievementIDs: []string{"b"}}}
		if diff := cmp.Diff(wantAdded, got.Added); diff != "" {
			t.Errorf("withReplacements() added (-want +got):\n%s", diff)
		}
	})

	t.Run("AppendSlot adds after the last bullet, in order", func(t *testing.T) {
		t.Parallel()
		reps := []Replacement{
			{PositionID: "p1", SlotIndex: AppendSlot, AchievementIDs: []string{"c"}},
			{PositionID: "p1", SlotIndex: AppendSlot, AchievementIDs: []string{"d"}},
		}
		got := newPlan().withReplacements(bank, reps).Positions[0].Added
		want := []cvedit.AddedBullet{
			{SlotID: "new-p1-0", After: "s2", Text: "C", AchievementIDs: []string{"c"}},
			{SlotID: "new-p1-1", After: "new-p1-0", Text: "D", AchievementIDs: []string{"d"}},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("withReplacements() added (-want +got):\n%s", diff)
		}
	})
}

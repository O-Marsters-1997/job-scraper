package cvtailor

import (
	"maps"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func TestLeanRule_Replacements(t *testing.T) {
	t.Parallel()
	bank := []dto.Position{
		{ID: "p1", Achievements: []dto.Achievement{
			{ID: "a1", PositionID: "p1", Text: "Ran on-call"},
			{ID: "a2", PositionID: "p1", Text: "Cut p99 latency"},
			{ID: "a3", PositionID: "p1", Text: "Wrote docs"},
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

	tests := []struct {
		name    string
		content dto.DraftContent
		answers map[string]dto.Answer
		want    []Replacement
	}{
		{
			name:    "replaces a low-lean bullet with the best unused achievement of its position",
			content: dto.DraftContent{Positions: []dto.DraftPosition{{PositionID: "p1", Bullets: bullets("Ran on-call", "Wrote docs")}}},
			answers: merge(lean("a1", -0.5), lean("a3", 0.1), lean("a2", 0.6)),
			want:    []Replacement{{PositionID: "p1", SlotIndex: 0, AchievementID: "a2"}},
		},
		{
			name:    "keeps a bullet above the threshold",
			content: dto.DraftContent{Positions: []dto.DraftPosition{{PositionID: "p1", Bullets: bullets("Ran on-call")}}},
			answers: merge(lean("a1", -0.1), lean("a2", 0.9)),
		},
		{
			name:    "keeps a bullet when no unused achievement beats it",
			content: dto.DraftContent{Positions: []dto.DraftPosition{{PositionID: "p1", Bullets: bullets("Ran on-call")}}},
			answers: merge(lean("a1", -0.5), lean("a2", -0.5), lean("a3", -0.7)),
		},
		{
			name:    "never picks an achievement already in the CV",
			content: dto.DraftContent{Positions: []dto.DraftPosition{{PositionID: "p1", Bullets: bullets("Ran on-call", "Cut p99 latency")}}},
			answers: merge(lean("a1", -0.5), lean("a2", 0.9), lean("a3", 0.2)),
			want:    []Replacement{{PositionID: "p1", SlotIndex: 0, AchievementID: "a3"}},
		},
		{
			name:    "leaves an unmapped position untouched",
			content: dto.DraftContent{Positions: []dto.DraftPosition{{PositionID: "p1", Bullets: bullets("Ran on-call")}}},
			answers: merge(lean("a1", -0.5), lean("b1", -0.9), lean("b2", 0.9)),
		},
		{
			name:    "leaves a bullet that matches no achievement",
			content: dto.DraftContent{Positions: []dto.DraftPosition{{PositionID: "p1", Bullets: bullets("Something the bank never saw")}}},
			answers: merge(lean("a2", 0.9)),
		},
		{
			name:    "two low bullets cannot take the same achievement",
			content: dto.DraftContent{Positions: []dto.DraftPosition{{PositionID: "p1", Bullets: bullets("Ran on-call", "Wrote docs")}}},
			answers: merge(lean("a1", -0.5), lean("a3", -0.4), lean("a2", 0.6)),
			want:    []Replacement{{PositionID: "p1", SlotIndex: 0, AchievementID: "a2"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := leanRule{}.Replacements(bank, tt.content, tt.answers)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Replacements() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

package cvedit_test

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
)

func TestEditSet_StoredJobSkillsAreIgnored(t *testing.T) {
	var e cvedit.EditSet
	stored := `{"positions":[],"skills":[{"label":"Languages","items":["Go"]}],"jobSkills":["Pulumi"]}`
	if err := json.Unmarshal([]byte(stored), &e); err != nil {
		t.Fatalf("Unmarshal(%s): %v", stored, err)
	}
	if len(e.Skills) != 1 || e.Skills[0].Items[0] != "Go" {
		t.Errorf("skills = %v, want the Languages line", e.Skills)
	}
}

func TestEditSet_SkillsRoundTrip(t *testing.T) {
	t.Run("grouped skills decode and encode unchanged", func(t *testing.T) {
		raw := `{"positions":[],"skills":[{"label":"Languages","items":["Go","SQL"]},{"label":"","items":["Git"]}]}`
		var got cvedit.EditSet
		if err := json.Unmarshal([]byte(raw), &got); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		want := []cvedit.SkillGroup{{Label: "Languages", Items: []string{"Go", "SQL"}}, {Items: []string{"Git"}}}
		if diff := cmp.Diff(want, got.Skills); diff != "" || got.LegacySkills {
			t.Errorf("Skills mismatch (-want +got):\n%s\nLegacySkills = %v, want false", diff, got.LegacySkills)
		}
		out, err := json.Marshal(got)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		if diff := cmp.Diff(raw, string(out)); diff != "" {
			t.Errorf("Marshal mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("flat skills decode as one unlabelled legacy group and encode back flat", func(t *testing.T) {
		raw := `{"positions":[],"skills":["Go","SQL"]}`
		var got cvedit.EditSet
		if err := json.Unmarshal([]byte(raw), &got); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		want := []cvedit.SkillGroup{{Items: []string{"Go", "SQL"}}}
		if diff := cmp.Diff(want, got.Skills); diff != "" || !got.LegacySkills {
			t.Errorf("Skills mismatch (-want +got):\n%s\nLegacySkills = %v, want true", diff, got.LegacySkills)
		}
		out, err := json.Marshal(got)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		if diff := cmp.Diff(raw, string(out)); diff != "" {
			t.Errorf("Marshal mismatch (-want +got):\n%s", diff)
		}
	})
}

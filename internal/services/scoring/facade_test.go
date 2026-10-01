package scoring_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
	"github.com/ollymarsters/job-scraper/internal/services/scoring"
)

func TestCompanyProfiles(t *testing.T) {
	const userID = "user-1"
	st := newFakeStore()
	st.SeedOptions(append([]dto.ScoringOption{retiredCobol}, bank...))
	st.SeedSearchConfig(picking(userID, "tech:go", "tech:rust", "tech:cobol"))

	yes := dto.Answer{PYes: 0.9, PNo: 0.05, PNotStated: 0.05}
	no := dto.Answer{PYes: 0.05, PNo: 0.9, PNotStated: 0.05}
	unsure := dto.Answer{PYes: 0.4, PNo: 0.3, PNotStated: 0.3}
	goHash, rustHash := scoring.QuestionHash("Does the role use Go?"), scoring.QuestionHash("Does the role use Rust?")
	for _, j := range []struct {
		id      string
		company string
		answers map[string]dto.Answer
	}{
		{"a1", "acme", map[string]dto.Answer{goHash: yes, rustHash: unsure}},
		{"a2", "acme", map[string]dto.Answer{goHash: no}},
		{"a3", "acme", nil},
		{"b1", "bare", nil},
	} {
		st.SeedJob(dto.Job{ID: j.id, CompanyID: j.company, ContentFingerprint: "fp"}, nil)
		st.SeedAnswers(j.id, "fp", jev.Model, j.answers)
	}

	m := scoring.Build(scoring.Deps{Store: st})
	got, err := m.CompanyProfiles(t.Context(), userID, []string{"acme", "bare", "empty"})
	if err != nil {
		t.Fatalf("CompanyProfiles() err = %v", err)
	}

	want := map[string][]dto.CompanyProfileEntry{
		"acme": {
			{Dimension: dto.DimensionTech, Label: "Go", Yes: 1, Known: 2, Total: 3},
			{Dimension: dto.DimensionTech, Label: "Rust", Total: 3},
		},
		"bare": {
			{Dimension: dto.DimensionTech, Label: "Go", Total: 1},
			{Dimension: dto.DimensionTech, Label: "Rust", Total: 1},
		},
		"empty": {
			{Dimension: dto.DimensionTech, Label: "Go"},
			{Dimension: dto.DimensionTech, Label: "Rust"},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("CompanyProfiles() mismatch (-want +got):\n%s", diff)
	}
}

package docparse_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/docparse"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		file string
		want docparse.DocStructure
	}{
		{
			name: "profile and list skills",
			file: "profile_skills_list.json",
			want: docparse.DocStructure{
				Headings: []docparse.Heading{
					{Text: "Profile", Level: 1, Index: 10},
					{Text: "Experience", Level: 1, Index: 91},
					{Text: "Senior Engineer, Acme Ltd (2021 - Present)", Level: 2, Index: 102},
					{Text: "Engineer, Widgets Inc (2018 - 2021)", Level: 2, Index: 267},
					{Text: "Skills", Level: 1, Index: 357},
					{Text: "Education", Level: 1, Index: 389},
				},
				Slots: []docparse.Slot{
					{ID: "s0", HeadingIndex: 2, Text: "Led migration of billing to event-driven services, cutting p99 latency by 40%.", StartIndex: 145, EndIndex: 224},
					{ID: "s1", HeadingIndex: 2, Text: "Mentored four engineers through promotion.", StartIndex: 224, EndIndex: 267},
					{ID: "s2", HeadingIndex: 3, Text: "Built the ingestion API handling 2M requests per day.", StartIndex: 303, EndIndex: 357},
				},
				Profile: &docparse.Slot{ID: "profile", HeadingIndex: 0, Text: "Backend engineer with six years building Go services and data pipelines.", StartIndex: 18, EndIndex: 91},
				Skills:  &docparse.SkillsSlot{Items: []string{"Go", "PostgreSQL", "Kubernetes"}, List: true, StartIndex: 364, EndIndex: 389},
			},
		},
		{
			name: "no profile or skills",
			file: "no_profile_skills.json",
			want: docparse.DocStructure{
				Headings: []docparse.Heading{
					{Text: "Work History", Level: 2, Index: 12},
					{Text: "Data Analyst | Northwind", Level: 3, Index: 25},
					{Text: "Intern | Contoso", Level: 3, Index: 149},
					{Text: "Education", Level: 2, Index: 216},
				},
				Slots: []docparse.Slot{
					{ID: "s0", HeadingIndex: 1, Text: "Automated weekly reporting in Python, saving 10 hours a week.", StartIndex: 50, EndIndex: 112},
					{ID: "s1", HeadingIndex: 1, Text: "Built dashboards used by 200 staff.", StartIndex: 113, EndIndex: 149},
					{ID: "s2", HeadingIndex: 2, Text: "Cleaned survey data for a 5,000 respondent study.", StartIndex: 166, EndIndex: 216},
					{ID: "s3", HeadingIndex: 3, Text: "MSc Statistics", StartIndex: 226, EndIndex: 241},
				},
			},
		},
		{
			name: "profile in a table and prose skills",
			file: "table_profile_prose_skills.json",
			want: docparse.DocStructure{
				Headings: []docparse.Heading{
					{Text: "About me", Level: 2, Index: 14},
					{Text: "Experience", Level: 2, Index: 76},
					{Text: "Tech Lead, Globex", Level: 3, Index: 87},
					{Text: "Developer, Initech", Level: 3, Index: 186},
					{Text: "Technical Skills", Level: 2, Index: 255},
				},
				Slots: []docparse.Slot{
					{ID: "s0", HeadingIndex: 2, Text: "Shipped a design system adopted by five teams.", StartIndex: 105, EndIndex: 152},
					{ID: "s1", HeadingIndex: 2, Text: "Cut CI time from 25 to 9 minutes.", StartIndex: 152, EndIndex: 186},
					{ID: "s2", HeadingIndex: 3, Text: "Rebuilt checkout in React, lifting conversion 8%.", StartIndex: 205, EndIndex: 255},
				},
				Profile: &docparse.Slot{ID: "profile", HeadingIndex: 0, Text: "Full-stack developer focused on accessible products.", StartIndex: 23, EndIndex: 76},
				Skills: &docparse.SkillsSlot{
					Items:      []string{"TypeScript", "React", "Node.js", "PostgreSQL", "Docker"},
					Separator:  ", ",
					StartIndex: 272,
					EndIndex:   319,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("testdata", tt.file))
			if err != nil {
				t.Fatal(err)
			}
			got, err := docparse.Parse(raw)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Parse mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseRejectsInvalidJSON(t *testing.T) {
	if _, err := docparse.Parse([]byte("{")); err == nil {
		t.Fatal("want error for invalid JSON")
	}
}

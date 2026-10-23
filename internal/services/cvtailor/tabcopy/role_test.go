package tabcopy_test

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/tabcopy"
)

func TestRestyleRoleHeading(t *testing.T) {
	t.Parallel()
	type role struct {
		Title, Employer string
		Start, End      *string
	}
	var fixture struct {
		Cases []struct {
			Name        string
			SiblingRuns []tabcopy.TestRun
			From, To    role
			WantRuns    []tabcopy.TestRun
		}
	}
	if err := json.Unmarshal(readFixture(t, "role_heading.json"), &fixture); err != nil {
		t.Fatal(err)
	}
	for _, tt := range fixture.Cases {
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()
			got := tabcopy.Restyle(tt.SiblingRuns, tabcopy.Role(tt.From), tabcopy.Role(tt.To))
			if diff := cmp.Diff(tt.WantRuns, got); diff != "" {
				t.Errorf("Restyle() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

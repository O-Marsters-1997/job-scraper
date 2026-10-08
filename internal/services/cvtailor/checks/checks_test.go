package checks_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/checks"
)

func bulletDraft(text, base string, cited, positionAchievements []string) checks.Draft {
	return checks.Draft{Positions: []checks.Position{{
		ID:           "p1",
		Achievements: positionAchievements,
		Bullets:      []checks.Slot{{ID: "s1", Text: text, BaseText: base, Cited: cited}},
	}}}
}

type want struct {
	Severity checks.Severity
	SlotID   string
}

func findings(got []checks.Finding, checkName string) []want {
	var out []want
	for _, f := range got {
		if f.Check == checkName {
			out = append(out, want{f.Severity, f.SlotID})
		}
	}
	return out
}

func TestGroundingBullets(t *testing.T) {
	tests := []struct {
		name  string
		draft checks.Draft
		want  []want
	}{
		{
			name: "invented number blocks",
			draft: bulletDraft("Cut latency by 40%", "", []string{"Reduced latency"},
				[]string{"Reduced latency"}),
			want: []want{{checks.Block, "s1"}},
		},
		{
			name: "number from cited Achievement passes",
			draft: bulletDraft("Cut latency by 40%", "", []string{"Reduced latency by 40%"},
				[]string{"Reduced latency by 40%"}),
		},
		{
			name: "thousands separator is ignored",
			draft: bulletDraft("Served 1,000,000 requests", "", []string{"Served 1000000 requests"},
				[]string{"Served 1000000 requests"}),
		},
		{
			name: "number only in an uncited Achievement blocks",
			draft: bulletDraft("Handled 300 tickets", "", []string{"Handled tickets"},
				[]string{"Handled tickets", "Closed 300 tickets"}),
			want: []want{{checks.Block, "s1"}},
		},
		{
			name: "technology from another Position blocks",
			draft: bulletDraft("Deployed services with Kubernetes", "", []string{"Deployed services"},
				[]string{"Deployed services", "Wrote Go"}),
			want: []want{{checks.Block, "s1"}},
		},
		{
			name: "technology from this Position passes",
			draft: bulletDraft("Deployed services on Kubernetes", "", []string{"Deployed services"},
				[]string{"Deployed services", "Ran Kubernetes clusters"}),
		},
		{
			name: "match is case-insensitive",
			draft: bulletDraft("Tuned PostgreSQL", "", []string{"tuned postgresql"},
				[]string{"tuned postgresql"}),
		},
		{
			name: "leading verb is not a proper noun",
			draft: bulletDraft("Reduced latency", "", []string{"Cut latency"},
				[]string{"Cut latency"}),
		},
		{
			name: "symbol technologies are checked",
			draft: bulletDraft("Rewrote parsers in C++ and Node.js", "", []string{"Rewrote parsers"},
				[]string{"Rewrote parsers"}),
			want: []want{{checks.Block, "s1"}, {checks.Block, "s1"}},
		},
		{
			name: "unchanged slot text is not checked",
			draft: bulletDraft("Ran 40 Kubernetes clusters", "Ran 40 Kubernetes clusters", nil,
				[]string{"Wrote Go"}),
		},
		{
			name: "substring of another term does not ground",
			draft: bulletDraft("Wrote Java services", "", []string{"Wrote JavaScript services"},
				[]string{"Wrote JavaScript services"}),
			want: []want{{checks.Block, "s1"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, findings(checks.Grounding(tt.draft), "grounding")); diff != "" {
				t.Errorf("grounding findings (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGroundingProfileUsesWholeBank(t *testing.T) {
	bank := []string{"Ran Kubernetes clusters", "Improved Postgres queries by 30%"}
	tests := []struct {
		name string
		text string
		want []want
	}{
		{"terms and numbers from anywhere in the Bank pass", "Engineer who runs Kubernetes and improved queries by 30%", nil},
		{"term outside the Bank blocks", "Engineer who runs Terraform", []want{{checks.Block, "profile"}}},
		{"number outside the Bank blocks", "Engineer with 10 years of experience", []want{{checks.Block, "profile"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := checks.Draft{Bank: bank, Profile: &checks.Slot{ID: "profile", Text: tt.text}}
			if diff := cmp.Diff(tt.want, findings(checks.Grounding(d), "grounding")); diff != "" {
				t.Errorf("grounding findings (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGroundingSkills(t *testing.T) {
	tests := []struct {
		name       string
		skills     []string
		baseSkills []string
		bank       []string
		baseText   []string
		jobSkills  []string
		wantSkills []want
	}{
		{
			name:       "skills from base or Bank pass",
			skills:     []string{"Go", "Kubernetes"},
			baseSkills: []string{"Go", "SQL"},
			bank:       []string{"Ran Kubernetes clusters"},
		},
		{
			name:       "unsourced skill blocks",
			skills:     []string{"Go", "Pulumi"},
			baseSkills: []string{"Go"},
			wantSkills: []want{{checks.Block, ""}},
		},
		{
			name:       "job skill without a source is an info gap and stays out of skills",
			skills:     []string{"Go"},
			baseSkills: []string{"Go"},
			bank:       []string{"Wrote Terraform modules"},
			jobSkills:  []string{"Pulumi", "Terraform", "go"},
			wantSkills: []want{{checks.Info, ""}},
		},
		{
			name:      "job skill named anywhere in the base CV is no gap",
			baseText:  []string{"Languages/ Frameworks \t\tTypescript", "Databases\t\t\t\tPostgreSQL"},
			jobSkills: []string{"TypeScript", "PostgreSQL"},
		},
		{
			name:       "job skill listed in skills is both blocked and reported",
			skills:     []string{"Pulumi"},
			jobSkills:  []string{"Pulumi"},
			wantSkills: []want{{checks.Block, ""}, {checks.Info, ""}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := checks.Draft{Skills: oneLine(tt.skills), BaseSkills: oneLine(tt.baseSkills), Bank: tt.bank, BaseText: tt.baseText, JobSkills: tt.jobSkills}
			if diff := cmp.Diff(tt.wantSkills, findings(checks.Grounding(d), "skills")); diff != "" {
				t.Errorf("skills findings (-want +got):\n%s", diff)
			}
		})
	}
}

func oneLine(items []string) []checks.SkillLine {
	if items == nil {
		return nil
	}
	return []checks.SkillLine{{Items: items}}
}

func TestSkillLines(t *testing.T) {
	base := []checks.SkillLine{{Label: "Languages", Items: []string{"Go", "TypeScript"}}, {Label: "Databases", Items: []string{"Postgres", "Redis"}}}
	tests := []struct {
		name   string
		skills []checks.SkillLine
		legacy bool
		want   []want
	}{
		{"items reordered within their lines pass", []checks.SkillLine{{Label: "Languages", Items: []string{"typescript", "Go"}}, {Label: "Databases", Items: []string{"Redis", "Postgres"}}}, false, nil},
		{"no skills edit passes", nil, false, nil},
		{"dropped line blocks", base[:1], false, []want{{checks.Block, ""}}},
		{"extra line blocks", append(slices.Clone(base), checks.SkillLine{Label: "Tools", Items: []string{"Git"}}), false, []want{{checks.Block, ""}}},
		{"swapped lines block", []checks.SkillLine{base[1], base[0]}, false, []want{{checks.Block, ""}, {checks.Block, ""}}},
		{"renamed label blocks", []checks.SkillLine{{Label: "Langs", Items: base[0].Items}, base[1]}, false, []want{{checks.Block, ""}}},
		{"dropped item blocks", []checks.SkillLine{{Label: "Languages", Items: []string{"Go"}}, base[1]}, false, []want{{checks.Block, ""}}},
		{"added item blocks", []checks.SkillLine{base[0], {Label: "Databases", Items: []string{"Postgres", "Redis", "MySQL"}}}, false, []want{{checks.Block, ""}}},
		{"item moved across lines blocks", []checks.SkillLine{{Label: "Languages", Items: []string{"Go", "TypeScript", "Redis"}}, {Label: "Databases", Items: []string{"Postgres"}}}, false, []want{{checks.Block, ""}, {checks.Block, ""}}},
		{"legacy skills are not line-checked", []checks.SkillLine{{Items: []string{"Go"}}}, true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := checks.Draft{Skills: tt.skills, BaseSkills: base, LegacySkills: tt.legacy}
			if diff := cmp.Diff(tt.want, findings(checks.SkillLines(d), "skills")); diff != "" {
				t.Errorf("SkillLines() findings (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGapMessageNamesSkill(t *testing.T) {
	got := checks.Grounding(checks.Draft{JobSkills: []string{"Pulumi"}})
	if len(got) != 1 || got[0].Severity != checks.Info || !strings.Contains(got[0].Message, "Pulumi") {
		t.Fatalf("got %+v", got)
	}
}

func TestBannedWords(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []want
	}{
		{"clean", "Reduced latency", nil},
		{"banned word blocks", "Leveraged Kafka", []want{{checks.Block, "s1"}}},
		{"inflections match", "Utilising and utilizing", []want{{checks.Block, "s1"}, {checks.Block, "s1"}}},
		{"repeat reported once", "Spearheaded, then spearheaded again", []want{{checks.Block, "s1"}}},
		{"substring does not match", "Delivered a passionately-named tool", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := bulletDraft(tt.text, "", nil, nil)
			if diff := cmp.Diff(tt.want, findings(checks.BannedWords(d), "banned_words")); diff != "" {
				t.Errorf("banned_words findings (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBannedWordsSkipUnchangedText(t *testing.T) {
	d := bulletDraft("Leveraged Kafka", "Leveraged Kafka", nil, nil)
	if got := findings(checks.BannedWords(d), "banned_words"); len(got) != 0 {
		t.Errorf("banned_words findings = %v, want none for the user's own text", got)
	}
}

func TestBannedWordsCoverProfile(t *testing.T) {
	d := checks.Draft{Profile: &checks.Slot{ID: "profile", Text: "A passionate engineer"}}
	if diff := cmp.Diff([]want{{checks.Block, "profile"}}, findings(checks.BannedWords(d), "banned_words")); diff != "" {
		t.Errorf("banned_words findings (-want +got):\n%s", diff)
	}
}

func TestSlotLength(t *testing.T) {
	base := strings.Repeat("a", 100)
	tests := []struct {
		name string
		text string
		want []want
	}{
		{"shorter", strings.Repeat("a", 60), nil},
		{"same length", strings.Repeat("a", 100), nil},
		{"one character over", strings.Repeat("a", 101), []want{{checks.Block, "s1"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, findings(checks.SlotLength(bulletDraft(tt.text, base, nil, nil)), "slot_length")); diff != "" {
				t.Errorf("slot_length findings (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSlotLengthProfileAndMultibyte(t *testing.T) {
	d := checks.Draft{Profile: &checks.Slot{ID: "profile", BaseText: strings.Repeat("é", 100), Text: strings.Repeat("é", 120)}}
	if diff := cmp.Diff([]want{{checks.Block, "profile"}}, findings(checks.SlotLength(d), "slot_length")); diff != "" {
		t.Errorf("slot_length findings (-want +got):\n%s", diff)
	}

	d.Profile.Text = strings.Repeat("é", 100)
	if diff := cmp.Diff([]want(nil), findings(checks.SlotLength(d), "slot_length")); diff != "" {
		t.Errorf("slot_length findings (-want +got):\n%s", diff)
	}
}

func TestPageCount(t *testing.T) {
	tests := []struct {
		name        string
		base, draft int
		want        []want
	}{
		{"same", 1, 1, nil},
		{"fewer", 2, 1, nil},
		{"over the base", 1, 2, []want{{checks.Block, ""}}},
		{"unmeasured", 1, 0, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checks.PageCount(checks.Draft{BasePages: tt.base, DraftPages: tt.draft})
			if diff := cmp.Diff(tt.want, findings(got, "page_count")); diff != "" {
				t.Errorf("page_count findings (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRunCollectsEveryCheck(t *testing.T) {
	d := bulletDraft("Leveraged Terraform to cut costs by 40%", "short", []string{"Cut costs"}, []string{"Cut costs"})
	d.BasePages, d.DraftPages = 1, 2

	seen := map[string]bool{}
	for _, f := range checks.Run(d) {
		seen[f.Check] = true
	}
	for _, name := range []string{"grounding", "banned_words", "slot_length", "page_count"} {
		if !seen[name] {
			t.Errorf("Run produced no %s finding", name)
		}
	}
}

func TestContact(t *testing.T) {
	tests := []struct {
		name  string
		input *checks.ContactInput
		want  []want
	}{
		{name: "not measured emits nothing"},
		{name: "in body emits nothing", input: &checks.ContactInput{InBody: true}},
		{name: "in body and header emits nothing", input: &checks.ContactInput{InBody: true, InHeaderFooter: true}},
		{name: "only in header or footer is info", input: &checks.ContactInput{InHeaderFooter: true}, want: []want{{Severity: checks.Info}}},
		{name: "nowhere is info", input: &checks.ContactInput{}, want: []want{{Severity: checks.Info}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findings(checks.Contact(checks.Draft{Contact: tt.input}), "contact")
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Contact() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

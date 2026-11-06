package promptparts_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/promptparts"
)

func TestMessage(t *testing.T) {
	t.Parallel()
	g := promptparts.Generation{
		Name: promptparts.ProfileRedraft,
		Main: "main.md",
		Parts: []promptparts.Part{
			{Path: "main.md", Default: "MAIN-DEFAULT"},
			{Path: "voice.md", Default: "VOICE-DEFAULT"},
		},
	}
	docs := []promptparts.Document{{Path: "job.md", Text: "JOB"}, {Path: "profile.md", Text: "PROFILE"}}
	const head = "<documents>\n" +
		"<document path=\"job.md\">\nJOB\n</document>\n" +
		"<document path=\"profile.md\">\nPROFILE\n</document>\n"

	tests := []struct {
		name  string
		texts map[string]string
		steer string
		want  string
	}{
		{
			name: "defaults fill every missing path",
			want: head + "<document path=\"voice.md\">\nVOICE-DEFAULT\n</document>\n" +
				"</documents>\n\n<instructions path=\"main.md\">\nMAIN-DEFAULT\n</instructions>",
		},
		{
			name:  "texts override defaults and the steer closes the message",
			texts: map[string]string{"voice.md": "MINE", "main.md": "MY-MAIN"},
			steer: "  shorter  ",
			want: head + "<document path=\"voice.md\">\nMINE\n</document>\n" +
				"</documents>\n\n<instructions path=\"main.md\">\nMY-MAIN\n</instructions>" +
				"\n\nThe person asks: shorter",
		},
		{
			name:  "a blank steer leaves no steer line",
			steer: " \n",
			want: head + "<document path=\"voice.md\">\nVOICE-DEFAULT\n</document>\n" +
				"</documents>\n\n<instructions path=\"main.md\">\nMAIN-DEFAULT\n</instructions>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := promptparts.Message(g, tt.texts, docs, tt.steer); got != tt.want {
				t.Errorf("Message() =\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}

func TestSystemIsEmbedded(t *testing.T) {
	t.Parallel()
	if strings.TrimSpace(promptparts.System) == "" {
		t.Error("System is empty")
	}
}

func TestResolve(t *testing.T) {
	t.Parallel()
	g := promptparts.Generation{Parts: []promptparts.Part{
		{Path: "main.md", Default: "MAIN"},
		{Path: "voice.md", Default: "VOICE"},
	}}

	got := promptparts.Resolve(g, []promptparts.Override{
		{Path: "voice.md", Text: "MINE"},
		{Path: "gone.md", Text: "ignored"},
	})

	want := []promptparts.Resolved{
		{Path: "main.md", Text: "MAIN", Source: promptparts.SourceDefault},
		{Path: "voice.md", Text: "MINE", Source: promptparts.SourceYours},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Resolve() (-want +got):\n%s", diff)
	}
}

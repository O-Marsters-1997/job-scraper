package promptparts

import (
	_ "embed"
	"fmt"
	"strings"
)

type GenerationName string

const ProfileRedraft GenerationName = "profile-redraft"

type Generation struct {
	Name  GenerationName
	Main  string
	Parts []Part
}

type Part struct {
	Path    string
	Default string
}

type Document struct {
	Path string
	Text string
}

// System is the shared system prompt (embedded system.md).
//
//go:embed system.md
var System string

// Message is the user turn: docs, then the non-main Parts as documents, then the main Part as
// instructions, then the steer. texts maps Part path to text; a missing path uses its Default.
func Message(g Generation, texts map[string]string, docs []Document, steer string) string {
	text := func(p Part) string {
		if t, ok := texts[p.Path]; ok {
			return t
		}
		return p.Default
	}
	var b strings.Builder
	b.WriteString("<documents>\n")
	for _, d := range docs {
		writeDocument(&b, d.Path, d.Text)
	}
	var main string
	for _, p := range g.Parts {
		if p.Path == g.Main {
			main = text(p)
			continue
		}
		writeDocument(&b, p.Path, text(p))
	}
	fmt.Fprintf(&b, "</documents>\n\n<instructions path=%q>\n%s\n</instructions>", g.Main, main)
	if steer = strings.TrimSpace(steer); steer != "" {
		fmt.Fprintf(&b, "\n\nThe person asks: %s", steer)
	}
	return b.String()
}

func writeDocument(b *strings.Builder, path, text string) {
	fmt.Fprintf(b, "<document path=%q>\n%s\n</document>\n", path, text)
}

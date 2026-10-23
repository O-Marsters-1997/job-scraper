package tabcopy

import (
	"encoding/json"
	"strings"
)

type TestRun struct {
	Text string
	Bold bool
}

func Restyle(runs []TestRun, from, to Role) []TestRun {
	objs := make([]object, len(runs))
	var b strings.Builder
	for i, r := range runs {
		b.WriteString(r.Text)
		objs[i] = object{"textRun": mustMarshal(map[string]any{"content": r.Text, "textStyle": map[string]any{"bold": r.Bold}})}
	}
	whole := b.String()
	named :=from.Title != "" && strings.Contains(whole, from.Title) || from.Employer != "" && strings.Contains(whole, from.Employer)
	out := restyle(objs, named, true, from, to)
	got := make([]TestRun, len(out))
	for i, o := range out {
		var tr struct {
			Content   string
			TextStyle struct{ Bold bool }
		}
		_ = json.Unmarshal(o["textRun"], &tr)
		got[i] = TestRun{tr.Content, tr.TextStyle.Bold}
	}
	return got
}

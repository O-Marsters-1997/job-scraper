package eval

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

//go:embed fixtures/suggest/*.json fixtures/profile/*.json
var fixtureFS embed.FS

func loadFixtures[T any](dir string, name func(*T) *string) ([]T, error) {
	entries, err := fs.ReadDir(fixtureFS, dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	var out []T
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		raw, err := fixtureFS.ReadFile(path.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read fixture %s: %w", e.Name(), err)
		}
		var f T
		if err := json.Unmarshal(raw, &f); err != nil {
			return nil, fmt.Errorf("fixture %s: %w", e.Name(), err)
		}
		if n := name(&f); *n == "" {
			*n = strings.TrimSuffix(e.Name(), ".json")
		}
		out = append(out, f)
	}
	return out, nil
}

func rate(n, total int) string {
	if total == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%d/%d (%.0f%%)", n, total, 100*float64(n)/float64(total))
}

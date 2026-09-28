// Package sourcetest holds fixture-driven test helpers for source parsers.
package sourcetest

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

// RunSnapshotTests runs snapshot tests for every HTML file in snapshots/.
// list_*.html fixtures parse with ParseURLs; detail_*.html fixtures parse
// with ParseJobDetail, using the JSON fixture's first entry's URL.
func RunSnapshotTests(t *testing.T, src sources.SnapshotSource) {
	t.Helper()

	files, _ := filepath.Glob("snapshots/*.html")
	if len(files) == 0 {
		t.Skip("no snapshots")
	}

	for _, htmlPath := range files {
		name := strings.TrimSuffix(filepath.Base(htmlPath), ".html")
		t.Run(name, func(t *testing.T) {
			jsonPath := filepath.Join("snapshots", name+".json")
			wantBytes, err := os.ReadFile(jsonPath)
			if err != nil {
				t.Fatalf("read snapshot json %s: %v (run `just cli rebase <source>` to create it)", jsonPath, err)
			}

			var want []dto.Job
			if err := json.Unmarshal(wantBytes, &want); err != nil {
				t.Fatalf("unmarshal snapshot json: %v", err)
			}

			f, err := os.Open(htmlPath)
			if err != nil {
				t.Fatalf("open html: %v", err)
			}
			defer func() { _ = f.Close() }()

			var got []dto.Job
			if strings.HasPrefix(name, "detail_") {
				if len(want) == 0 {
					t.Fatal("detail snapshot json must have at least one entry with URL set")
				}
				job, err := src.ParseJobDetail(f, want[0].URL)
				if err != nil {
					t.Fatalf("parse: %v", err)
				}
				got = []dto.Job{job}
			} else {
				got, err = src.ParseURLs(f)
				if err != nil {
					t.Fatalf("parse: %v", err)
				}
			}

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("parser output does not match snapshot (-want +got):\n%s", diff)
			}
		})
	}
}

// FuzzSnapshots fuzzes src's list and detail parsers, seeded from snapshots/*.html. The
// invariant is that neither parser panics.
func FuzzSnapshots(f *testing.F, src sources.SnapshotSource) {
	f.Helper()

	files, _ := filepath.Glob("snapshots/*.html")
	for _, path := range files {
		body, err := os.ReadFile(path)
		if err != nil {
			f.Fatalf("read seed %s: %v", path, err)
		}
		f.Add(body)
	}

	f.Fuzz(func(t *testing.T, body []byte) {
		_, _ = src.ParseURLs(bytes.NewReader(body))
		_, _ = src.ParseJobDetail(bytes.NewReader(body), "https://example.com/job")
	})
}

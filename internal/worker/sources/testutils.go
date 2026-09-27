package sources

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// SnapshotSource is implemented by any source that has snapshot-testable parsers.
// ParseURLs handles list pages; ParseJobDetail handles individual job pages.
// Both return []dto.Job so all snapshot fixtures share a single JSON schema.
type SnapshotSource interface {
	ParseURLs(r io.Reader) ([]dto.Job, error)
	ParseJobDetail(r io.Reader, url string) (dto.Job, error)
}

// RunSnapshotTests runs snapshot tests for every HTML file in snapshots/.
// list_*.html fixtures parse with ParseURLs; detail_*.html fixtures parse
// with ParseJobDetail, using the JSON fixture's first entry's URL.
func RunSnapshotTests(t *testing.T, src SnapshotSource) {
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

			if !reflect.DeepEqual(got, want) {
				t.Errorf("parser output does not match snapshot\ngot  %+v\nwant %+v", got, want)
			}
		})
	}
}

package wis_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ollymarsters/job-scraper/internal/sources"
	"github.com/ollymarsters/job-scraper/internal/sources/wis"
)

func TestSnapshots(t *testing.T) {
	files, _ := filepath.Glob("snapshots/*.html")
	if len(files) == 0 {
		t.Skip("no snapshots")
	}

	for _, htmlPath := range files {
		name := strings.TrimSuffix(filepath.Base(htmlPath), ".html")
		t.Run(name, func(t *testing.T) {
			f, err := os.Open(htmlPath)
			if err != nil {
				t.Fatalf("open html: %v", err)
			}
			defer f.Close()

			got, err := wis.ParseHTML(f)
			if err != nil {
				t.Fatalf("ParseHTML: %v", err)
			}

			jsonPath := filepath.Join("snapshots", name+".json")
			wantBytes, err := os.ReadFile(jsonPath)
			if err != nil {
				t.Fatalf("read snapshot json %s: %v (run `just snapshot-rebase wis` to create it)", jsonPath, err)
			}

			var want []sources.Job
			if err := json.Unmarshal(wantBytes, &want); err != nil {
				t.Fatalf("unmarshal snapshot json: %v", err)
			}

			if !reflect.DeepEqual(got, want) {
				t.Errorf("parser output does not match snapshot\ngot  %+v\nwant %+v", got, want)
			}
		})
	}
}

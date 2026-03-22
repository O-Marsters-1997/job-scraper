package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/sources/wis"
)

var parsers = map[string]func(io.Reader) ([]string, error){
	"wis": wis.ParseURLs,
}

func main() {
	if len(os.Args) < 3 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "download":
		if len(os.Args) != 5 {
			fmt.Fprintln(os.Stderr, "usage: snapshot download <source> <name> <url>")
			os.Exit(1)
		}
		if err := download(os.Args[2], os.Args[3], os.Args[4]); err != nil {
			fmt.Fprintf(os.Stderr, "download: %v\n", err)
			os.Exit(1)
		}
	case "rebase":
		if len(os.Args) != 3 {
			fmt.Fprintln(os.Stderr, "usage: snapshot rebase <source>")
			os.Exit(1)
		}
		if err := rebase(os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "rebase: %v\n", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(1)
	}
}

func download(source, name, url string) error {
	dir := snapshotDir(source)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}

	outPath := filepath.Join(dir, name+".html")
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return err
	}

	fmt.Printf("saved %s\n", outPath)
	return nil
}

func rebase(source string) error {
	parse, ok := parsers[source]
	if !ok {
		return fmt.Errorf("unknown source %q (known: %s)", source, knownSources())
	}

	dir := snapshotDir(source)
	htmlFiles, err := filepath.Glob(filepath.Join(dir, "*.html"))
	if err != nil {
		return err
	}
	if len(htmlFiles) == 0 {
		return fmt.Errorf("no html snapshots in %s", dir)
	}

	for _, htmlPath := range htmlFiles {
		name := strings.TrimSuffix(filepath.Base(htmlPath), ".html")

		f, err := os.Open(htmlPath)
		if err != nil {
			return fmt.Errorf("open %s: %w", htmlPath, err)
		}

		urls, err := parse(f)
		_ = f.Close()
		if err != nil {
			return fmt.Errorf("parse %s: %w", htmlPath, err)
		}

		data, err := json.MarshalIndent(urls, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal %s: %w", name, err)
		}

		jsonPath := filepath.Join(dir, name+".json")
		if err := os.WriteFile(jsonPath, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", jsonPath, err)
		}

		fmt.Printf("rebased %s (%d urls)\n", jsonPath, len(urls))
	}
	return nil
}

func snapshotDir(source string) string {
	_, filename, _, _ := runtime.Caller(0)
	moduleRoot := filepath.Join(filepath.Dir(filename), "../..")
	return filepath.Join(moduleRoot, "internal", "sources", source, "snapshots")
}

func knownSources() string {
	keys := make([]string, 0, len(parsers))
	for k := range parsers {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  snapshot download <source> <name> <url>")
	fmt.Fprintln(os.Stderr, "  snapshot rebase <source>")
}

package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/worker/discover"
)

const maxPageBytes = 2 << 20

func runResolveDomains(args []string) {
	if len(args) != 1 {
		usage()
	}
	data, err := os.ReadFile(filepath.Clean(args[0]))
	if err != nil {
		fatal("read domains file", err)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	get := func(ctx context.Context, url string) ([]byte, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxPageBytes))
		closeErr := resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("status %d", resp.StatusCode)
		}
		return body, errors.Join(err, closeErr)
	}

	var resolved, total int
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		domain := strings.TrimSpace(scanner.Text())
		if domain == "" {
			continue
		}
		total++
		board, ok, err := discover.ResolveDomain(context.Background(), get, domain)
		switch {
		case err != nil:
			fmt.Fprintf(os.Stderr, "%s: %v\n", domain, err)
			fmt.Printf("%s\t-\n", domain)
		case !ok:
			fmt.Printf("%s\t-\n", domain)
		default:
			resolved++
			fmt.Printf("%s\t%s:%s\n", domain, board.Source, board.Token)
		}
	}
	if err := scanner.Err(); err != nil {
		fatal("read domains file", err)
	}
	fmt.Printf("resolved %d/%d\n", resolved, total)
}

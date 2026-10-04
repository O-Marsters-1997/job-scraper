// Package commoncrawl harvests ATS board tokens from the newest Common Crawl
// indexes.
package commoncrawl

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
	"github.com/ollymarsters/job-scraper/internal/worker/discover"
)

// CollinfoURL lists Common Crawl indexes, newest first.
const CollinfoURL = "https://index.commoncrawl.org/collinfo.json"

const (
	userAgent      = "job-scraper-board-discovery"
	interval       = 30 * 24 * time.Hour
	crawls         = 3
	attempts       = 4
	defaultBackoff = time.Second
)

var patterns = []string{
	"jobs.ashbyhq.com/*",
	"boards.greenhouse.io/*",
	"job-boards.greenhouse.io/*",
	"job-boards.eu.greenhouse.io/*",
	"jobs.lever.co/*",
	"apply.workable.com/*",
	"*.recruitee.com",
	"*.jobs.personio.de",
}

var notBoards = map[string]map[string]bool{
	"workable":  {"j": true, "api": true},
	"recruitee": {"www": true, "app": true, "api": true, "support": true, "blog": true, "careers": true},
}

type Harvester struct {
	client      *http.Client
	collinfoURL string
	backoff     time.Duration
}

type Option func(*Harvester)

// WithBackoff sets the base delay before the first CDX retry; later retries double it.
func WithBackoff(base time.Duration) Option {
	return func(h *Harvester) { h.backoff = base }
}

func New(client *http.Client, collinfoURL string, opts ...Option) *Harvester {
	h := &Harvester{client: client, collinfoURL: collinfoURL, backoff: defaultBackoff}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func (h *Harvester) Name() string            { return "commoncrawl" }
func (h *Harvester) Interval() time.Duration { return interval }

func (h *Harvester) Harvest(ctx context.Context) (discover.Harvest, error) {
	cdxAPIs, err := h.newestCDX(ctx)
	if err != nil {
		return discover.Harvest{}, err
	}
	var out discover.Harvest
	seen := make(map[discover.Board]bool)
	var failed int
	var lastErr error
	for _, cdx := range cdxAPIs {
		for _, pattern := range patterns {
			if err := h.harvestPattern(ctx, cdx, pattern, seen, &out); err != nil {
				if ctx.Err() != nil {
					return discover.Harvest{}, ctx.Err()
				}
				failed++
				lastErr = err
			}
		}
	}
	if failed == len(cdxAPIs)*len(patterns) {
		return discover.Harvest{}, fmt.Errorf("every pattern failed: %w", lastErr)
	}
	out.Skipped += failed
	return out, nil
}

func (h *Harvester) harvestPattern(ctx context.Context, cdx, pattern string, seen map[discover.Board]bool, out *discover.Harvest) error {
	query := cdx + "?url=" + url.QueryEscape(pattern) + "&output=json&fl=url&filter=status:200"
	pages, err := h.numPages(ctx, query)
	if err != nil {
		return fmt.Errorf("pages for %s: %w", pattern, err)
	}
	for page := range pages {
		body, err := h.get(ctx, query+"&page="+strconv.Itoa(page))
		if err != nil {
			return fmt.Errorf("page %d of %s: %w", page, pattern, err)
		}
		collect(body, seen, out)
	}
	return nil
}

func (h *Harvester) newestCDX(ctx context.Context) ([]string, error) {
	body, err := discover.Get(ctx, h.client, h.collinfoURL, userAgent)
	if err != nil {
		return nil, err
	}
	var indexes []struct {
		CDXAPI string `json:"cdx-api"`
	}
	if err := json.Unmarshal(body, &indexes); err != nil {
		return nil, fmt.Errorf("decode collinfo: %w", err)
	}
	var apis []string
	for _, index := range indexes[:min(len(indexes), crawls)] {
		if index.CDXAPI != "" {
			apis = append(apis, index.CDXAPI)
		}
	}
	if len(apis) == 0 {
		return nil, errors.New("collinfo lists no index")
	}
	return apis, nil
}

func (h *Harvester) numPages(ctx context.Context, query string) (int, error) {
	body, err := h.get(ctx, query+"&showNumPages=true")
	if err != nil {
		return 0, err
	}
	var resp struct {
		Pages int `json:"pages"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("decode page count: %w", err)
	}
	return resp.Pages, nil
}

func (h *Harvester) get(ctx context.Context, target string) ([]byte, error) {
	delay := h.backoff
	for attempt := 1; ; attempt++ {
		body, retryable, err := h.fetch(ctx, target)
		if err == nil || !retryable || attempt == attempts {
			return body, err
		}
		wait := delay + rand.N(delay+1)
		delay *= 2
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
	}
}

func (h *Harvester) fetch(ctx context.Context, target string) (body []byte, retryable bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, false, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := h.client.Do(req)
	if err != nil {
		var netErr net.Error
		timedOut := errors.As(err, &netErr) && netErr.Timeout()
		return nil, timedOut && ctx.Err() == nil, fmt.Errorf("fetch %s: %w", target, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode >= 500, fmt.Errorf("fetch %s: status %s", target, resp.Status)
	}
	body, err = io.ReadAll(resp.Body)
	return body, false, err
}

func collect(body []byte, seen map[discover.Board]bool, out *discover.Harvest) {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(nil, 1<<20)
	for scanner.Scan() {
		var row struct {
			URL string `json:"url"`
		}
		if json.Unmarshal(scanner.Bytes(), &row) != nil {
			out.Skipped++
			continue
		}
		board, ok := resolve(row.URL)
		if !ok {
			out.Skipped++
			continue
		}
		if !seen[board] {
			seen[board] = true
			out.Boards = append(out.Boards, board)
		}
	}
}

func resolve(rawURL string) (discover.Board, bool) {
	source, token, ok := detect.ResolveBoard(rawURL)
	if !ok || notBoards[source][token] {
		return discover.Board{}, false
	}
	if role, known := sourcespec.SourceRole(source); !known || role != sourcespec.RoleATS {
		return discover.Board{}, false
	}
	if u, err := url.Parse(rawURL); err == nil && source == "greenhouse" {
		if forToken := u.Query().Get("for"); forToken != "" {
			token = forToken
		}
	}
	if !sourcespec.ValidBoardToken(token) {
		return discover.Board{}, false
	}
	return discover.Board{Source: source, Token: token}, true
}

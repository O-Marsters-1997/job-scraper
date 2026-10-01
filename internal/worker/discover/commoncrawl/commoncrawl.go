// Package commoncrawl harvests Ashby and Greenhouse board tokens from the
// latest Common Crawl index.
package commoncrawl

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	userAgent = "job-scraper-board-discovery"
	interval  = 30 * 24 * time.Hour
)

var patterns = []string{"jobs.ashbyhq.com/*", "boards.greenhouse.io/*"}

type Harvester struct {
	client      *http.Client
	collinfoURL string
}

func New(client *http.Client, collinfoURL string) *Harvester {
	return &Harvester{client: client, collinfoURL: collinfoURL}
}

func (h *Harvester) Name() string            { return "commoncrawl" }
func (h *Harvester) Interval() time.Duration { return interval }

func (h *Harvester) Harvest(ctx context.Context) (discover.Harvest, error) {
	cdx, err := h.latestCDX(ctx)
	if err != nil {
		return discover.Harvest{}, err
	}
	var out discover.Harvest
	seen := make(map[discover.Board]bool)
	for _, pattern := range patterns {
		query := cdx + "?url=" + url.QueryEscape(pattern) + "&output=json&fl=url&filter=status:200"
		pages, err := h.numPages(ctx, query)
		if err != nil {
			return discover.Harvest{}, fmt.Errorf("pages for %s: %w", pattern, err)
		}
		for page := range pages {
			body, err := discover.Get(ctx, h.client, query+"&page="+strconv.Itoa(page), userAgent)
			if err != nil {
				return discover.Harvest{}, fmt.Errorf("page %d of %s: %w", page, pattern, err)
			}
			collect(body, seen, &out)
		}
	}
	return out, nil
}

func (h *Harvester) latestCDX(ctx context.Context) (string, error) {
	body, err := discover.Get(ctx, h.client, h.collinfoURL, userAgent)
	if err != nil {
		return "", err
	}
	var indexes []struct {
		CDXAPI string `json:"cdx-api"`
	}
	if err := json.Unmarshal(body, &indexes); err != nil {
		return "", fmt.Errorf("decode collinfo: %w", err)
	}
	if len(indexes) == 0 || indexes[0].CDXAPI == "" {
		return "", errors.New("collinfo lists no index")
	}
	return indexes[0].CDXAPI, nil
}

func (h *Harvester) numPages(ctx context.Context, query string) (int, error) {
	body, err := discover.Get(ctx, h.client, query+"&showNumPages=true", userAgent)
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
	if !ok || source != "ashby" && source != "greenhouse" {
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

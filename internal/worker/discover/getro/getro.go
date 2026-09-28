// Package getro harvests VC-fund "portfolio jobs" boards built on Getro
// (getro.com), a platform powering 850+ fund job boards.
//
// Getro's SPA calls a same-origin /api/jobs XHR, but that endpoint is
// bot-gated (verified: returns HTTP 403 with a message directing scrapers to
// api.getro.com's paid partner API). The board's server-rendered HTML embeds
// the same first-page job/company data in a Next.js "__NEXT_DATA__" script
// tag, which is what this harvester parses instead.
package getro

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/worker/discover"
)

var seedBoards = []string{
	"https://jobsinvc.getro.com/jobs",
	"https://jobs.underscore.vc/jobs",
	"https://jobs.generalcatalyst.com/jobs",
	"https://jobs.accel.com/jobs",
}

var nextDataRe = regexp.MustCompile(`(?s)<script id="__NEXT_DATA__" type="application/json">(.*?)</script>`)

type Harvester struct {
	client *http.Client
	boards []string
}

func New() *Harvester {
	return &Harvester{client: &http.Client{Timeout: 20 * time.Second, Transport: logger.FetchTransport(nil)}, boards: seedBoards}
}

func (h *Harvester) WithBoards(boards []string) *Harvester {
	h.boards = boards
	return h
}

func (h *Harvester) Name() string { return "getro" }

// Harvest fetches each seed board independently; a fetch or parse failure on
// one board is logged and skipped so a single flaky fund site doesn't blank
// out the rest.
func (h *Harvester) Harvest(ctx context.Context) ([]discover.Company, error) {
	var all []discover.Company
	for _, board := range h.boards {
		body, err := h.fetch(ctx, board)
		if err != nil {
			slog.WarnContext(ctx, "getro: board fetch failed, skipping", slog.String(logger.KeyURL, board), slog.Any(logger.KeyErr, err))
			continue
		}
		companies, err := parseGetro(body)
		if err != nil {
			slog.WarnContext(ctx, "getro: board parse failed, skipping", slog.String(logger.KeyURL, board), slog.Any(logger.KeyErr, err))
			continue
		}
		all = append(all, companies...)
	}
	return all, nil
}

func (h *Harvester) fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	// A browser-like UA is required: Getro's SSR path serves a stripped shell
	// with no __NEXT_DATA__ payload to non-browser clients.
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; job-scraper-harvester/1.0)")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: status %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

type nextData struct {
	Props struct {
		PageProps struct {
			InitialState struct {
				Jobs struct {
					Found []getroJob `json:"found"`
				} `json:"jobs"`
			} `json:"initialState"`
		} `json:"pageProps"`
	} `json:"props"`
}

type getroJob struct {
	URL          string            `json:"url"`
	Organization getroOrganization `json:"organization"`
}

type getroOrganization struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func parseGetro(body []byte) ([]discover.Company, error) {
	m := nextDataRe.FindSubmatch(body)
	if m == nil {
		return nil, fmt.Errorf("getro: __NEXT_DATA__ script not found")
	}

	var data nextData
	if err := json.Unmarshal(m[1], &data); err != nil {
		return nil, fmt.Errorf("getro: unmarshal __NEXT_DATA__: %w", err)
	}

	seen := make(map[string]bool)
	var companies []discover.Company
	for _, job := range data.Props.PageProps.InitialState.Jobs.Found {
		org := job.Organization
		if org.Name == "" || seen[org.Slug] {
			continue
		}
		seen[org.Slug] = true

		c := discover.Company{Name: org.Name}
		if source, token, ok := detect.ResolveBoard(job.URL); ok {
			c.ATSSource = source
			c.ATSToken = token
		}
		companies = append(companies, c)
	}
	return companies, nil
}

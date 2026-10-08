package workablesearch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/slug"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
	"github.com/ollymarsters/job-scraper/internal/worker/discover"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

const (
	SearchURL = "https://jobs.workable.com/api/v1/jobs?location=United%20Kingdom"

	source    = "workable"
	userAgent = "Mozilla/5.0 (compatible; job-scraper/1.0)"
	interval  = 7 * 24 * time.Hour
	maxPages  = 500
	applyHost = "apply.workable.com"

	pageGap       = time.Second
	rateLimitWait = 30 * time.Second
	maxRateLimits = 3
)

type Harvester struct {
	client    *http.Client
	searchURL string
	pageGap   time.Duration
	backoff   time.Duration
}

func New(client *http.Client, searchURL string) *Harvester {
	return &Harvester{client: client, searchURL: searchURL, pageGap: pageGap, backoff: rateLimitWait}
}

func (h *Harvester) Name() string            { return "workablesearch" }
func (h *Harvester) Interval() time.Duration { return interval }

type page struct {
	NextPageToken string `json:"nextPageToken"`
	Jobs          []struct {
		URL     string `json:"url"`
		Company struct {
			Title string `json:"title"`
		} `json:"company"`
	} `json:"jobs"`
}

func (h *Harvester) Harvest(ctx context.Context) (discover.Harvest, error) {
	var out discover.Harvest
	seenBoards := map[string]bool{}
	seenCompanies := map[string]bool{}
	pageToken := ""
	for range maxPages {
		p, err := h.fetchPaced(ctx, pageToken)
		if err != nil {
			if len(seenBoards)+len(seenCompanies)+out.Skipped == 0 || ctx.Err() != nil {
				return discover.Harvest{}, err
			}
			slog.WarnContext(ctx, "workable search stopped early, keeping the pages fetched", slog.Any(logger.KeyErr, err))
			break
		}
		for _, j := range p.Jobs {
			if board, ok := applyBoard(j.URL); ok {
				if !seenBoards[board.Token] {
					seenBoards[board.Token] = true
					out.Boards = append(out.Boards, board)
				}
				continue
			}
			companySlug := slug.Make(j.Company.Title)
			if !sourcespec.ValidBoardToken(companySlug) {
				out.Skipped++
				continue
			}
			if !seenCompanies[companySlug] {
				seenCompanies[companySlug] = true
				out.Companies = append(out.Companies, discover.Company{
					Slug: companySlug, Name: j.Company.Title, Board: discover.Board{Source: source, Token: companySlug},
				})
			}
		}
		if p.NextPageToken == "" || p.NextPageToken == pageToken || len(p.Jobs) == 0 {
			break
		}
		pageToken = p.NextPageToken
	}
	return out, nil
}

func (h *Harvester) fetchPaced(ctx context.Context, pageToken string) (page, error) {
	if err := sleep(ctx, h.pageGap); err != nil {
		return page{}, err
	}
	for attempt := 1; ; attempt++ {
		p, err := h.fetch(ctx, pageToken)
		var status *sources.StatusError
		if !errors.As(err, &status) || status.Code != http.StatusTooManyRequests || attempt > maxRateLimits {
			return p, err
		}
		if err := sleep(ctx, time.Duration(attempt)*h.backoff); err != nil {
			return page{}, err
		}
	}
}

func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *Harvester) fetch(ctx context.Context, pageToken string) (page, error) {
	u := h.searchURL
	if pageToken != "" {
		u += "&pageToken=" + url.QueryEscape(pageToken)
	}
	body, err := discover.Get(ctx, h.client, u, userAgent)
	if err != nil {
		return page{}, err
	}
	var p page
	if err := json.Unmarshal(body, &p); err != nil {
		return page{}, fmt.Errorf("parse workable search: %w", err)
	}
	return p, nil
}

func applyBoard(jobURL string) (discover.Board, bool) {
	u, err := url.Parse(jobURL)
	if err != nil || u.Hostname() != applyHost {
		return discover.Board{}, false
	}
	src, token, ok := detect.ResolveBoard(jobURL)
	if !ok || src != source {
		return discover.Board{}, false
	}
	return discover.Board{Source: src, Token: token}, true
}

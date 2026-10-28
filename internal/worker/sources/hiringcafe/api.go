package hiringcafe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"

	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

const (
	origin          = "https://hiringcafe.com"
	sourceName      = "hiringcafe"
	apiAttempts     = 3
	apiTimeout      = 90 * time.Second
	errBodyBytes    = 200
	DecodoScrapeURL = "https://data.decodo.com/v1/scrape"
)

var errRefused = errors.New("request refused")

var _ Tab = (*API)(nil)

type API struct {
	endpoint string
	token    string
	gate     *sources.Gate
	client   *http.Client

	mu      sync.Mutex
	buildID string
}

func NewAPI(endpoint, token string, gate *sources.Gate) *API {
	return &API{endpoint: endpoint, token: token, gate: gate, client: &http.Client{Timeout: apiTimeout}}
}

func (a *API) Search(ctx context.Context, state json.RawMessage, page int) ([]byte, error) {
	var probe struct {
		Locations json.RawMessage `json:"locations"`
	}
	if err := json.Unmarshal(state, &probe); err != nil || len(probe.Locations) == 0 {
		return nil, errors.New("search state needs locations")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if page == 0 {
		return a.retry(ctx, func(int) ([]byte, error) { return a.loadClassic(ctx, state) })
	}
	return a.retry(ctx, func(attempt int) ([]byte, error) {
		if a.buildID == "" {
			if _, err := a.loadClassic(ctx, state); err != nil {
				return nil, err
			}
		}
		target := fmt.Sprintf("%s/_next/data/%s/classic.json?searchState=%s&page=%d", origin, a.buildID, url.QueryEscape(string(state)), page)
		content, err := a.scrape(ctx, target, attempt > 0)
		if errors.Is(err, sources.ErrGone) {
			err = fmt.Errorf("%w: %w", errRefused, err)
		}
		if err == nil {
			var env struct {
				PageProps json.RawMessage `json:"pageProps"`
			}
			if err = json.Unmarshal([]byte(jsonText(content)), &env); err == nil && len(env.PageProps) > 0 {
				return env.PageProps, nil
			}
			err = fmt.Errorf("%w: %s has no pageProps", errRefused, target)
		}
		a.buildID = ""
		return nil, err
	})
}

func (a *API) Description(ctx context.Context, id string) ([]byte, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.retry(ctx, func(int) ([]byte, error) {
		content, err := a.scrape(ctx, origin+"/api/job-description?id="+url.QueryEscape(id), false)
		if err != nil {
			return nil, err
		}
		var probe struct {
			Code string `json:"code"`
		}
		if err := json.Unmarshal([]byte(content), &probe); err != nil || probe.Code == "cloudflare_challenge" {
			return nil, fmt.Errorf("%w: description is not a job", errRefused)
		}
		return []byte(content), nil
	})
}

func (a *API) retry(ctx context.Context, op func(attempt int) ([]byte, error)) ([]byte, error) {
	var err error
	for attempt := range apiAttempts {
		var out []byte
		out, err = op(attempt)
		if !errors.Is(err, errRefused) {
			return out, err
		}
	}
	a.gate.Block(0)
	if waitErr := a.gate.Wait(ctx); waitErr != nil {
		return nil, fmt.Errorf("%w: %w", waitErr, err)
	}
	return nil, err
}

func (a *API) loadClassic(ctx context.Context, state json.RawMessage) ([]byte, error) {
	content, err := a.scrape(ctx, origin+"/classic?searchState="+url.QueryEscape(string(state)), true)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("parse classic page: %w", err)
	}
	var data struct {
		BuildID string `json:"buildId"`
		Props   struct {
			PageProps json.RawMessage `json:"pageProps"`
		} `json:"props"`
	}
	if err := json.Unmarshal([]byte(doc.Find("script#__NEXT_DATA__").Text()), &data); err != nil || data.BuildID == "" || len(data.Props.PageProps) == 0 {
		return nil, fmt.Errorf("%w: classic page has no next data", errRefused)
	}
	a.buildID = data.BuildID
	return data.Props.PageProps, nil
}

func (a *API) scrape(ctx context.Context, target string, render bool) (string, error) {
	if err := a.gate.Wait(ctx); err != nil {
		return "", err
	}
	body := map[string]string{"url": target, "geo": "GB", "proxy_pool": "premium", "session_id": uuid.NewString()}
	if render {
		body["headless"] = "html"
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("%w: %w", errRefused, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("%w: read response: %w", errRefused, err)
	}
	switch {
	case resp.StatusCode == http.StatusUnauthorized, resp.StatusCode == http.StatusForbidden:
		return "", fmt.Errorf("decodo rejected the scrape token: status %d", resp.StatusCode)
	case resp.StatusCode != http.StatusOK:
		return "", fmt.Errorf("%w: %s status %d body %q", errRefused, target, resp.StatusCode, raw[:min(len(raw), errBodyBytes)])
	}
	var out struct {
		Results []struct {
			Content    string `json:"content"`
			StatusCode int    `json:"status_code"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || len(out.Results) == 0 {
		return "", fmt.Errorf("%w: %s has no results", errRefused, target)
	}
	switch res := out.Results[0]; res.StatusCode {
	case 0, http.StatusOK:
	case http.StatusNotFound:
		return "", fmt.Errorf("%w: %s", sources.ErrGone, target)
	default:
		return "", fmt.Errorf("%w: %s answered %d", errRefused, target, res.StatusCode)
	}
	return out.Results[0].Content, nil
}

func jsonText(content string) string {
	if strings.HasPrefix(strings.TrimSpace(content), "<") {
		if doc, err := goquery.NewDocumentFromReader(strings.NewReader(content)); err == nil {
			return doc.Find("body").Text()
		}
	}
	return content
}

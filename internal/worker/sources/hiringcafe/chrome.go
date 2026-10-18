package hiringcafe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"

	"github.com/ollymarsters/job-scraper/internal/worker/proxy"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

const (
	origin           = "https://hiringcafe.com"
	sourceName       = "hiringcafe"
	clearTimeout     = 45 * time.Second
	clearPoll        = time.Second
	fetchScript      = `async (u) => { const r = await fetch(u, {credentials: 'include'}); return {status: r.status, type: r.headers.get('content-type') ?? '', mitigated: r.headers.get('cf-mitigated') ?? '', server: r.headers.get('server') ?? '', body: await r.text()}; }`
	refusalBodyBytes = 200
	bodyTextScript   = `() => document.body?.innerText ?? ''`

	cloudflareChallengeCode = "cloudflare_challenge"
	nextDataScript          = `() => document.querySelector('script#__NEXT_DATA__')?.textContent ?? ''`
)

var errChallenged = errors.New("cloudflare challenge")

type Stats struct {
	Challenges int
	Rotations  int
}

var _ Tab = (*Chrome)(nil)

// Chrome is a Tab backed by one stealth Chrome page behind a residential Tunnel. It launches on
// first use and lives until the worker exits, when go-rod's launcher kills the browser.
type Chrome struct {
	gate *sources.Gate

	mu      sync.Mutex
	tunnel  *proxy.Tunnel
	browser *rod.Browser
	page    *rod.Page
	buildID string
	state   json.RawMessage
	stats   Stats
}

func NewChrome(gate *sources.Gate) *Chrome {
	return &Chrome{gate: gate}
}

func (c *Chrome) Stats() Stats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stats
}

func (c *Chrome) Search(ctx context.Context, state json.RawMessage, page int) ([]byte, error) {
	var probe struct {
		Locations json.RawMessage `json:"locations"`
	}
	if err := json.Unmarshal(state, &probe); err != nil || len(probe.Locations) == 0 {
		return nil, errors.New("search state needs locations")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = state

	if page == 0 {
		return c.recoverWith(ctx, nil)
	}
	return c.recoverWith(ctx, func() ([]byte, error) {
		if c.page == nil {
			return nil, errChallenged
		}
		path := fmt.Sprintf("/_next/data/%s/classic.json?searchState=%s&page=%d", c.buildID, url.QueryEscape(string(state)), page)
		body, err := c.fetch(ctx, path)
		if err != nil {
			return nil, err
		}
		var env struct {
			PageProps json.RawMessage `json:"pageProps"`
		}
		if err := json.Unmarshal(body, &env); err != nil || len(env.PageProps) == 0 {
			return nil, fmt.Errorf("%w: %s has no pageProps", errChallenged, path)
		}
		return env.PageProps, nil
	})
}

func (c *Chrome) Description(ctx context.Context, id string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state == nil {
		return nil, errors.New("description before search")
	}
	return c.recoverWith(ctx, func() ([]byte, error) {
		if c.page == nil {
			return nil, errChallenged
		}
		return c.document(ctx, "/api/job-description?id="+url.QueryEscape(id))
	})
}

func (c *Chrome) recoverWith(ctx context.Context, op func() ([]byte, error)) ([]byte, error) {
	first := op
	if op == nil {
		first = func() ([]byte, error) { return c.load(ctx, c.state) }
	}
	out, err := first()
	if !errors.Is(err, errChallenged) {
		return out, err
	}
	for attempt := range 2 {
		c.stats.Challenges++
		if attempt == 1 {
			if err := c.rotate(ctx); err != nil {
				return nil, err
			}
		}
		out, err = c.load(ctx, c.state)
		if err == nil && op != nil {
			out, err = op()
			if errors.Is(err, errChallenged) {
				err = fmt.Errorf("refused on a freshly loaded page: %w", err)
				break
			}
		}
		if !errors.Is(err, errChallenged) {
			return out, err
		}
	}
	c.stats.Challenges++
	c.gate.Block(0)
	if waitErr := c.gate.Wait(ctx); waitErr != nil {
		return nil, fmt.Errorf("%w: %w", waitErr, err)
	}
	return nil, errors.New("hiring.cafe challenge persists")
}

func (c *Chrome) rotate(ctx context.Context) error {
	c.stats.Rotations++
	c.tunnel.Rotate(ctx, "cloudflare challenge")
	browser := c.browser
	c.browser, c.page, c.buildID = nil, nil, ""
	if browser == nil {
		return nil
	}
	if err := browser.Close(); err != nil {
		return fmt.Errorf("close browser: %w", err)
	}
	return nil
}

func (c *Chrome) load(ctx context.Context, state json.RawMessage) ([]byte, error) {
	if err := c.launch(); err != nil { //nolint:contextcheck // NewTunnel's frozen signature (#1020) takes no context
		return nil, err
	}
	if err := c.gate.Wait(ctx); err != nil {
		return nil, err
	}
	page := c.page.Context(ctx)
	if err := page.Navigate(origin + "/classic?searchState=" + url.QueryEscape(string(state))); err != nil {
		return nil, fmt.Errorf("navigate: %w", err)
	}
	deadline := time.Now().Add(clearTimeout)
	for {
		res, err := page.Eval(nextDataScript)
		if err == nil && res.Value.Str() != "" {
			var data struct {
				BuildID string `json:"buildId"`
				Props   struct {
					PageProps json.RawMessage `json:"pageProps"`
				} `json:"props"`
			}
			if err := json.Unmarshal([]byte(res.Value.Str()), &data); err != nil {
				return nil, fmt.Errorf("decode next data: %w", err)
			}
			c.buildID = data.BuildID
			return data.Props.PageProps, nil
		}
		if time.Now().After(deadline) {
			return nil, errChallenged
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(clearPoll):
		}
	}
}

func (c *Chrome) document(ctx context.Context, path string) ([]byte, error) {
	if err := c.gate.Wait(ctx); err != nil {
		return nil, err
	}
	page := c.page.Context(ctx)
	if err := page.Navigate(origin + path); err != nil {
		return nil, fmt.Errorf("navigate: %w", err)
	}
	deadline := time.Now().Add(clearTimeout)
	var seen string
	for {
		res, err := page.Eval(bodyTextScript)
		if err == nil {
			seen = res.Value.Str()
			var probe struct {
				Code string `json:"code"`
			}
			if json.Unmarshal([]byte(seen), &probe) == nil {
				if probe.Code == cloudflareChallengeCode {
					return nil, fmt.Errorf("%w: %s answered with the json challenge %q", errChallenged, path, seen[:min(len(seen), refusalBodyBytes)])
				}
				return []byte(seen), nil
			}
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("%w: %s not json after %s body %q", errChallenged, path, clearTimeout, seen[:min(len(seen), refusalBodyBytes)])
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(clearPoll):
		}
	}
}

func (c *Chrome) fetch(ctx context.Context, path string) ([]byte, error) {
	if err := c.gate.Wait(ctx); err != nil {
		return nil, err
	}
	res, err := c.page.Context(ctx).Evaluate(rod.Eval(fetchScript, path).ByPromise())
	if err != nil {
		return nil, fmt.Errorf("fetch in page: %w", err)
	}
	status := res.Value.Get("status").Int()
	body := res.Value.Get("body").Str()
	if status == 200 {
		return []byte(body), nil
	}
	refusal := fmt.Sprintf("%s status %d content-type %q cf-mitigated %q server %q body %q",
		path, status, res.Value.Get("type").Str(), res.Value.Get("mitigated").Str(),
		res.Value.Get("server").Str(), body[:min(len(body), refusalBodyBytes)])
	switch {
	case status == 403, status == 429, status == 503, status == 404 && strings.HasPrefix(path, "/_next/"):
		return nil, fmt.Errorf("%w: %s", errChallenged, refusal)
	default:
		return nil, fmt.Errorf("hiring.cafe %s", refusal)
	}
}

func (c *Chrome) launch() error {
	if c.page != nil {
		return nil
	}
	if c.tunnel == nil {
		t, err := proxy.NewRestrictedTunnel(sourceName, "hiringcafe.com", "challenges.cloudflare.com")
		if err != nil {
			return fmt.Errorf("start tunnel: %w", err)
		}
		c.tunnel = t
	}
	bin, _ := launcher.LookPath()
	wsURL, err := launcher.New().Bin(bin).Headless(true).Proxy("http://"+c.tunnel.Addr()).
		Set("disable-component-update").
		Set("disable-features", "site-per-process", "TranslateUI", "OptimizationHints", "OptimizationGuideModelDownloading", "AutofillServerCommunication").
		Launch()
	if err != nil {
		return fmt.Errorf("launch chrome: %w", err)
	}
	browser := rod.New().ControlURL(wsURL)
	if err := browser.Connect(); err != nil {
		return errors.Join(fmt.Errorf("connect chrome: %w", err), browser.Close())
	}
	page, err := stealth.Page(browser)
	if err != nil {
		return errors.Join(fmt.Errorf("open stealth page: %w", err), browser.Close())
	}
	router := page.HijackRequests()
	if err := router.Add("*", "", blockHeavy); err != nil {
		return errors.Join(fmt.Errorf("hijack requests: %w", err), browser.Close())
	}
	go router.Run()
	c.browser, c.page = browser, page
	return nil
}

func blockHeavy(h *rod.Hijack) {
	blocked := []proto.NetworkResourceType{
		proto.NetworkResourceTypeImage, proto.NetworkResourceTypeFont, proto.NetworkResourceTypeMedia,
	}
	if slices.Contains(blocked, h.Request.Type()) {
		h.Response.Fail(proto.NetworkErrorReasonBlockedByClient)
		return
	}
	h.ContinueRequest(&proto.FetchContinueRequest{})
}

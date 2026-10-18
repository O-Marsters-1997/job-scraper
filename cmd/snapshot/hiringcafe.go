package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	dto "github.com/prometheus/client_model/go"

	"github.com/ollymarsters/job-scraper/internal/worker/proxy"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/hiringcafe"
)

const (
	spikeDuration = time.Hour
	spikeGap      = 3 * time.Second
	spikeBlock    = 5 * time.Minute
)

func hiringcafeSpike(searchURL string, d time.Duration, out io.Writer) error {
	u, err := url.Parse(searchURL)
	if err != nil {
		return fmt.Errorf("parse search url: %w", err)
	}
	state := json.RawMessage(u.Query().Get("searchState"))
	if len(state) == 0 {
		return errors.New("search url has no searchState")
	}

	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()

	chrome := hiringcafe.NewChrome(sources.NewGate(spikeGap, spikeBlock))
	var pages, descriptions int
	for ctx.Err() == nil {
		n, descs, err := sweep(ctx, chrome, state)
		pages += n
		descriptions += descs
		var deferred *sources.DeferredError
		switch {
		case errors.As(err, &deferred):
			if _, err := fmt.Fprintf(out, "deferred until %s\n", deferred.Until.Format(time.RFC3339)); err != nil {
				return err
			}
			select {
			case <-ctx.Done():
			case <-time.After(time.Until(deferred.Until)):
			}
		case err != nil && ctx.Err() == nil:
			return err
		}
	}

	stats := chrome.Stats()
	_, err = fmt.Fprintf(out, "pages fetched: %d\ndescriptions fetched: %d\nchallenges met: %d\nrotations: %d\ndecodo bytes: %d\n",
		pages, descriptions, stats.Challenges, stats.Rotations, wireBytes())
	return err
}

func sweep(ctx context.Context, tab hiringcafe.Tab, state json.RawMessage) (pages, descriptions int, err error) {
	for page := 0; ctx.Err() == nil; page++ {
		props, err := tab.Search(ctx, state, page)
		if err != nil {
			return pages, descriptions, err
		}
		pages++
		var tree any
		if err := json.Unmarshal(props, &tree); err != nil {
			return pages, descriptions, fmt.Errorf("decode pageProps: %w", err)
		}
		ids := hitIDs(tree)
		if len(ids) == 0 {
			return pages, descriptions, nil
		}
		for _, id := range ids {
			if _, err := tab.Description(ctx, id); err != nil {
				return pages, descriptions, err
			}
			descriptions++
		}
	}
	return pages, descriptions, nil
}

func hitIDs(tree any) []string {
	switch v := tree.(type) {
	case []any:
		var ids []string
		for _, e := range v {
			if m, ok := e.(map[string]any); ok {
				if id, ok := m["id"].(string); ok {
					ids = append(ids, id)
				}
			}
		}
		if len(ids) > 0 {
			return ids
		}
		for _, e := range v {
			if ids := hitIDs(e); len(ids) > 0 {
				return ids
			}
		}
	case map[string]any:
		for _, e := range v {
			if ids := hitIDs(e); len(ids) > 0 {
				return ids
			}
		}
	}
	return nil
}

func wireBytes() int64 {
	var total float64
	for _, dir := range []string{"read", "write"} {
		var m dto.Metric
		if err := proxy.FetchWireBytes.WithLabelValues("hiringcafe", "residential", dir).Write(&m); err == nil {
			total += m.GetCounter().GetValue()
		}
	}
	return int64(total)
}

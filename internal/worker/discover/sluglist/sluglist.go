// Package sluglist harvests ATS board tokens from public company slug lists.
package sluglist

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/ollymarsters/job-scraper/internal/detect"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
	"github.com/ollymarsters/job-scraper/internal/worker/discover"
)

type Format int

const (
	CSV       Format = iota // header name,slug,url (kalil0321)
	JSONSlugs               // JSON array of slugs (Feashliaa)
)

type List struct {
	Source string
	URL    string
	Format Format
}

const (
	name      = "sluglist"
	userAgent = "Mozilla/5.0 (compatible; job-scraper/1.0)"
	interval  = 7 * 24 * time.Hour

	kalilBase = "https://raw.githubusercontent.com/kalil0321/ats-scrapers/main/ats-companies/"
	feashBase = "https://raw.githubusercontent.com/Feashliaa/job-board-aggregator/main/data/"
)

// Lists covers the 9 supported ATSes in kalil0321 and greenhouse, lever and ashby in Feashliaa.
var Lists = buildLists()

func buildLists() []List {
	var out []List
	for _, s := range []string{"greenhouse", "lever", "ashby", "workable", "recruitee", "personio", "pinpoint", "teamtailor", "smartrecruiters"} {
		out = append(out, List{Source: s, URL: kalilBase + s + ".csv", Format: CSV})
	}
	for _, s := range []string{"greenhouse", "lever", "ashby"} {
		out = append(out, List{Source: s, URL: feashBase + s + "_companies.json", Format: JSONSlugs})
	}
	return out
}

type Harvester struct {
	client *http.Client
	lists  []List
}

func New(client *http.Client, lists []List) *Harvester {
	return &Harvester{client: client, lists: lists}
}

func (h *Harvester) Name() string            { return name }
func (h *Harvester) Interval() time.Duration { return interval }

func (h *Harvester) Harvest(ctx context.Context) (discover.Harvest, error) {
	var out discover.Harvest
	seen := make(map[discover.Board]bool)
	var errs []error
	for _, l := range h.lists {
		body, err := discover.Get(ctx, h.client, l.URL, userAgent)
		if err == nil {
			err = collect(&out, seen, l, body)
		}
		if err != nil {
			if ctx.Err() != nil {
				return discover.Harvest{}, ctx.Err()
			}
			slog.WarnContext(ctx, "sluglist: skipping list", slog.String(logger.KeySource, l.Source), slog.String("url", l.URL), slog.Any(logger.KeyErr, err))
			errs = append(errs, err)
		}
	}
	if len(h.lists) > 0 && len(errs) == len(h.lists) {
		return discover.Harvest{}, errors.Join(errs...)
	}
	return out, nil
}

func collect(out *discover.Harvest, seen map[discover.Board]bool, l List, body []byte) error {
	tokens, err := parse(l, body)
	if err != nil {
		return err
	}
	for _, token := range tokens {
		b := discover.Board{Source: l.Source, Token: token}
		if !sourcespec.ValidBoardToken(token) || seen[b] {
			out.Skipped++
			continue
		}
		seen[b] = true
		out.Boards = append(out.Boards, b)
	}
	return nil
}

func parse(l List, body []byte) ([]string, error) {
	switch l.Format {
	case JSONSlugs:
		var tokens []string
		if err := json.Unmarshal(body, &tokens); err != nil {
			return nil, fmt.Errorf("parse %s: %w", l.URL, err)
		}
		return tokens, nil
	case CSV:
		r := csv.NewReader(bytes.NewReader(body))
		r.FieldsPerRecord = -1
		rows, err := r.ReadAll()
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("parse %s: %w", l.URL, err)
		}
		var tokens []string
		for i, row := range rows {
			if i == 0 || len(row) < 2 {
				continue
			}
			tokens = append(tokens, csvToken(l.Source, row))
		}
		return tokens, nil
	default:
		return nil, fmt.Errorf("parse %s: unknown format %d", l.URL, l.Format)
	}
}

func csvToken(source string, row []string) string {
	if len(row) > 2 {
		if s, token, ok := detect.ResolveBoard(row[2]); ok && s == source {
			return token
		}
	}
	return row[1]
}

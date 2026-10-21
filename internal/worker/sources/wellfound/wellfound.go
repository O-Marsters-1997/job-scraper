package wellfound

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/slug"
	"github.com/ollymarsters/job-scraper/internal/worker/proxy"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

const (
	sourceName = "wellfound"
	baseURL    = "https://wellfound.com"

	gap        = 3 * time.Second
	blockedFor = 5 * time.Minute
	resultsKey = "seoLandingPageJobSearchResults("
)

var nextDataRe = regexp.MustCompile(`(?s)<script id="__NEXT_DATA__"[^>]*>(.*?)</script>`)

var browserHeader = http.Header{
	"User-Agent":                {"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/153.0.0.0 Safari/537.36"},
	"Accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"},
	"Sec-Ch-Ua":                 {`"Google Chrome";v="153", "Not_A Brand";v="8", "Chromium";v="153"`},
	"Sec-Ch-Ua-Mobile":          {"?0"},
	"Sec-Ch-Ua-Platform":        {`"macOS"`},
	"Sec-Fetch-Dest":            {"document"},
	"Sec-Fetch-Mode":            {"navigate"},
	"Sec-Fetch-Site":            {"none"},
	"Sec-Fetch-User":            {"?1"},
	"Upgrade-Insecure-Requests": {"1"},
}

type Search struct {
	Role     string
	Location string
}

type Scraper struct {
	sources.PaginatedBase
	search Search
	gate   *sources.Gate
}

var _ sources.Source = (*Scraper)(nil)

func NewGate() *sources.Gate { return sources.NewGate(gap, blockedFor) }

func New(search Search, gate *sources.Gate) *Scraper {
	return &Scraper{PaginatedBase: newBase(), search: search, gate: gate}
}

func newBase() sources.PaginatedBase {
	return sources.NewBase(sources.Config{
		Name:    sourceName,
		Route:   sources.RouteResidential,
		Header:  browserHeader,
		Cookies: true,
	})
}

func (s Search) pageURL(page int) (string, error) {
	var path string
	switch {
	case s.Role != "" && s.Location != "":
		path = "/role/l/" + url.PathEscape(s.Role) + "/" + url.PathEscape(s.Location)
	case s.Role != "":
		path = "/role/" + url.PathEscape(s.Role)
	case s.Location != "":
		path = "/location/" + url.PathEscape(s.Location)
	default:
		return "", errors.New("wellfound search needs a role or a location")
	}
	if page > 1 {
		path += "?page=" + strconv.Itoa(page)
	}
	return baseURL + path, nil
}

func (s *Scraper) FetchPage(ctx context.Context, cursor string) ([]dto.Job, string, error) {
	page := 1
	if cursor != "" {
		n, err := strconv.Atoi(cursor)
		if err != nil || n < 1 {
			return nil, "", fmt.Errorf("invalid wellfound cursor %q", cursor)
		}
		page = n
	}
	pageURL, err := s.search.pageURL(page)
	if err != nil {
		return nil, "", err
	}
	if err := s.gate.Wait(ctx); err != nil {
		return nil, "", err
	}
	body, err := s.Get(ctx, pageURL)
	if err != nil {
		return nil, "", s.blocked(err)
	}
	jobs, pageCount, err := parse(bytes.NewReader(body))
	if err != nil {
		return nil, "", s.blocked(err)
	}
	if page >= pageCount {
		return jobs, "", nil
	}
	return jobs, strconv.Itoa(page + 1), nil
}

func (s *Scraper) blocked(err error) error {
	if errors.Is(err, proxy.ErrBlocked) {
		s.gate.Block(blockedFor)
	}
	return err
}

func (s *Scraper) ParseURLs(r io.Reader) ([]dto.Job, error) {
	jobs, _, err := parse(r)
	return jobs, err
}

func (s *Scraper) ParseJobDetail(io.Reader, string) (dto.Job, error) {
	return dto.Job{}, errors.New("wellfound cards are complete jobs and have no detail page")
}

type ref struct {
	Ref string `json:"__ref"`
}

type results struct {
	PageCount int   `json:"pageCount"`
	Startups  []ref `json:"startups"`
}

type startup struct {
	Name     string `json:"name"`
	Listings []ref  `json:"highlightedJobListings"`
}

type listing struct {
	ID           string   `json:"id"`
	Slug         string   `json:"slug"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	ATSSource    string   `json:"atsSource"`
	Compensation string   `json:"compensation"`
	LiveStartAt  int64    `json:"liveStartAt"`
	LocationName []string `json:"locationNames"`
	Remote       bool     `json:"remote"`
	RemoteConfig *struct {
		Kind string `json:"kind"`
	} `json:"remoteConfig"`
}

func parse(r io.Reader) ([]dto.Job, int, error) {
	var jobs []dto.Job
	pageCount, err := walk(r, func(st startup, l listing) { jobs = append(jobs, card(st, l)) })
	if err != nil {
		return nil, 0, err
	}
	return jobs, pageCount, nil
}

type Listing struct {
	Company, ATSSource, URL string
}

func ParseATSListings(r io.Reader) ([]Listing, int, error) {
	var out []Listing
	pageCount, err := walk(r, func(st startup, l listing) {
		if l.ATSSource != "" {
			out = append(out, Listing{Company: st.Name, ATSSource: l.ATSSource, URL: listingURL(l)})
		}
	})
	if err != nil {
		return nil, 0, err
	}
	return out, pageCount, nil
}

func LocationPageURL(location string, page int) (string, error) {
	return Search{Location: location}.pageURL(page)
}

func NewPages(gate *sources.Gate) func(ctx context.Context, url string) ([]byte, error) {
	base := newBase()
	return func(ctx context.Context, url string) ([]byte, error) {
		if err := gate.Wait(ctx); err != nil {
			return nil, err
		}
		body, err := base.Get(ctx, url)
		if errors.Is(err, proxy.ErrBlocked) {
			gate.Block(blockedFor)
		}
		return body, err
	}
}

func walk(r io.Reader, visit func(startup, listing)) (int, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return 0, fmt.Errorf("read page: %w", err)
	}
	m := nextDataRe.FindSubmatch(raw)
	if m == nil {
		return 0, fmt.Errorf("page has no __NEXT_DATA__, likely a DataDome challenge: %w", proxy.ErrBlocked)
	}
	var next struct {
		Props struct {
			PageProps struct {
				ApolloState struct {
					Data map[string]json.RawMessage `json:"data"`
				} `json:"apolloState"`
			} `json:"pageProps"`
		} `json:"props"`
	}
	if err := json.Unmarshal(m[1], &next); err != nil {
		return 0, fmt.Errorf("decode __NEXT_DATA__: %w", err)
	}
	data := next.Props.PageProps.ApolloState.Data

	var root struct {
		Talent map[string]json.RawMessage `json:"talent"`
	}
	if err := json.Unmarshal(data["ROOT_QUERY"], &root); err != nil {
		return 0, fmt.Errorf("decode ROOT_QUERY: %w", err)
	}
	var res results
	found := false
	for k, v := range root.Talent {
		if !strings.HasPrefix(k, resultsKey) {
			continue
		}
		if err := json.Unmarshal(v, &res); err != nil {
			return 0, fmt.Errorf("decode search results: %w", err)
		}
		found = true
		break
	}
	if !found {
		return 0, errors.New("search results not found in __NEXT_DATA__")
	}

	for _, sref := range res.Startups {
		var st startup
		if err := json.Unmarshal(data[sref.Ref], &st); err != nil {
			return 0, fmt.Errorf("decode %s: %w", sref.Ref, err)
		}
		for _, lref := range st.Listings {
			var l listing
			if err := json.Unmarshal(data[lref.Ref], &l); err != nil {
				return 0, fmt.Errorf("decode %s: %w", lref.Ref, err)
			}
			visit(st, l)
		}
	}
	return res.PageCount, nil
}

func card(st startup, l listing) dto.Job {
	job := dto.Job{
		Title:           l.Title,
		Location:        strings.Join(l.LocationName, ", "),
		URL:             listingURL(l),
		CompanySlug:     slug.Company(st.Name),
		Source:          sourceName,
		Description:     l.Description,
		SalaryRaw:       l.Compensation,
		WorkArrangement: workArrangement(l),
	}
	if l.LiveStartAt > 0 {
		t := time.Unix(l.LiveStartAt, 0).UTC()
		job.PostedAt = &t
	}
	return job
}

func listingURL(l listing) string {
	return fmt.Sprintf("%s/jobs/%s-%s", baseURL, l.ID, l.Slug)
}

func workArrangement(l listing) string {
	if l.RemoteConfig != nil {
		switch l.RemoteConfig.Kind {
		case "REMOTE":
			return "remote"
		case "ONSITE_OR_REMOTE":
			return "hybrid"
		case "ONSITE":
			return "onsite"
		}
	}
	if l.Remote {
		return "remote"
	}
	return ""
}

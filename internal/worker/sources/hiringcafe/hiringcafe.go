package hiringcafe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/slug"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

const (
	requestGap = 3 * time.Second
	blockedFor = 5 * time.Minute
	recencyPad = time.Hour
	recencyKey = "dateFetchedPastNDays"
	sortKey    = "sortBy"
	day        = 24 * time.Hour
)

func NewGate() *sources.Gate { return sources.NewGate(requestGap, blockedFor) }

type Scraper struct {
	tab   Tab
	state json.RawMessage
}

var (
	_ sources.Source        = (*Scraper)(nil)
	_ sources.DetailFetcher = (*Scraper)(nil)
)

// Recency returns state sorted newest first, with dateFetchedPastNDays narrowed to the whole
// days since lastSucceeded plus an hour, never wider than the pasted value. A state that is
// not a JSON object is returned unchanged.
func Recency(state json.RawMessage, lastSucceeded *time.Time, now time.Time) json.RawMessage {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(state, &fields); err != nil || fields == nil {
		return state
	}
	fields[sortKey] = json.RawMessage(`"date"`)
	if lastSucceeded != nil {
		elapsed := max(now.Sub(*lastSucceeded), 0) + recencyPad
		days := max(int((elapsed+day-1)/day), 1)
		var configured int
		if err := json.Unmarshal(fields[recencyKey], &configured); err != nil || configured <= 0 || days < configured {
			fields[recencyKey] = json.RawMessage(strconv.Itoa(days))
		}
	}
	out, err := json.Marshal(fields)
	if err != nil {
		return state
	}
	return out
}

func New(state json.RawMessage, tab Tab) *Scraper {
	return &Scraper{tab: tab, state: state}
}

func (s *Scraper) Cfg() sources.Config {
	return sources.Config{Name: sourceName, Route: sources.RouteResidential, NewestFirst: true}
}

func (s *Scraper) FetchPage(ctx context.Context, cursor string) ([]dto.Job, string, error) {
	page := 0
	if cursor != "" {
		n, err := strconv.Atoi(cursor)
		if err != nil || n < 0 {
			return nil, "", fmt.Errorf("hiringcafe: bad cursor %q", cursor)
		}
		page = n
	}
	body, err := s.tab.Search(ctx, s.state, page)
	if err != nil {
		return nil, "", fmt.Errorf("hiringcafe: search: %w", err)
	}
	var props pageProps
	if err := json.Unmarshal(body, &props); err != nil {
		return nil, "", fmt.Errorf("hiringcafe: parse page props: %w", err)
	}
	if props.Error != nil {
		return nil, "", fmt.Errorf("hiringcafe: search error: %v", props.Error)
	}
	next := strconv.Itoa(page + 1)
	if props.IsLastPage {
		next = ""
	}
	return cards(props.Hits), next, nil
}

func (s *Scraper) GetDetails(ctx context.Context, card dto.Job) (dto.Job, error) {
	if card.ProviderPostingID == "" {
		return dto.Job{}, errors.New("hiringcafe: card has no posting id")
	}
	body, err := s.tab.Description(ctx, card.ProviderPostingID)
	if err != nil {
		return dto.Job{}, fmt.Errorf("hiringcafe: description: %w", err)
	}
	var d description
	if err := json.Unmarshal(body, &d); err != nil {
		return dto.Job{}, fmt.Errorf("hiringcafe: parse description: %w", err)
	}
	card.Description = d.Job.Info.Description
	return card, nil
}

func cards(hits []hit) []dto.Job {
	seen := map[string]bool{}
	jobs := make([]dto.Job, 0, len(hits))
	for _, h := range hits {
		if h.ID == "" || h.ApplyURL == "" || h.IsExpired {
			continue
		}
		if h.CollapseKey != "" {
			if seen[h.CollapseKey] {
				continue
			}
			seen[h.CollapseKey] = true
		}
		jobs = append(jobs, card(h))
	}
	return jobs
}

func card(h hit) dto.Job {
	p := h.Processed
	url := h.ApplyURL
	var postedAt *time.Time
	if t, err := time.Parse(time.RFC3339, p.PublishDate); err == nil {
		t = t.UTC()
		postedAt = &t
	} else {
		sources.WarnDefaulted(sourceName, "PostedAt", url)
	}
	return dto.Job{
		Title:             h.Info.Title,
		Location:          p.Location,
		URL:               url,
		ApplyURL:          url,
		CompanySlug:       slug.Company(h.Company.Name),
		ProviderPostingID: h.ID,
		SalaryRaw:         salary(h),
		WorkArrangement:   sources.DetectWorkArrangement(p.WorkplaceType),
		PostedAt:          postedAt,
	}
}

func salary(h hit) string {
	p := h.Processed
	if !p.SalaryTransparent {
		return ""
	}
	symbol := map[string]string{"GBP": "£", "USD": "$", "EUR": "€"}[p.Currency]
	if symbol == "" {
		symbol = p.Currency + " "
	}
	var amounts []string
	for _, v := range []*float64{p.YearlyMin, p.YearlyMax} {
		if v != nil {
			amounts = append(amounts, fmt.Sprintf("%s%.0f", symbol, *v))
		}
	}
	if len(amounts) == 0 {
		return ""
	}
	return strings.Join(amounts, " - ") + " per year"
}

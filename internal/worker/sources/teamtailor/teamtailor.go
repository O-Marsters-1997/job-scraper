package teamtailor

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

// New builds a Teamtailor source for one company subdomain (e.g. "acmecorp").
// The feed returns 100 items unless per_page is given.
func New(token string) *sources.BoardSource {
	return sources.NewBoardSource(sources.BoardSpec{
		Name:        "teamtailor",
		URL:         fmt.Sprintf("https://%s.teamtailor.com/jobs.rss?per_page=1000", token),
		CompanySlug: token,
		Parse:       parse,
		Count:       count,
	})
}

type feed struct {
	Title string `xml:"channel>title"`
	Items []item `xml:"channel>item"`
}

type item struct {
	Title        string     `xml:"title"`
	Description  string     `xml:"description"`
	Link         string     `xml:"link"`
	PubDate      string     `xml:"pubDate"`
	GUID         string     `xml:"guid"`
	RemoteStatus string     `xml:"remoteStatus"`
	Locations    []location `xml:"https://teamtailor.com/locations locations>location"`
}

type location struct {
	Name    string `xml:"name"`
	City    string `xml:"city"`
	Country string `xml:"country"`
}

func (l location) label() string {
	if l.City != "" {
		return l.City
	}
	return l.Name
}

func (i item) location() string {
	var labels []string
	for _, l := range i.Locations {
		if label := l.label(); label != "" {
			labels = append(labels, label)
		}
	}
	return strings.Join(labels, "; ")
}

func (i item) workArrangement() string {
	switch i.RemoteStatus {
	case "fully":
		return "remote"
	case "hybrid":
		return "hybrid"
	case "none", "onsite":
		return "onsite"
	default:
		return ""
	}
}

func decode(body []byte) (feed, error) {
	var f feed
	if err := xml.Unmarshal(body, &f); err != nil {
		return feed{}, fmt.Errorf("parse xml: %w", err)
	}
	return f, nil
}

// BoardName returns the channel title of a jobs.rss body, or "" when it is unreadable.
func BoardName(body []byte) string {
	f, err := decode(body)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(f.Title)
}

func count(body []byte) (int, error) {
	f, err := decode(body)
	return len(f.Items), err
}

func parse(body []byte) ([]dto.Job, error) {
	f, err := decode(body)
	if err != nil {
		return nil, err
	}
	jobs := make([]dto.Job, 0, len(f.Items))
	for _, it := range f.Items {
		jobs = append(jobs, dto.Job{
			Title:             it.Title,
			Location:          it.location(),
			URL:               it.Link,
			ProviderPostingID: it.GUID,
			Description:       it.Description,
			WorkArrangement:   it.workArrangement(),
			UpdatedAt:         sources.TimeOrNow(time.RFC1123Z, it.PubDate),
		})
	}
	return jobs, nil
}

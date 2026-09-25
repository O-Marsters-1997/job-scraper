package crawl

import (
	"time"

	"github.com/temoto/robotstxt"
)

const (
	// userAgentToken is the crawler's identity for robots.txt group matching.
	// It's also the product token in the outbound User-Agent header, so a
	// robots.txt "User-agent: job-scraper-crawler" group matches either way.
	userAgentToken = "job-scraper-crawler"

	defaultCrawlDelay = 2 * time.Second
	maxCrawlDelay     = 10 * time.Second
)

// politeness wraps a parsed robots.txt for one host. A nil *politeness or a
// nil data field both mean allow-all — the correct behaviour when robots.txt
// is missing or unreachable, per spec.
type politeness struct {
	data *robotstxt.RobotsData
}

func parseRobots(body []byte) (*politeness, error) {
	data, err := robotstxt.FromBytes(body)
	if err != nil {
		return nil, err
	}
	return &politeness{data: data}, nil
}

// allowAllPoliteness is used when robots.txt could not be fetched or parsed:
// unreachable/missing robots.txt means "allow all", not "block everything".
func allowAllPoliteness() *politeness {
	data, _ := robotstxt.FromBytes(nil)
	return &politeness{data: data}
}

func (p *politeness) allowed(path string) bool {
	if p == nil || p.data == nil {
		return true
	}
	return p.data.TestAgent(path, userAgentToken)
}

// crawlDelay returns the inter-fetch delay to honour on this host: robots.txt's
// Crawl-delay for our user-agent group, capped at maxCrawlDelay, or
// defaultCrawlDelay when robots.txt specifies none.
func (p *politeness) crawlDelay() time.Duration {
	if p == nil || p.data == nil {
		return defaultCrawlDelay
	}
	delay := p.data.FindGroup(userAgentToken).CrawlDelay
	if delay <= 0 {
		return defaultCrawlDelay
	}
	if delay > maxCrawlDelay {
		return maxCrawlDelay
	}
	return delay
}

package crawl

import (
	"testing"
	"time"
)

func TestPolitenessAllowed(t *testing.T) {
	body := []byte("User-agent: *\nDisallow: /private/\n")
	p, err := parseRobots(body)
	if err != nil {
		t.Fatalf("parseRobots: %v", err)
	}
	if p.allowed("/private/secret") {
		t.Error("expected /private/secret to be disallowed")
	}
	if !p.allowed("/careers") {
		t.Error("expected /careers to be allowed")
	}
}

func TestPolitenessCrawlDelay(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want time.Duration
	}{
		{"no crawl-delay specified defaults", []byte("User-agent: *\nDisallow:\n"), defaultCrawlDelay},
		{"crawl-delay honoured", []byte("User-agent: *\nCrawl-delay: 5\n"), 5 * time.Second},
		{"crawl-delay capped at max", []byte("User-agent: *\nCrawl-delay: 30\n"), maxCrawlDelay},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parseRobots(tt.body)
			if err != nil {
				t.Fatalf("parseRobots: %v", err)
			}
			if got := p.crawlDelay(); got != tt.want {
				t.Errorf("crawlDelay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAllowAllPolitenessOnUnreachableRobots(t *testing.T) {
	// A missing or unreachable robots.txt means allow-all, per spec — never
	// treat a fetch failure as "block everything".
	p := allowAllPoliteness()
	if !p.allowed("/anything") {
		t.Error("expected allow-all when robots.txt is unreachable")
	}
	if p.crawlDelay() != defaultCrawlDelay {
		t.Errorf("crawlDelay() = %v, want default %v", p.crawlDelay(), defaultCrawlDelay)
	}
}

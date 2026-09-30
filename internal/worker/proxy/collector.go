package proxy

import (
	"context"
	"sync"
)

type collectorKey struct{}

// Collector records the cached URLs a task touched so the task can forget them on success.
type Collector struct {
	mu   sync.Mutex
	urls []string
}

func WithCollector(ctx context.Context) (context.Context, *Collector) {
	c := &Collector{}
	return context.WithValue(ctx, collectorKey{}, c), c
}

func (c *Collector) URLs() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.urls...)
}

func (c *Collector) add(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.urls = append(c.urls, url)
}

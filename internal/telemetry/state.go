package telemetry

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// StateReader reads the pipeline's current state for the state collector to scrape.
type StateReader interface {
	OpsState(ctx context.Context) (dto.OpsState, error)
}

const stateReadTimeout = 5 * time.Second

var (
	stateUpDesc = prometheus.NewDesc(
		"jobscraper_state_up", "1 if the last OpsState read succeeded, 0 otherwise.", nil, nil)
	outboxPendingDesc = prometheus.NewDesc(
		"jobscraper_outbox_pending", "Scoring effects with status pending or running.", nil, nil)
	outboxOldestPendingSecondsDesc = prometheus.NewDesc(
		"jobscraper_outbox_oldest_pending_seconds", "Age in seconds of the oldest pending or running scoring effect.", nil, nil)
	outboxFailedDesc = prometheus.NewDesc(
		"jobscraper_outbox_failed", "Scoring effects with status failed.", nil, nil)
)

type stateCollector struct {
	store StateReader
}

// NewStateCollector builds a prometheus.Collector that scrapes store.OpsState on each Collect,
// emitting jobscraper_state_up plus the outbox gauges. On error it emits only
// jobscraper_state_up 0, so missing data never reads as zeros.
func NewStateCollector(store StateReader) prometheus.Collector {
	return &stateCollector{store: store}
}

func (c *stateCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- stateUpDesc
	ch <- outboxPendingDesc
	ch <- outboxOldestPendingSecondsDesc
	ch <- outboxFailedDesc
}

func (c *stateCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), stateReadTimeout)
	defer cancel()

	state, err := c.store.OpsState(ctx)
	if err != nil {
		ch <- prometheus.MustNewConstMetric(stateUpDesc, prometheus.GaugeValue, 0)
		return
	}

	ch <- prometheus.MustNewConstMetric(stateUpDesc, prometheus.GaugeValue, 1)
	ch <- prometheus.MustNewConstMetric(outboxPendingDesc, prometheus.GaugeValue, float64(state.OutboxPending))
	ch <- prometheus.MustNewConstMetric(outboxOldestPendingSecondsDesc, prometheus.GaugeValue, state.OutboxOldestPendingAge.Seconds())
	ch <- prometheus.MustNewConstMetric(outboxFailedDesc, prometheus.GaugeValue, float64(state.OutboxFailed))
}

package sourcetargets

import "github.com/prometheus/client_golang/prometheus"

// TargetsDisabled counts Source Targets turned off because their Source's key
// was rejected. Register it on the registry of the process that disables them.
var TargetsDisabled = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "jobscraper_source_targets_disabled_total",
	Help: "Source Targets disabled by source and reason.",
}, []string{"source", "reason"})

func RegisterMetrics(reg prometheus.Registerer) {
	reg.MustRegister(TargetsDisabled)
}

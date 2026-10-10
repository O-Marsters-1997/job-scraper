package dto

import "time"

type UsageLevel string

const (
	UsageOK       UsageLevel = "ok"
	UsageWarn     UsageLevel = "warn"
	UsageCritical UsageLevel = "critical"
)

const (
	warnPercent     = 80
	criticalPercent = 95
)

const (
	QuotaOK            = "ok"
	QuotaNotConfigured = "not_configured"
	QuotaError         = "error"
)

// QuotaView is one provider's quota row. Level is UsageOK unless Status is
// QuotaOK and a limit exists.
type QuotaView struct {
	Provider  string     `json:"provider"`
	Status    string     `json:"status"`
	Used      *float64   `json:"used"`
	Limit     *float64   `json:"limit"`
	Unit      string     `json:"unit"`
	Percent   *float64   `json:"percent"`
	Level     UsageLevel `json:"level"`
	ResetsAt  *time.Time `json:"resetsAt"`
	FetchedAt *time.Time `json:"fetchedAt"`
	Error     string     `json:"error"`
}

// QuotaPercent returns used as a percentage of limit (0 to 100+) and the
// level it falls in. Missing or non-positive data yields no percent and
// UsageOK.
func QuotaPercent(used, limit *float64) (*float64, UsageLevel) {
	if used == nil || limit == nil || *limit <= 0 {
		return nil, UsageOK
	}
	percent := *used / *limit * 100
	switch {
	case percent >= criticalPercent:
		return &percent, UsageCritical
	case percent >= warnPercent:
		return &percent, UsageWarn
	default:
		return &percent, UsageOK
	}
}

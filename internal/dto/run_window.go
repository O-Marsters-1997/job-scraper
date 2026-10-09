package dto

import (
	"bytes"
	"encoding/json"
	"time"
)

type RunWindow struct {
	IntervalMinutes *int           `json:"interval_minutes"`
	Weekdays        []time.Weekday `json:"weekdays"`
	Start           string         `json:"start"`
	End             string         `json:"end"`
	Timezone        string         `json:"timezone"`
}

// RunWindowInput distinguishes an omitted run_window (Set false) from an
// explicit null (Set true, Window nil), which means manual only.
type RunWindowInput struct {
	Set    bool
	Window *RunWindow
}

func (i *RunWindowInput) UnmarshalJSON(b []byte) error {
	i.Set = true
	if bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
		i.Window = nil
		return nil
	}
	return json.Unmarshal(b, &i.Window)
}

// Package runwindow computes when a discovery Source Target runs next
// (ADR 0022).
package runwindow

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"time"
	_ "time/tzdata"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

const (
	MinIntervalMinutes = 60
	MaxIntervalMinutes = 7 * 24 * 60
	clockLayout        = "15:04"
	maxJitterFraction  = 0.1
)

// Default is hourly, Monday to Friday, 08:00 to 18:00 in Europe/London.
func Default() dto.RunWindow {
	interval := 60
	return dto.RunWindow{
		IntervalMinutes: &interval,
		Weekdays:        []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
		Start:           "08:00",
		End:             "18:00",
		Timezone:        "Europe/London",
	}
}

// Jitter spreads an interval by up to plus or minus 10%.
func Jitter(d time.Duration) time.Duration {
	spread := (rand.Float64()*2 - 1) * maxJitterFraction
	return d + time.Duration(float64(d)*spread)
}

// Validate rejects an unknown timezone, an empty or out-of-range weekday set,
// an inverted window and an interval under an hour.
func Validate(w dto.RunWindow) error {
	if _, err := time.LoadLocation(w.Timezone); err != nil || w.Timezone == "" || w.Timezone == "Local" {
		return fmt.Errorf("unknown timezone %q", w.Timezone)
	}
	if len(w.Weekdays) == 0 {
		return errors.New("weekdays must not be empty")
	}
	for _, d := range w.Weekdays {
		if d < time.Sunday || d > time.Saturday {
			return fmt.Errorf("invalid weekday %d", d)
		}
	}
	start, err := time.Parse(clockLayout, w.Start)
	if err != nil {
		return fmt.Errorf("start must be HH:MM, got %q", w.Start)
	}
	end, err := time.Parse(clockLayout, w.End)
	if err != nil {
		return fmt.Errorf("end must be HH:MM, got %q", w.End)
	}
	if !start.Before(end) {
		return errors.New("window start must be before end")
	}
	if w.IntervalMinutes != nil && (*w.IntervalMinutes < MinIntervalMinutes || *w.IntervalMinutes > MaxIntervalMinutes) {
		return fmt.Errorf("interval_minutes must be between %d and %d", MinIntervalMinutes, MaxIntervalMinutes)
	}
	return nil
}

// Next returns the next run after from: from plus the jittered interval, or
// the next window opening when that falls outside the window. It reports
// false for a manual window (nil interval) or one that fails Validate.
func Next(w dto.RunWindow, from time.Time, jitter func(time.Duration) time.Duration) (time.Time, bool) {
	if w.IntervalMinutes == nil || Validate(w) != nil {
		return time.Time{}, false
	}
	loc, _ := time.LoadLocation(w.Timezone)
	candidate := from.Add(jitter(time.Duration(*w.IntervalMinutes) * time.Minute)).In(loc)
	start, _ := time.Parse(clockLayout, w.Start)
	end, _ := time.Parse(clockLayout, w.End)

	day := time.Date(candidate.Year(), candidate.Month(), candidate.Day(), 0, 0, 0, 0, loc)
	for range 8 {
		if slices.Contains(w.Weekdays, day.Weekday()) {
			opens := clockOn(day, start, loc)
			closes := clockOn(day, end, loc)
			switch {
			case candidate.Before(opens):
				return opens, true
			case candidate.Before(closes):
				return candidate, true
			}
		}
		day = day.AddDate(0, 0, 1)
	}
	return time.Time{}, false
}

func clockOn(day, clock time.Time, loc *time.Location) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), clock.Hour(), clock.Minute(), 0, 0, loc)
}

// WeekdayMask packs weekdays into the stored bitmask: bit 0 is Monday, bit 6
// is Sunday.
func WeekdayMask(days []time.Weekday) int16 {
	var mask int16
	for _, d := range days {
		mask |= 1 << ((int(d) + 6) % 7)
	}
	return mask
}

// Weekdays unpacks a WeekdayMask, Monday first.
func Weekdays(mask int16) []time.Weekday {
	var days []time.Weekday
	for bit := range 7 {
		if mask&(1<<bit) != 0 {
			days = append(days, time.Weekday((bit+1)%7))
		}
	}
	return days
}

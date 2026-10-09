package runwindow_test

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/runwindow"
)

func exact(d time.Duration) time.Duration { return d }

func utc(year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, time.UTC)
}

func windowOf(interval int, days ...time.Weekday) dto.RunWindow {
	w := runwindow.Default()
	w.IntervalMinutes = &interval
	if len(days) > 0 {
		w.Weekdays = days
	}
	return w
}

func TestNext(t *testing.T) {
	everyDay := []time.Weekday{time.Sunday, time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday}
	tests := []struct {
		name string
		w    dto.RunWindow
		from time.Time
		want time.Time
	}{
		{"advances within the window", windowOf(60), utc(2026, time.October, 7, 9, 0), utc(2026, time.October, 7, 10, 0)},
		{"rolls over to the next morning", windowOf(60), utc(2026, time.October, 7, 16, 30), utc(2026, time.October, 8, 7, 0)},
		{"before opening waits for opening", windowOf(60), utc(2026, time.October, 7, 5, 0), utc(2026, time.October, 7, 7, 0)},
		{"skips the weekend", windowOf(60), utc(2026, time.October, 9, 16, 30), utc(2026, time.October, 12, 7, 0)},
		{"spring forward moves the opening an hour earlier in UTC", windowOf(60), utc(2026, time.March, 27, 17, 30), utc(2026, time.March, 30, 7, 0)},
		{"autumn back keeps the opening at 08:00 UTC", windowOf(60), utc(2026, time.October, 23, 16, 30), utc(2026, time.October, 26, 8, 0)},
		{"spring forward day stays in window", windowOf(60, everyDay...), utc(2026, time.March, 29, 6, 30), utc(2026, time.March, 29, 7, 30)},
		{"autumn back day opens at 08:00 local", windowOf(60, everyDay...), utc(2026, time.October, 25, 6, 30), utc(2026, time.October, 25, 8, 0)},
		{"window end is exclusive", windowOf(60), utc(2026, time.October, 7, 16, 0), utc(2026, time.October, 8, 7, 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := runwindow.Next(tt.w, tt.from, exact)
			if !ok || !got.Equal(tt.want) {
				t.Errorf("Next(%s) = %v, %t, want %v, true", tt.from, got, ok, tt.want)
			}
		})
	}

	t.Run("manual window never runs", func(t *testing.T) {
		w := runwindow.Default()
		w.IntervalMinutes = nil
		if got, ok := runwindow.Next(w, utc(2026, time.October, 7, 9, 0), exact); ok {
			t.Errorf("Next(manual) = %v, true, want false", got)
		}
	})

	t.Run("jitter stays within ten percent", func(t *testing.T) {
		from := utc(2026, time.October, 7, 9, 0)
		for range 200 {
			got, ok := runwindow.Next(windowOf(120), from, runwindow.Jitter)
			if !ok {
				t.Fatal("Next() ok = false, want true")
			}
			if d := got.Sub(from); d < 108*time.Minute || d > 132*time.Minute {
				t.Fatalf("Next() advanced %v, want within 108m..132m", d)
			}
		}
	})

	t.Run("jitter stub shifts the result", func(t *testing.T) {
		later := func(d time.Duration) time.Duration { return d + 6*time.Minute }
		got, _ := runwindow.Next(windowOf(60), utc(2026, time.October, 7, 9, 0), later)
		if want := utc(2026, time.October, 7, 10, 6); !got.Equal(want) {
			t.Errorf("Next() = %v, want %v", got, want)
		}
	})
}

func TestValidate(t *testing.T) {
	with := func(edit func(*dto.RunWindow)) dto.RunWindow {
		w := runwindow.Default()
		edit(&w)
		return w
	}
	short := 30
	tests := []struct {
		name    string
		w       dto.RunWindow
		wantErr bool
	}{
		{"default is valid", runwindow.Default(), false},
		{"manual is valid", with(func(w *dto.RunWindow) { w.IntervalMinutes = nil }), false},
		{"unknown timezone", with(func(w *dto.RunWindow) { w.Timezone = "Mars/Olympus" }), true},
		{"empty timezone", with(func(w *dto.RunWindow) { w.Timezone = "" }), true},
		{"empty weekdays", with(func(w *dto.RunWindow) { w.Weekdays = nil }), true},
		{"weekday out of range", with(func(w *dto.RunWindow) { w.Weekdays = []time.Weekday{7} }), true},
		{"inverted window", with(func(w *dto.RunWindow) { w.Start, w.End = "18:00", "08:00" }), true},
		{"empty window", with(func(w *dto.RunWindow) { w.End = w.Start }), true},
		{"malformed start", with(func(w *dto.RunWindow) { w.Start = "8am" }), true},
		{"interval under an hour", with(func(w *dto.RunWindow) { w.IntervalMinutes = &short }), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := runwindow.Validate(tt.w); (err != nil) != tt.wantErr {
				t.Errorf("Validate() err = %v, wantErr %t", err, tt.wantErr)
			}
		})
	}
}

func TestWeekdayMask(t *testing.T) {
	weekdays := []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday}
	if got := runwindow.WeekdayMask(weekdays); got != 31 {
		t.Errorf("WeekdayMask(Mon-Fri) = %d, want 31", got)
	}
	if got := runwindow.WeekdayMask([]time.Weekday{time.Sunday}); got != 64 {
		t.Errorf("WeekdayMask(Sun) = %d, want 64", got)
	}
	if diff := cmp.Diff(weekdays, runwindow.Weekdays(31)); diff != "" {
		t.Errorf("Weekdays(31) mismatch (-want +got):\n%s", diff)
	}
}

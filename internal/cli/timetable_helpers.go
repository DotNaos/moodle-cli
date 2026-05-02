package cli

import (
	"sort"
	"strings"
	"time"

	"github.com/DotNaos/moodle-services/internal/moodle"
)

func uniqueSummaries(events []moodle.CalendarEvent) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(events))
	for _, ev := range events {
		key := strings.TrimSpace(ev.Summary)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func filterNextWeekWithEvents(events []moodle.CalendarEvent, now time.Time) []moodle.CalendarEvent {
	if len(events) == 0 {
		return events
	}

	// Start from current week (Monday)
	year, week := now.ISOWeek()
	for i := 0; i < 10; i++ { // scan up to 10 weeks ahead
		start := isoWeekStart(year, week)
		end := start.AddDate(0, 0, 7)

		weekEvents := make([]moodle.CalendarEvent, 0)
		for _, ev := range events {
			if !ev.Start.IsZero() && (ev.Start.Equal(start) || (ev.Start.After(start) && ev.Start.Before(end))) {
				weekEvents = append(weekEvents, ev)
			}
		}

		if len(weekEvents) > 0 {
			sort.Slice(weekEvents, func(i, j int) bool {
				return weekEvents[i].Start.Before(weekEvents[j].Start)
			})
			return weekEvents
		}

		// advance week
		week++
		if week > 53 {
			week = 1
			year++
		}
	}

	return []moodle.CalendarEvent{}
}

func isoWeekStart(year int, week int) time.Time {
	// ISO week start: Monday
	// Start from Jan 4th which is always in week 1
	t := time.Date(year, 1, 4, 0, 0, 0, 0, time.Local)
	// find Monday of week 1
	for t.Weekday() != time.Monday {
		t = t.AddDate(0, 0, -1)
	}
	// advance to desired week
	return t.AddDate(0, 0, (week-1)*7)
}

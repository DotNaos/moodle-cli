package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/DotNaos/moodle-services/internal/config"
	"github.com/DotNaos/moodle-services/internal/moodle"
	"github.com/spf13/cobra"
)

var timetableDays int
var timetableUnique bool
var timetableNextWeek bool

var timetableCmd = &cobra.Command{
	Use:     "timetable",
	Short:   "List timetable events",
	Long:    "List upcoming timetable events from your calendar.\n\nRequires a calendar URL set in config (config set --calendar-url).",
	Example: "  moodle list timetable\n  moodle list timetable --days 30\n  moodle --json list timetable",
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig(opts.ConfigPath)
		if err != nil {
			return err
		}
		if cfg.CalendarURL == "" {
			return fmt.Errorf("calendar URL not set. Run: moodle config set --calendar-url <url>")
		}

		now := time.Now()
		from := now.Add(-24 * time.Hour)
		to := now.Add(time.Duration(timetableDays) * 24 * time.Hour)

		events, err := moodle.FetchCalendarEvents(cfg.CalendarURL, from, to)
		if err != nil {
			return err
		}

		if timetableNextWeek {
			events = filterNextWeekWithEvents(events, now)
		}

		if timetableUnique {
			summaries := uniqueSummaries(events)
			return writeCommandOutput(cmd, summaries, func(w io.Writer) error {
				for _, entry := range summaries {
					if _, err := fmt.Fprintln(w, entry); err != nil {
						return err
					}
				}
				return nil
			})
		}

		return writeCommandOutput(cmd, events, func(w io.Writer) error {
			for _, d := range events {
				if _, err := fmt.Fprintf(w, "%s\t%s\t%s\n", d.Start.Format(time.RFC3339), d.Summary, d.Location); err != nil {
					return err
				}
			}
			return nil
		})
	},
}

func init() {
	timetableCmd.Flags().IntVar(&timetableDays, "days", 90, "Number of days to look ahead")
	timetableCmd.Flags().BoolVar(&timetableUnique, "unique", false, "Show unique event summaries only")
	timetableCmd.Flags().BoolVar(&timetableNextWeek, "next-week", false, "Only show events from the next week with entries")
}

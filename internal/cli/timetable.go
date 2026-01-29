package cli

import (
  "encoding/json"
  "fmt"
  "time"

  "github.com/DotNaos/moodle-cli/internal/config"
  "github.com/DotNaos/moodle-cli/internal/moodle"
  "github.com/spf13/cobra"
)

var timetableJSON bool
var timetableDays int
var timetableUnique bool
var timetableNextWeek bool

var timetableCmd = &cobra.Command{
  Use:   "timetable",
  Short: "List timetable events",
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
      if timetableJSON {
        data, err := json.MarshalIndent(uniqueSummaries(events), "", "  ")
        if err != nil {
          return err
        }
        fmt.Println(string(data))
        return nil
      }
      for _, entry := range uniqueSummaries(events) {
        fmt.Println(entry)
      }
      return nil
    }

    if timetableJSON {
      data, err := json.MarshalIndent(events, "", "  ")
      if err != nil {
        return err
      }
      fmt.Println(string(data))
      return nil
    }

    for _, d := range events {
      fmt.Printf("%s\t%s\t%s\n", d.Start.Format(time.RFC3339), d.Summary, d.Location)
    }
    return nil
  },
}

func init() {
  timetableCmd.Flags().BoolVar(&timetableJSON, "json", false, "Output JSON")
  timetableCmd.Flags().IntVar(&timetableDays, "days", 90, "Number of days to look ahead")
  timetableCmd.Flags().BoolVar(&timetableUnique, "unique", false, "Show unique event summaries only")
  timetableCmd.Flags().BoolVar(&timetableNextWeek, "next-week", false, "Only show events from the next week with entries")
}

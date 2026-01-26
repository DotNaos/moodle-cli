package cli

import (
  "encoding/json"
  "fmt"

  "github.com/DotNaos/moodle-cli/internal/config"
  "github.com/spf13/cobra"
)

var (
  cfgSchoolID    string
  cfgUsername    string
  cfgPassword    string
  cfgCalendarURL string
  cfgJSON        bool
)

var configCmd = &cobra.Command{
  Use:   "config",
  Short: "Configure moodle-cli",
  ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    return nil, cobra.ShellCompDirectiveNoFileComp
  },
}

var configShowCmd = &cobra.Command{
  Use:   "show",
  Short: "Show current configuration",
  RunE: func(cmd *cobra.Command, args []string) error {
    cfg, err := config.LoadConfig(opts.ConfigPath)
    if err != nil {
      return err
    }
    if cfgJSON {
      data, err := json.MarshalIndent(cfg, "", "  ")
      if err != nil {
        return err
      }
      fmt.Println(string(data))
      return nil
    }

    if cfg.SchoolID != "" {
      fmt.Printf("schoolId: %s\n", cfg.SchoolID)
    }
    if cfg.Username != "" {
      fmt.Printf("username: %s\n", cfg.Username)
    }
    if cfg.CalendarURL != "" {
      fmt.Printf("calendarUrl: %s\n", cfg.CalendarURL)
    }
    if cfg.Password != "" {
      fmt.Println("password: (set)")
    }
    return nil
  },
}

var configSetCmd = &cobra.Command{
  Use:   "set",
  Short: "Set configuration values",
  RunE: func(cmd *cobra.Command, args []string) error {
    cfg, err := config.LoadConfig(opts.ConfigPath)
    if err != nil {
      return err
    }
    if cfgSchoolID != "" {
      cfg.SchoolID = cfgSchoolID
    }
    if cfgUsername != "" {
      cfg.Username = cfgUsername
    }
    if cfgPassword != "" {
      cfg.Password = cfgPassword
    }
    if cfgCalendarURL != "" {
      cfg.CalendarURL = cfgCalendarURL
    }

    if err := config.SaveConfig(opts.ConfigPath, cfg); err != nil {
      return err
    }
    fmt.Printf("config saved to %s\n", opts.ConfigPath)
    return nil
  },
}

func init() {
  configShowCmd.Flags().BoolVar(&cfgJSON, "json", false, "Output JSON")

  configSetCmd.Flags().StringVar(&cfgSchoolID, "school", "", "School id (e.g. fhgr, phgr)")
  configSetCmd.Flags().StringVar(&cfgUsername, "username", "", "Moodle username/email")
  configSetCmd.Flags().StringVar(&cfgPassword, "password", "", "Moodle password")
  configSetCmd.Flags().StringVar(&cfgCalendarURL, "calendar-url", "", "ICS calendar URL")

  configCmd.AddCommand(configShowCmd, configSetCmd)
}

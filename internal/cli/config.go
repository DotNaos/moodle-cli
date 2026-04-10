package cli

import (
	"fmt"
	"io"

	"github.com/DotNaos/moodle-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	cfgSchoolID    string
	cfgUsername    string
	cfgPassword    string
	cfgCalendarURL string
)

type configSetResult struct {
	ConfigPath string        `json:"configPath" yaml:"configPath"`
	Config     config.Config `json:"config" yaml:"config"`
}

var configCmd = &cobra.Command{
	Use:     "config",
	Short:   "Manage configuration (credentials, calendar, optional school override)",
	Long:    "Show or set configuration values used by moodle-cli.\n\nUse 'config show' to inspect current values or 'config set' to update them.",
	Example: "  moodle config show\n  moodle config set --username you@example.com --password \"secret\"\n  moodle config set --calendar-url \"https://.../calendar.ics\"",
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return helpOrMachineError(cmd, "expected a config subcommand")
	},
}

var configShowCmd = &cobra.Command{
	Use:     "show",
	Short:   "Show current configuration",
	Long:    "Show the current configuration values.\nPasswords are masked in text output.",
	Example: "  moodle config show\n  moodle --json config show\n  moodle --yaml config show",
	ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig(opts.ConfigPath)
		if err != nil {
			return err
		}
		return writeCommandOutput(cmd, cfg, func(w io.Writer) error {
			if cfg.SchoolID != "" {
				if _, err := fmt.Fprintf(w, "schoolId: %s\n", cfg.SchoolID); err != nil {
					return err
				}
			}
			if cfg.Username != "" {
				if _, err := fmt.Fprintf(w, "username: %s\n", cfg.Username); err != nil {
					return err
				}
			}
			if cfg.CalendarURL != "" {
				if _, err := fmt.Fprintf(w, "calendarUrl: %s\n", cfg.CalendarURL); err != nil {
					return err
				}
			}
			if cfg.Password != "" {
				if _, err := fmt.Fprintln(w, "password: (set)"); err != nil {
					return err
				}
			}
			return nil
		})
	},
}

var configSetCmd = &cobra.Command{
	Use:     "set",
	Short:   "Set configuration values",
	Long:    "Update configuration values used for login and timetable.\nOnly provided flags are updated; other values remain unchanged.",
	Example: "  moodle config set --username you@example.com --password \"secret\"\n  moodle config set --calendar-url \"https://.../calendar.ics\"",
	ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
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
		result := configSetResult{
			ConfigPath: opts.ConfigPath,
			Config:     cfg,
		}
		return writeCommandOutput(cmd, result, func(w io.Writer) error {
			_, err := fmt.Fprintf(w, "config saved to %s\n", opts.ConfigPath)
			return err
		})
	},
}

func init() {
	configSetCmd.Flags().StringVar(&cfgSchoolID, "school", "", "School id override. Only fhgr is currently active; multi-school support is not active")
	configSetCmd.Flags().StringVar(&cfgUsername, "username", "", "Moodle username/email")
	configSetCmd.Flags().StringVar(&cfgPassword, "password", "", "Moodle password")
	configSetCmd.Flags().StringVar(&cfgCalendarURL, "calendar-url", "", "ICS calendar URL")

	configSetCmd.RegisterFlagCompletionFunc("school", completeSchoolIDs)

	configCmd.AddCommand(configShowCmd, configSetCmd)
}

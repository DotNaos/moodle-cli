package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/DotNaos/moodle-services/internal/config"
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

type configMigrateHomeResult struct {
	Status      string `json:"status" yaml:"status"`
	Source      string `json:"source" yaml:"source"`
	Target      string `json:"target" yaml:"target"`
	CopiedFiles int    `json:"copiedFiles" yaml:"copiedFiles"`
}

var configCmd = &cobra.Command{
	Use:     "config",
	Short:   "Manage configuration (credentials, calendar, optional school override)",
	Long:    "Show or set configuration values used by Moodle Services.\n\nUse 'config show' to inspect current values or 'config set' to update them.",
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

var configMigrateHomeCmd = &cobra.Command{
	Use:   "migrate-home",
	Short: "Copy legacy ~/.moodle-cli data into ~/.moodle",
	Long:  "Copy legacy data from ~/.moodle-cli into the shared ~/.moodle folder without deleting the old data.",
	Args:  cobra.NoArgs,
	ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := migrateLegacyHome()
		if err != nil {
			return err
		}
		return writeCommandOutput(cmd, result, func(w io.Writer) error {
			return renderConfigMigrateHomeText(w, result)
		})
	},
}

func init() {
	configSetCmd.Flags().StringVar(&cfgSchoolID, "school", "", "School id override. Only fhgr is currently active; multi-school support is not active")
	configSetCmd.Flags().StringVar(&cfgUsername, "username", "", "Moodle username/email")
	configSetCmd.Flags().StringVar(&cfgPassword, "password", "", "Moodle password")
	configSetCmd.Flags().StringVar(&cfgCalendarURL, "calendar-url", "", "ICS calendar URL")

	configSetCmd.RegisterFlagCompletionFunc("school", completeSchoolIDs)

	configCmd.AddCommand(configShowCmd, configSetCmd, configMigrateHomeCmd)
}

func migrateLegacyHome() (configMigrateHomeResult, error) {
	source := config.LegacyBaseDir()
	target := config.BaseDir()
	result := configMigrateHomeResult{
		Source: source,
		Target: target,
	}
	if filepath.Clean(source) == filepath.Clean(target) {
		result.Status = "same-path"
		return result, nil
	}
	sourceInfo, err := os.Stat(source)
	if err != nil {
		if os.IsNotExist(err) {
			result.Status = "nothing-to-migrate"
			return result, nil
		}
		return result, err
	}
	if !sourceInfo.IsDir() {
		return result, fmt.Errorf("legacy path exists but is not a directory: %s", source)
	}

	targetEntries, err := os.ReadDir(target)
	if err == nil && len(targetEntries) > 0 {
		return result, fmt.Errorf("target %s already contains data; leaving legacy data untouched", target)
	}
	if err != nil && !os.IsNotExist(err) {
		return result, err
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return result, err
	}

	copied, err := copyDirectoryContents(source, target)
	if err != nil {
		return result, err
	}
	result.Status = "migrated"
	result.CopiedFiles = copied
	return result, nil
}

func copyDirectoryContents(source string, target string) (int, error) {
	copied := 0
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		dst := filepath.Join(target, rel)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.MkdirAll(dst, info.Mode().Perm())
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if err := copyFile(path, dst, info.Mode().Perm()); err != nil {
			return err
		}
		copied++
		return nil
	})
	return copied, err
}

func copyFile(source string, target string, mode os.FileMode) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if mode == 0 {
		mode = 0o600
	}
	return os.WriteFile(target, data, mode)
}

func renderConfigMigrateHomeText(w io.Writer, result configMigrateHomeResult) error {
	if _, err := fmt.Fprintf(w, "status: %s\n", result.Status); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "source: %s\n", result.Source); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "target: %s\n", result.Target); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "copied files: %d\n", result.CopiedFiles)
	return err
}

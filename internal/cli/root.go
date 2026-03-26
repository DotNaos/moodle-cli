package cli

import (
	"fmt"
	"strings"

	"github.com/DotNaos/moodle-cli/internal/config"
	"github.com/spf13/cobra"
)

type Options struct {
	ConfigPath   string
	SessionPath  string
	CacheDBPath  string
	FileCacheDir string
	StatePath    string
	ExportDir    string
	Unsanitized  bool
}

var opts Options

var rootCmd = &cobra.Command{
	Use:   "moodle",
	Short: "CLI for FHGR Moodle",
	Long:  "Command-line access to Moodle for listing courses and files, downloading resources, exporting courses, and viewing your timetable.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return maybeCheckForUpdates(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return launchTUI(selectorOptions{})
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&opts.ConfigPath, "config", config.ConfigPath(), "Config file path")
	rootCmd.PersistentFlags().StringVar(&opts.SessionPath, "session", config.SessionPath(), "Session cookie file path")
	rootCmd.PersistentFlags().StringVar(&opts.CacheDBPath, "cache", config.CacheDBPath(), "SQLite cache path")
	rootCmd.PersistentFlags().StringVar(&opts.FileCacheDir, "files-cache", config.FileCacheDir(), "File cache directory")
	rootCmd.PersistentFlags().StringVar(&opts.StatePath, "state", config.StatePath(), "State file path")
	rootCmd.PersistentFlags().StringVar(&opts.ExportDir, "output-dir", config.ExportDir(), "Output directory")
	rootCmd.PersistentFlags().BoolVar(&opts.Unsanitized, "unsanitized", false, "Preserve raw scraped names instead of sanitized defaults")

	rootCmd.SetHelpTemplate(fmt.Sprintf("%s\n\nDefault paths:\n  config: %s\n  session: %s\n  cache: %s\n  files: %s\n  state: %s\n  output: %s\n", rootCmd.HelpTemplate(), config.ConfigPath(), config.SessionPath(), config.CacheDBPath(), config.FileCacheDir(), config.StatePath(), config.ExportDir()))
	rootCmd.SilenceUsage = true

	rootCmd.AddCommand(
		configCmd,
		loginCmd,
		listCmd,
		navCmd,
		openCmd,
		downloadCmd,
		exportCmd,
		printCmd,
		tuiCmd,
		versionCmd,
		updateCmd,
	)
}

func Execute() error {
	return rootCmd.Execute()
}

func commandPathHas(cmd *cobra.Command, name string) bool {
	for current := cmd; current != nil; current = current.Parent() {
		if strings.EqualFold(current.Name(), name) {
			return true
		}
	}
	return false
}

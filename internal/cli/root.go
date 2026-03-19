package cli

import (
	"fmt"

	"github.com/DotNaos/moodle-cli/internal/config"
	"github.com/spf13/cobra"
)

type Options struct {
	ConfigPath   string
	SessionPath  string
	CacheDBPath  string
	FileCacheDir string
	ExportDir    string
}

var opts Options

var rootCmd = &cobra.Command{
	Use:   "moodle",
	Short: "CLI for FHGR Moodle",
	Long:  "Command-line access to Moodle for listing courses and files, downloading resources, exporting courses, and viewing your timetable.",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&opts.ConfigPath, "config", config.ConfigPath(), "Config file path")
	rootCmd.PersistentFlags().StringVar(&opts.SessionPath, "session", config.SessionPath(), "Session cookie file path")
	rootCmd.PersistentFlags().StringVar(&opts.CacheDBPath, "cache", config.CacheDBPath(), "SQLite cache path")
	rootCmd.PersistentFlags().StringVar(&opts.FileCacheDir, "files-cache", config.FileCacheDir(), "File cache directory")
	rootCmd.PersistentFlags().StringVar(&opts.ExportDir, "output-dir", config.ExportDir(), "Output directory")

	rootCmd.SetHelpTemplate(fmt.Sprintf("%s\n\nDefault paths:\n  config: %s\n  session: %s\n  cache: %s\n  files: %s\n  output: %s\n", rootCmd.HelpTemplate(), config.ConfigPath(), config.SessionPath(), config.CacheDBPath(), config.FileCacheDir(), config.ExportDir()))

	rootCmd.AddCommand(
		configCmd,
		loginCmd,
		listCmd,
		openCmd,
		downloadCmd,
		exportCmd,
		printCmd,
	)
}

func Execute() error {
	return rootCmd.Execute()
}

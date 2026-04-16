package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var (
	logsLines  int
	logsFollow bool
	logsErrors bool
)

var logTailPollInterval = 500 * time.Millisecond

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Tail moodle-cli debug or error logs",
	Long:  "Stream the CLI debug or error logs so agents and humans can inspect complete command events instead of raw line fragments.",
	Args:  cobra.NoArgs,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logPath := debugLogPath()
		label := "debug"
		if logsErrors {
			logPath = errorLogPath()
			label = "error"
		}

		if err := ensureLogFilePresent(logPath); err != nil {
			return err
		}

		if isMachineOutput() {
			return streamLogs(cmd, logPath, label, logsLines, logsFollow)
		}
		return renderLogs(cmd, logPath, label, logsLines, logsFollow)
	},
}

func init() {
	logsCmd.Flags().BoolVar(&logsErrors, "error", false, "Show the error log instead of the debug log")
	logsCmd.Flags().BoolVar(&logsFollow, "follow", true, "Follow updates to the log (like tail -f)")
	logsCmd.Flags().IntVar(&logsLines, "lines", 200, "Number of recent log entries to show before following")
}

func ensureLogFilePresent(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return os.WriteFile(path, []byte{}, 0o644)
	}
	return nil
}

type logsEvent struct {
	Type   string     `json:"type" yaml:"type"`
	Label  string     `json:"label,omitempty" yaml:"label,omitempty"`
	Path   string     `json:"path,omitempty" yaml:"path,omitempty"`
	Follow bool       `json:"follow,omitempty" yaml:"follow,omitempty"`
	Record *logRecord `json:"record,omitempty" yaml:"record,omitempty"`
}

func streamLogs(cmd *cobra.Command, logPath string, label string, lines int, follow bool) error {
	if err := writeStreamEvent(cmd.OutOrStdout(), logsEvent{
		Type:   "meta",
		Label:  label,
		Path:   logPath,
		Follow: follow,
	}); err != nil {
		return err
	}

	err := streamLogRecords(cmd.Context(), logPath, label, lines, follow, func(record logRecord) error {
		return writeStreamEvent(cmd.OutOrStdout(), logsEvent{
			Type:   "record",
			Label:  label,
			Path:   logPath,
			Record: &record,
		})
	})
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func renderLogs(cmd *cobra.Command, logPath string, label string, limit int, follow bool) error {
	header := "Tailing"
	if !follow {
		header = "Showing"
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s %s log at %s (last %d entries)\n", header, label, logPath, limit); err != nil {
		return err
	}

	err := streamLogRecords(cmd.Context(), logPath, label, limit, follow, func(record logRecord) error {
		return renderLogRecord(cmd.OutOrStdout(), record)
	})
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

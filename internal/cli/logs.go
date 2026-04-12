package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	Long:  "Stream the CLI debug or error logs (similar to `tail -f`) so agents can observe command activity and failures.",
	Args:  cobra.NoArgs,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if isMachineOutput() {
			return machineCommandError("logs_text_only", "logs command emits plain text; omit --json/--yaml")
		}

		logPath := debugLogPath()
		label := "debug"
		if logsErrors {
			logPath = errorLogPath()
			label = "error"
		}

		if err := ensureLogFilePresent(logPath); err != nil {
			return err
		}

		header := "Tailing"
		if !logsFollow {
			header = "Showing"
		}
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s %s log at %s (last %d lines)\n", header, label, logPath, logsLines); err != nil {
			return err
		}

		err := tailLogFile(cmd.Context(), logPath, logsLines, logsFollow, cmd.OutOrStdout())
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	},
}

func init() {
	logsCmd.Flags().BoolVar(&logsErrors, "error", false, "Show the error log instead of the debug log")
	logsCmd.Flags().BoolVar(&logsFollow, "follow", true, "Follow updates to the log (like tail -f)")
	logsCmd.Flags().IntVar(&logsLines, "lines", 200, "Number of recent lines to show before following")
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

func tailLogFile(ctx context.Context, path string, lines int, follow bool, out io.Writer) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	offset, err := tailStartOffset(file, lines)
	if err != nil {
		return err
	}
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return err
	}

	written, err := io.Copy(out, file)
	if err != nil {
		return err
	}
	currentOffset := offset + written

	if !follow {
		return nil
	}

	ticker := time.NewTicker(logTailPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			stat, err := os.Stat(path)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					currentOffset = 0
					if err := ensureLogFilePresent(path); err != nil {
						return err
					}
					file.Close()
					file, err = os.Open(path)
					if err != nil {
						return err
					}
					continue
				}
				return err
			}

			if stat.Size() < currentOffset {
				currentOffset = 0
				if _, err := file.Seek(0, io.SeekStart); err != nil {
					return err
				}
			}

			if stat.Size() > currentOffset {
				if _, err := file.Seek(currentOffset, io.SeekStart); err != nil {
					return err
				}
				n, err := io.CopyN(out, file, stat.Size()-currentOffset)
				if err != nil && !errors.Is(err, io.EOF) {
					return err
				}
				currentOffset += n
			}
		}
	}
}

func tailStartOffset(file *os.File, lines int) (int64, error) {
	if lines <= 0 {
		return 0, nil
	}

	stat, err := file.Stat()
	if err != nil {
		return 0, err
	}

	size := stat.Size()
	if size == 0 {
		return 0, nil
	}

	const chunkSize int64 = 4096
	var (
		offset    = size
		lineCount = 0
		buf       = make([]byte, chunkSize)
	)

	for offset > 0 && lineCount <= lines {
		toRead := chunkSize
		if offset < toRead {
			toRead = offset
		}
		offset -= toRead

		if _, err := file.ReadAt(buf[:toRead], offset); err != nil {
			return 0, err
		}
		for i := toRead - 1; i >= 0; i-- {
			if buf[i] == '\n' {
				lineCount++
				if lineCount > lines {
					return offset + i + 1, nil
				}
			}
		}
	}

	return 0, nil
}

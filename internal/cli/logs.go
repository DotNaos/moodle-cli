package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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

type logsEvent struct {
	Type   string `json:"type" yaml:"type"`
	Label  string `json:"label,omitempty" yaml:"label,omitempty"`
	Path   string `json:"path,omitempty" yaml:"path,omitempty"`
	Follow bool   `json:"follow,omitempty" yaml:"follow,omitempty"`
	Line   string `json:"line,omitempty" yaml:"line,omitempty"`
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

	writer := &logEventWriter{
		emit: func(line string) error {
			return writeStreamEvent(cmd.OutOrStdout(), logsEvent{
				Type:  "line",
				Label: label,
				Path:  logPath,
				Line:  line,
			})
		},
	}

	err := tailLogFile(cmd.Context(), logPath, lines, follow, writer)
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	if !follow {
		if err := writer.Flush(); err != nil {
			return err
		}
	}
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

type logEventWriter struct {
	pending string
	emit    func(string) error
}

func (w *logEventWriter) Write(p []byte) (int, error) {
	w.pending += string(p)
	for {
		index := strings.IndexByte(w.pending, '\n')
		if index < 0 {
			break
		}
		line := w.pending[:index]
		if err := w.emit(line); err != nil {
			return 0, err
		}
		w.pending = w.pending[index+1:]
	}
	return len(p), nil
}

func (w *logEventWriter) Flush() error {
	if w.pending == "" {
		return nil
	}
	line := w.pending
	w.pending = ""
	return w.emit(line)
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

package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func debugLogPath() string {
	return filepath.Join(filepath.Dir(opts.StatePath), "cli.log")
}

func errorLogPath() string {
	return filepath.Join(filepath.Dir(opts.StatePath), "error.log")
}

func appendDebugLog(scope string, lines ...string) (string, error) {
	logPath := debugLogPath()
	return appendLogFile(logPath, "debug", scope, lines...)
}

func appendErrorLog(scope string, lines ...string) (string, error) {
	logPath := errorLogPath()
	return appendLogFile(logPath, "error", scope, lines...)
}

func appendLogFile(logPath string, level string, scope string, lines ...string) (string, error) {
	return appendLogRecord(logPath, newLogRecord(level, scope, lines))
}

func logDebug(scope string, lines ...string) string {
	logPath, err := appendDebugLog(scope, lines...)
	if err != nil {
		return ""
	}
	return logPath
}

func logUnexpected(scope string, err error, lines ...string) string {
	if err == nil {
		return ""
	}
	payload := append([]string{"error: " + strings.TrimSpace(err.Error())}, lines...)
	logPath, logErr := appendErrorLog(scope, payload...)
	if logErr != nil {
		return ""
	}
	return logPath
}

func presentUIError(scope string, err error, lines ...string) string {
	if err == nil {
		return ""
	}
	raw := strings.TrimSpace(err.Error())
	if raw == "" {
		raw = "unexpected error"
	}
	if isUnexpectedUIError(raw) {
		logPath := ""
		if existing := extractDetailsPath(raw); existing != "" {
			logPath = existing
		} else {
			logPath = logUnexpected(scope, err, lines...)
		}
		return unexpectedUIMessage(scope, logPath)
	}
	return stripDetailsSuffix(raw)
}

func isUnexpectedUIError(raw string) bool {
	lower := strings.ToLower(strings.TrimSpace(raw))
	markers := []string{
		"details:",
		"exit status",
		"error domain=",
		"panic",
		"runtime error",
		"signal:",
		"nil pointer",
	}
	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func stripDetailsSuffix(raw string) string {
	const marker = " (details: "
	if index := strings.Index(raw, marker); index >= 0 {
		return strings.TrimSpace(raw[:index])
	}
	return raw
}

func extractDetailsPath(raw string) string {
	const marker = "(details: "
	start := strings.Index(raw, marker)
	if start < 0 {
		return ""
	}
	start += len(marker)
	end := strings.Index(raw[start:], ")")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(raw[start : start+end])
}

func unexpectedUIMessage(scope string, logPath string) string {
	base := "Something went wrong."
	switch scope {
	case "tui.open":
		base = "Could not open the file."
	case "tui.print":
		base = "Could not load the preview."
	case "tui.download":
		base = "Could not save the file."
	case "tui.children":
		base = "Could not load this section."
	}
	if strings.TrimSpace(logPath) == "" {
		logPath = errorLogPath()
	}
	return fmt.Sprintf("%s See %s.", base, logPath)
}

func joinErrors(left error, right error) error {
	switch {
	case left == nil:
		return right
	case right == nil:
		return left
	default:
		return errors.Join(left, right)
	}
}

var (
	currentCommandPath  string
	currentCommandArgs  []string
	currentCommandStart time.Time
)

func recordCommandInvocation(cmd *cobra.Command) {
	currentCommandPath = cmd.CommandPath()
	currentCommandArgs = redactSensitiveArgs(os.Args[1:])
	currentCommandStart = time.Now()

	lines := []string{
		"event: start",
		"command: " + currentCommandPath,
	}
	if len(currentCommandArgs) > 0 {
		lines = append(lines, "args: "+strings.Join(currentCommandArgs, " "))
	}
	lines = append(lines, "output: "+string(currentOutputFormat()))
	logDebug("cli", lines...)
}

func logCommandResult(err error) {
	if currentCommandPath == "" || currentCommandStart.IsZero() {
		return
	}

	lines := []string{
		"event: finish",
		"command: " + currentCommandPath,
		"duration: " + time.Since(currentCommandStart).Round(time.Millisecond).String(),
	}
	if len(currentCommandArgs) > 0 {
		lines = append(lines, "args: "+strings.Join(currentCommandArgs, " "))
	}

	if err != nil {
		lines = append(lines, "status: error", "error: "+strings.TrimSpace(err.Error()))
		_, _ = appendDebugLog("cli", lines...)
		_, _ = appendErrorLog("cli", lines...)
		return
	}

	lines = append(lines, "status: ok")
	_, _ = appendDebugLog("cli", lines...)
}

func redactSensitiveArgs(args []string) []string {
	out := make([]string, 0, len(args))
	redactNext := false
	for _, arg := range args {
		if redactNext {
			out = append(out, "<redacted>")
			redactNext = false
			continue
		}

		name, _, hasValue := strings.Cut(arg, "=")
		if isSensitiveFlag(name) {
			if hasValue {
				out = append(out, name+"=<redacted>")
			} else {
				out = append(out, name)
				redactNext = true
			}
			continue
		}

		out = append(out, arg)
	}
	return out
}

func isSensitiveFlag(name string) bool {
	clean := strings.TrimLeft(name, "-")
	switch clean {
	case "password":
		return true
	default:
		return false
	}
}

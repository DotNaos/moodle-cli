package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func debugLogPath() string {
	return filepath.Join(filepath.Dir(opts.StatePath), "cli.log")
}

func errorLogPath() string {
	return filepath.Join(filepath.Dir(opts.StatePath), "error.log")
}

func appendDebugLog(scope string, lines ...string) (string, error) {
	logPath := debugLogPath()
	return appendLogFile(logPath, scope, lines...)
}

func appendErrorLog(scope string, lines ...string) (string, error) {
	logPath := errorLogPath()
	return appendLogFile(logPath, scope, lines...)
}

func appendLogFile(logPath string, scope string, lines ...string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return "", err
	}
	entry := []string{
		"---",
		"time: " + time.Now().Format(time.RFC3339),
		"scope: " + scope,
	}
	entry = append(entry, lines...)
	entry = append(entry, "")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := file.WriteString(strings.Join(entry, "\n")); err != nil {
		return "", err
	}
	return logPath, nil
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

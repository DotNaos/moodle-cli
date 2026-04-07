package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPresentUIErrorFiltersUnexpectedErrorAndWritesErrorLog(t *testing.T) {
	originalStatePath := opts.StatePath
	opts.StatePath = filepath.Join(t.TempDir(), "state.json")
	t.Cleanup(func() {
		opts.StatePath = originalStatePath
	})

	err := errors.New(`open failed for "/tmp/file": exit status 1 (details: /tmp/ignored.log)`)
	got := presentUIError("tui.open", err)
	if !strings.Contains(got, "Could not open the file.") {
		t.Fatalf("expected filtered open message, got %q", got)
	}
	if strings.Contains(got, "exit status 1") {
		t.Fatalf("expected raw error details to be hidden, got %q", got)
	}
}

func TestPresentUIErrorKeepsExpectedErrorReadable(t *testing.T) {
	err := errors.New("calendar URL not set. Run: moodle config set --calendar-url <url>")
	got := presentUIError("tui.children", err)
	if got != err.Error() {
		t.Fatalf("expected expected error to pass through, got %q", got)
	}
}

func TestLogUnexpectedWritesSeparateErrorLog(t *testing.T) {
	originalStatePath := opts.StatePath
	opts.StatePath = filepath.Join(t.TempDir(), "state.json")
	t.Cleanup(func() {
		opts.StatePath = originalStatePath
	})

	logPath := logUnexpected("open", errors.New("exit status 1"), "target: /tmp/file.pdf")
	if logPath != filepath.Join(filepath.Dir(opts.StatePath), "error.log") {
		t.Fatalf("unexpected error log path %q", logPath)
	}
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("expected error log to be written: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "scope: open") || !strings.Contains(text, "target: /tmp/file.pdf") {
		t.Fatalf("expected error log content, got %q", text)
	}
}

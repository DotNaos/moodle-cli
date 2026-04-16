package cli

import (
	"errors"
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
	records, err := readLogRecords(logPath, "error")
	if err != nil {
		t.Fatalf("expected error log to be readable: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected one error record, got %#v", records)
	}
	if records[0].Scope != "open" {
		t.Fatalf("unexpected record scope: %#v", records[0])
	}
	foundTarget := false
	for _, field := range records[0].Fields {
		if field.Key == "target" && field.Value == "/tmp/file.pdf" {
			foundTarget = true
			break
		}
	}
	if !foundTarget {
		t.Fatalf("expected target field in error record, got %#v", records[0].Fields)
	}
}

func TestRedactSensitiveArgsHidesPasswords(t *testing.T) {
	args := []string{"login", "--username=user@example.com", "--password", "secret", "--password=another", "--other=ok"}
	got := redactSensitiveArgs(args)
	want := []string{"login", "--username=user@example.com", "--password", "<redacted>", "--password=<redacted>", "--other=ok"}
	if len(got) != len(want) {
		t.Fatalf("unexpected args length: %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("arg %d mismatch: got %q want %q (full: %#v)", i, got[i], want[i], got)
		}
	}
}

func TestAppendDebugLogWritesStructuredRecord(t *testing.T) {
	originalStatePath := opts.StatePath
	opts.StatePath = filepath.Join(t.TempDir(), "state.json")
	t.Cleanup(func() {
		opts.StatePath = originalStatePath
	})

	logPath, err := appendDebugLog("cli", "event: start", "command: moodle version")
	if err != nil {
		t.Fatalf("appendDebugLog: %v", err)
	}

	records, err := readLogRecords(logPath, "debug")
	if err != nil {
		t.Fatalf("read log records: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected one log record, got %#v", records)
	}
	if records[0].Level != "debug" || records[0].Scope != "cli" {
		t.Fatalf("unexpected record: %#v", records[0])
	}
}

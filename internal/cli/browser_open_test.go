package cli

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeBrowserURLRemovesResourceRedirectFlag(t *testing.T) {
	input := "https://moodle.fhgr.ch/mod/resource/view.php?id=956877&redirect=1"
	got := normalizeBrowserURL(input)
	want := "https://moodle.fhgr.ch/mod/resource/view.php?id=956877"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestNormalizeBrowserURLLeavesOtherURLsUntouched(t *testing.T) {
	input := "https://moodle.fhgr.ch/course/view.php?id=22583"
	got := normalizeBrowserURL(input)
	if got != input {
		t.Fatalf("expected %q, got %q", input, got)
	}
}

func TestOpenURLReturnsDetailedErrorAndWritesLog(t *testing.T) {
	originalRunner := browserOpenRunner
	originalStatePath := opts.StatePath
	t.Cleanup(func() {
		browserOpenRunner = originalRunner
		opts.StatePath = originalStatePath
	})

	tempDir := t.TempDir()
	opts.StatePath = filepath.Join(tempDir, "state.json")
	browserOpenRunner = func(cmd *exec.Cmd) ([]byte, error) {
		return []byte("LSOpenURLsWithRole() failed with error -10810"), errors.New("exit status 1")
	}

	err := openURL("/tmp/example.pdf")
	if err == nil {
		t.Fatalf("expected openURL to fail")
	}
	if !strings.Contains(err.Error(), "LSOpenURLsWithRole() failed with error -10810") {
		t.Fatalf("expected detailed stderr in error, got %q", err.Error())
	}
	logPath := filepath.Join(tempDir, "cli.log")
	records, readErr := readLogRecords(logPath, "debug")
	if readErr != nil {
		t.Fatalf("expected open log to be written: %v", readErr)
	}
	if len(records) == 0 {
		t.Fatalf("expected at least one log record")
	}
	if records[0].Scope != "open" {
		t.Fatalf("unexpected log scope: %#v", records[0])
	}
	foundTarget := false
	for _, field := range records[0].Fields {
		if field.Key == "target" && field.Value == "/tmp/example.pdf" {
			foundTarget = true
			break
		}
	}
	if !foundTarget {
		t.Fatalf("expected log to contain target path, got %#v", records[0].Fields)
	}
}

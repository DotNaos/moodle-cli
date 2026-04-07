package cli

import (
	"errors"
	"os"
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
	content, readErr := os.ReadFile(logPath)
	if readErr != nil {
		t.Fatalf("expected open log to be written: %v", readErr)
	}
	if !strings.Contains(string(content), "/tmp/example.pdf") || !strings.Contains(string(content), "scope: open") {
		t.Fatalf("expected log to contain target path, got %q", string(content))
	}
}

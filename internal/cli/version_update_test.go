package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestVersionCommandOutput(t *testing.T) {
	var output bytes.Buffer
	versionCmd.SetOut(&output)

	if err := versionCmd.RunE(versionCmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rendered := output.String()
	for _, want := range []string{"version:", "commit:", "buildDate:"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected %q in output, got %q", want, rendered)
		}
	}
}

func TestShouldCheckForUpdatesSkipsVersionAndUpdateCommands(t *testing.T) {
	original := isInteractiveTerminal
	isInteractiveTerminal = func(file *os.File) bool { return true }
	defer func() { isInteractiveTerminal = original }()

	if shouldCheckForUpdates(versionCmd) {
		t.Fatal("expected version command to skip background checks")
	}
	if shouldCheckForUpdates(updateCmd) {
		t.Fatal("expected update command to skip background checks")
	}
}

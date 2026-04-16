package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTailLogFileFollowsAppends(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "cli.log")
	if err := os.WriteFile(path, []byte("first\n"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	var buf bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	prevInterval := logTailPollInterval
	logTailPollInterval = 25 * time.Millisecond
	t.Cleanup(func() { logTailPollInterval = prevInterval })

	done := make(chan error, 1)
	go func() {
		done <- tailLogFile(ctx, path, 10, true, &buf)
	}()

	time.Sleep(50 * time.Millisecond)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open for append: %v", err)
	}
	if _, err := file.WriteString("second\nthird\n"); err != nil {
		t.Fatalf("append: %v", err)
	}
	file.Close()

	time.Sleep(75 * time.Millisecond)
	cancel()

	if err := <-done; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("tailLogFile: %v", err)
	}

	text := buf.String()
	if !strings.Contains(text, "first") || !strings.Contains(text, "second") || !strings.Contains(text, "third") {
		t.Fatalf("expected tailed output, got %q", text)
	}
}

func TestLogsCommandSupportsMachineReadableOutput(t *testing.T) {
	reset := saveOutputFlagState()
	defer reset()

	outputJSON = true
	logsFollow = false
	logsErrors = false
	logsLines = 10

	previousStatePath := opts.StatePath
	opts.StatePath = filepath.Join(t.TempDir(), "state.json")
	t.Cleanup(func() { opts.StatePath = previousStatePath })

	if err := os.WriteFile(debugLogPath(), []byte("alpha\nbeta\n"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	var out bytes.Buffer
	logsCmd.SetOut(&out)
	t.Cleanup(func() { logsCmd.SetOut(nil) })

	if err := logsCmd.RunE(logsCmd, nil); err != nil {
		t.Fatalf("logs command: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected meta + 2 line events, got %d: %q", len(lines), out.String())
	}

	var meta logsEvent
	if err := json.Unmarshal([]byte(lines[0]), &meta); err != nil {
		t.Fatalf("decode meta: %v", err)
	}
	if meta.Type != "meta" || meta.Path == "" || meta.Label != "debug" {
		t.Fatalf("unexpected meta event: %#v", meta)
	}

	var first logsEvent
	if err := json.Unmarshal([]byte(lines[1]), &first); err != nil {
		t.Fatalf("decode line: %v", err)
	}
	if first.Type != "line" || first.Line != "alpha" {
		t.Fatalf("unexpected first line event: %#v", first)
	}
}

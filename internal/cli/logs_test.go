package cli

import (
	"bytes"
	"context"
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

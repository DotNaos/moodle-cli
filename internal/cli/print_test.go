package cli

import (
	"strings"
	"testing"
	"time"
)

func TestCleanExtractedTextWithTimeoutFallsBackToRaw(t *testing.T) {
	input := "  raw text  "
	got := cleanExtractedTextWithTimeout(input, 0)
	if strings.TrimSpace(got) != "raw text" {
		t.Fatalf("expected raw fallback, got %q", got)
	}
}

func TestCleanExtractedTextWithTimeoutCleansWhenAllowed(t *testing.T) {
	input := "hello-\nworld"
	got := cleanExtractedTextWithTimeout(input, 50*time.Millisecond)
	if got != "helloworld" {
		t.Fatalf("expected cleaned text, got %q", got)
	}
}

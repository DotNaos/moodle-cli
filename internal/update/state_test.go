package update

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStateRoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "state.json")
	input := State{
		LastUpdateCheckAt: time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC),
		LastNotifiedTag:   "v1.2.3",
	}

	if err := SaveState(path, input); err != nil {
		t.Fatalf("save state: %v", err)
	}
	output, err := LoadState(path)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if !output.LastUpdateCheckAt.Equal(input.LastUpdateCheckAt) {
		t.Fatalf("expected %v, got %v", input.LastUpdateCheckAt, output.LastUpdateCheckAt)
	}
	if output.LastNotifiedTag != input.LastNotifiedTag {
		t.Fatalf("expected %q, got %q", input.LastNotifiedTag, output.LastNotifiedTag)
	}
}

func TestShouldCheck(t *testing.T) {
	now := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)

	if !ShouldCheck(State{}, now, 24*time.Hour) {
		t.Fatal("expected empty state to require check")
	}
	if ShouldCheck(State{LastUpdateCheckAt: now.Add(-2 * time.Hour)}, now, 24*time.Hour) {
		t.Fatal("expected recent check to be skipped")
	}
	if !ShouldCheck(State{LastUpdateCheckAt: now.Add(-25 * time.Hour)}, now, 24*time.Hour) {
		t.Fatal("expected old check to run")
	}
}

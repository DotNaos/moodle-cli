package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateLegacyHomeCopiesDataNonDestructively(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MOODLE_HOME", "")
	t.Setenv("MOODLE_CLI_HOME", "")

	legacy := filepath.Join(home, ".moodle-cli")
	target := filepath.Join(home, ".moodle")
	if err := os.MkdirAll(filepath.Join(legacy, "files"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "mobile-session.json"), []byte(`{"token":"secret"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "files", "one.txt"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := migrateLegacyHome()
	if err != nil {
		t.Fatalf("migrateLegacyHome: %v", err)
	}
	if result.Status != "migrated" {
		t.Fatalf("unexpected status %q", result.Status)
	}
	if result.CopiedFiles != 2 {
		t.Fatalf("expected 2 copied files, got %d", result.CopiedFiles)
	}
	if _, err := os.Stat(filepath.Join(legacy, "mobile-session.json")); err != nil {
		t.Fatalf("expected legacy file to remain: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(target, "files", "one.txt")); err != nil || string(data) != "content" {
		t.Fatalf("unexpected copied file: %q %v", data, err)
	}
}

func TestMigrateLegacyHomeRefusesNonEmptyTarget(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MOODLE_HOME", "")
	t.Setenv("MOODLE_CLI_HOME", "")

	legacy := filepath.Join(home, ".moodle-cli")
	target := filepath.Join(home, ".moodle")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "config.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := migrateLegacyHome(); err == nil {
		t.Fatalf("expected non-empty target to fail")
	}
}

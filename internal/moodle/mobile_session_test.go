package moodle

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestSaveMobileSessionUsesPrivatePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mobile-session.json")
	session := MobileSession{
		SiteURL:   "https://moodle.example.test",
		UserID:    42,
		Token:     "test-token",
		CreatedAt: time.Unix(100, 0).UTC(),
	}

	if err := SaveMobileSession(path, session); err != nil {
		t.Fatalf("SaveMobileSession: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat mobile session: %v", err)
	}
	if got := info.Mode().Perm(); runtime.GOOS != "windows" && got != 0o600 {
		t.Fatalf("expected 0600 permissions, got %o", got)
	}

	loaded, err := LoadMobileSession(path)
	if err != nil {
		t.Fatalf("LoadMobileSession: %v", err)
	}
	if loaded.Token != session.Token {
		t.Fatalf("unexpected token %q", loaded.Token)
	}
}

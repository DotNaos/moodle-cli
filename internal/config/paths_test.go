package config

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestBaseDirDefaultsToSharedMoodleHome(t *testing.T) {
	home := t.TempDir()
	setTestHome(t, home)
	t.Setenv("MOODLE_HOME", "")
	t.Setenv("MOODLE_CLI_HOME", "")

	if got, want := BaseDir(), filepath.Join(home, ".moodle"); got != want {
		t.Fatalf("BaseDir() = %q, want %q", got, want)
	}
}

func TestBaseDirPrefersMoodleHome(t *testing.T) {
	home := t.TempDir()
	legacy := t.TempDir()
	t.Setenv("MOODLE_HOME", home)
	t.Setenv("MOODLE_CLI_HOME", legacy)

	if got, want := BaseDir(), home; got != want {
		t.Fatalf("BaseDir() = %q, want %q", got, want)
	}
}

func TestBaseDirFallsBackToLegacyMoodleCLIHome(t *testing.T) {
	legacy := t.TempDir()
	t.Setenv("MOODLE_HOME", "")
	t.Setenv("MOODLE_CLI_HOME", legacy)

	if got, want := BaseDir(), legacy; got != want {
		t.Fatalf("BaseDir() = %q, want %q", got, want)
	}
}

func TestMobileSessionPathUsesBaseDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("MOODLE_HOME", home)
	t.Setenv("MOODLE_CLI_HOME", "")

	if got, want := MobileSessionPath(), filepath.Join(home, "mobile-session.json"); got != want {
		t.Fatalf("MobileSessionPath() = %q, want %q", got, want)
	}
}

func setTestHome(t *testing.T, home string) {
	t.Helper()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
		t.Setenv("HOMEDRIVE", "")
		t.Setenv("HOMEPATH", "")
	}
}

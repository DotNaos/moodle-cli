package config

import (
	"path/filepath"
	"testing"
)

func TestBaseDirDefaultsToSharedMoodleHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MOODLE_HOME", "")
	t.Setenv("MOODLE_CLI_HOME", "")

	if got, want := BaseDir(), filepath.Join(home, ".moodle"); got != want {
		t.Fatalf("BaseDir() = %q, want %q", got, want)
	}
}

func TestBaseDirPrefersMoodleHome(t *testing.T) {
	t.Setenv("MOODLE_HOME", "/tmp/moodle-home")
	t.Setenv("MOODLE_CLI_HOME", "/tmp/legacy-home")

	if got, want := BaseDir(), "/tmp/moodle-home"; got != want {
		t.Fatalf("BaseDir() = %q, want %q", got, want)
	}
}

func TestBaseDirFallsBackToLegacyMoodleCLIHome(t *testing.T) {
	t.Setenv("MOODLE_HOME", "")
	t.Setenv("MOODLE_CLI_HOME", "/tmp/legacy-home")

	if got, want := BaseDir(), "/tmp/legacy-home"; got != want {
		t.Fatalf("BaseDir() = %q, want %q", got, want)
	}
}

func TestMobileSessionPathUsesBaseDir(t *testing.T) {
	t.Setenv("MOODLE_HOME", "/tmp/moodle-home")
	t.Setenv("MOODLE_CLI_HOME", "")

	if got, want := MobileSessionPath(), "/tmp/moodle-home/mobile-session.json"; got != want {
		t.Fatalf("MobileSessionPath() = %q, want %q", got, want)
	}
}

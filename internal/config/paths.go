package config

import (
	"os"
	"path/filepath"
)

const (
	envBaseDir       = "MOODLE_HOME"
	envLegacyBaseDir = "MOODLE_CLI_HOME"
	envExportDir     = "MOODLE_CLI_EXPORT_DIR"
)

func baseDir() string {
	if v := os.Getenv(envBaseDir); v != "" {
		return v
	}
	if v := os.Getenv(envLegacyBaseDir); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".moodle")
}

func BaseDir() string       { return baseDir() }
func LegacyBaseDir() string { home, _ := os.UserHomeDir(); return filepath.Join(home, ".moodle-cli") }

func ConfigPath() string   { return filepath.Join(baseDir(), "config.json") }
func SessionPath() string  { return filepath.Join(baseDir(), "session.json") }
func CacheDBPath() string  { return filepath.Join(baseDir(), "cache.db") }
func FileCacheDir() string { return filepath.Join(baseDir(), "files") }
func StatePath() string    { return filepath.Join(baseDir(), "state.json") }
func MobileSessionPath() string {
	return filepath.Join(baseDir(), "mobile-session.json")
}

func ExportDir() string {
	if v := os.Getenv(envExportDir); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Downloads", "moodle")
}

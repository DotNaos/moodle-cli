package cli

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
)

var browserOpenRunner = func(cmd *exec.Cmd) ([]byte, error) {
	return cmd.CombinedOutput()
}

func openURL(url string) error {
	normalized := normalizeBrowserURL(url)
	cmd, err := browserOpenCommand(runtime.GOOS, normalized)
	if err != nil {
		return err
	}
	output, err := browserOpenRunner(cmd)
	if err == nil {
		logDebug("open", "target: "+normalized, "command: "+strings.Join(cmd.Args, " "), "result: success")
		return nil
	}
	stderr := strings.TrimSpace(string(output))
	logDebug(
		"open",
		"os: "+runtime.GOOS,
		"target: "+normalized,
		"command: "+strings.Join(cmd.Args, " "),
		"error: "+err.Error(),
		"output: "+stderr,
	)
	logPath := logUnexpected(
		"open",
		err,
		"os: "+runtime.GOOS,
		"target: "+normalized,
		"command: "+strings.Join(cmd.Args, " "),
		"output: "+stderr,
	)
	if stderr != "" {
		return fmt.Errorf("open failed for %q: %s (details: %s)", normalized, stderr, logPath)
	}
	return fmt.Errorf("open failed for %q: %w (details: %s)", normalized, err, logPath)
}

func browserOpenCommand(goos string, url string) (*exec.Cmd, error) {
	switch goos {
	case "darwin":
		return exec.Command("open", url), nil
	case "linux":
		return exec.Command("xdg-open", url), nil
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url), nil
	default:
		return nil, fmt.Errorf("opening URLs is not supported on %s", goos)
	}
}

func normalizeBrowserURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return raw
	}
	if !strings.Contains(parsed.Path, "/mod/resource/view.php") {
		return raw
	}
	query := parsed.Query()
	if query.Get("redirect") != "1" {
		return raw
	}
	query.Del("redirect")
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

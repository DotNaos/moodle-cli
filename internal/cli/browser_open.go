package cli

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
)

func openURL(url string) error {
	cmd, err := browserOpenCommand(runtime.GOOS, normalizeBrowserURL(url))
	if err != nil {
		return err
	}
	return cmd.Run()
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

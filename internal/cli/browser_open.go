package cli

import (
	"fmt"
	"os/exec"
	"runtime"
)

func openURL(url string) error {
	cmd, err := browserOpenCommand(runtime.GOOS, url)
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

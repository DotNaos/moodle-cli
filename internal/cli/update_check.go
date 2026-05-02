package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/DotNaos/moodle-services/internal/update"
	ver "github.com/DotNaos/moodle-services/internal/version"
	"github.com/spf13/cobra"
)

func maybeCheckForUpdates(cmd *cobra.Command) error {
	if !shouldCheckForUpdates(cmd) {
		return nil
	}

	state, err := update.LoadState(opts.StatePath)
	if err != nil {
		return nil
	}
	now := time.Now()
	if !update.ShouldCheck(state, now, update.DefaultCheckInterval) {
		return nil
	}

	state.LastUpdateCheckAt = now
	if err := update.SaveState(opts.StatePath, state); err != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Second)
	defer cancel()

	client := update.NewClient()
	availability, _, err := client.Check(ctx, ver.Version())
	if err != nil {
		return nil
	}
	if availability.NeedsUpdate && state.LastNotifiedTag != availability.LatestTag {
		fmt.Fprintf(cmd.ErrOrStderr(), "Update available: %s -> %s. Run `moodle update`.\n", ver.Version(), availability.LatestTag)
		state.LastNotifiedTag = availability.LatestTag
		return update.SaveState(opts.StatePath, state)
	}
	if !availability.NeedsUpdate && state.LastNotifiedTag != "" {
		state.LastNotifiedTag = ""
		return update.SaveState(opts.StatePath, state)
	}
	return nil
}

func shouldCheckForUpdates(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	if isMachineOutput() {
		return false
	}
	if !isInteractiveTerminal(os.Stderr) {
		return false
	}
	if commandPathHas(cmd, "version") || commandPathHas(cmd, "update") || commandPathHas(cmd, "completion") {
		return false
	}
	return true
}

var isInteractiveTerminal = func(file *os.File) bool {
	if file == nil {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

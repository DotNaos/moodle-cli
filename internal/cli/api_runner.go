package cli

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
)

var currentExecutablePath = os.Executable

func runAPICommand(ctx context.Context, commandPath []string, arguments []string, stdout io.Writer, stderr io.Writer) error {
	executablePath, err := currentExecutablePath()
	if err != nil {
		return err
	}

	args := []string{"--json"}
	args = append(args, sharedCommandArgs()...)
	args = append(args, commandPath...)
	args = append(args, filterMachineOutputArgs(arguments)...)

	command := exec.CommandContext(ctx, executablePath, args...)
	command.Stdout = stdout
	command.Stderr = stderr
	return command.Run()
}

func sharedCommandArgs() []string {
	args := []string{}
	if value := strings.TrimSpace(opts.ConfigPath); value != "" {
		args = append(args, "--config", value)
	}
	if value := strings.TrimSpace(opts.SessionPath); value != "" {
		args = append(args, "--session", value)
	}
	if value := strings.TrimSpace(opts.CacheDBPath); value != "" {
		args = append(args, "--cache", value)
	}
	if value := strings.TrimSpace(opts.FileCacheDir); value != "" {
		args = append(args, "--files-cache", value)
	}
	if value := strings.TrimSpace(opts.StatePath); value != "" {
		args = append(args, "--state", value)
	}
	if value := strings.TrimSpace(opts.ExportDir); value != "" {
		args = append(args, "--output-dir", value)
	}
	if opts.Unsanitized {
		args = append(args, "--unsanitized")
	}
	return args
}

func filterMachineOutputArgs(arguments []string) []string {
	filtered := make([]string, 0, len(arguments))
	for _, argument := range arguments {
		switch strings.TrimSpace(argument) {
		case "--json", "--yaml", "--yml":
			continue
		default:
			filtered = append(filtered, argument)
		}
	}
	return filtered
}

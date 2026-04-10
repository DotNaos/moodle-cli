package cli

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type helpResult struct {
	Command          string            `json:"command" yaml:"command"`
	Use              string            `json:"use" yaml:"use"`
	Short            string            `json:"short,omitempty" yaml:"short,omitempty"`
	Long             string            `json:"long,omitempty" yaml:"long,omitempty"`
	Example          string            `json:"example,omitempty" yaml:"example,omitempty"`
	Subcommands      []helpCommandInfo `json:"subcommands,omitempty" yaml:"subcommands,omitempty"`
	Flags            []helpFlagInfo    `json:"flags,omitempty" yaml:"flags,omitempty"`
	InheritedFlags   []helpFlagInfo    `json:"inheritedFlags,omitempty" yaml:"inheritedFlags,omitempty"`
	InteractiveOnly  bool              `json:"interactiveOnly" yaml:"interactiveOnly"`
	MachineSupported bool              `json:"machineSupported" yaml:"machineSupported"`
}

type helpCommandInfo struct {
	Name   string `json:"name" yaml:"name"`
	Use    string `json:"use" yaml:"use"`
	Short  string `json:"short,omitempty" yaml:"short,omitempty"`
	Hidden bool   `json:"hidden,omitempty" yaml:"hidden,omitempty"`
}

type helpFlagInfo struct {
	Name         string `json:"name" yaml:"name"`
	Shorthand    string `json:"shorthand,omitempty" yaml:"shorthand,omitempty"`
	Usage        string `json:"usage,omitempty" yaml:"usage,omitempty"`
	DefaultValue string `json:"defaultValue,omitempty" yaml:"defaultValue,omitempty"`
}

func installMachineHelp() {
	defaultHelpFunc := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if !isMachineOutput() {
			defaultHelpFunc(cmd, args)
			return
		}
		_ = writeStructuredPayload(cmd.OutOrStdout(), buildHelpResult(cmd))
	})
}

func buildHelpResult(cmd *cobra.Command) helpResult {
	result := helpResult{
		Command:          cmd.CommandPath(),
		Use:              cmd.UseLine(),
		Short:            cmd.Short,
		Long:             cmd.Long,
		Example:          cmd.Example,
		Subcommands:      []helpCommandInfo{},
		Flags:            collectVisibleFlags(cmd.NonInheritedFlags()),
		InheritedFlags:   collectVisibleFlags(cmd.InheritedFlags()),
		InteractiveOnly:  isInteractiveOnly(cmd),
		MachineSupported: !isInteractiveOnly(cmd),
	}

	for _, sub := range cmd.Commands() {
		if !sub.IsAvailableCommand() {
			continue
		}
		result.Subcommands = append(result.Subcommands, helpCommandInfo{
			Name:   sub.Name(),
			Use:    sub.UseLine(),
			Short:  sub.Short,
			Hidden: sub.Hidden,
		})
	}
	return result
}

func collectVisibleFlags(flags *pflag.FlagSet) []helpFlagInfo {
	if flags == nil {
		return nil
	}
	out := []helpFlagInfo{}
	flags.VisitAll(func(flag *pflag.Flag) {
		if flag.Hidden {
			return
		}
		out = append(out, helpFlagInfo{
			Name:         flag.Name,
			Shorthand:    flag.Shorthand,
			Usage:        flag.Usage,
			DefaultValue: flag.DefValue,
		})
	})
	return out
}

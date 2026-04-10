package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type outputFormat string

const (
	outputFormatText outputFormat = "text"
	outputFormatJSON outputFormat = "json"
	outputFormatYAML outputFormat = "yaml"
)

const interactiveOnlyAnnotation = "interactiveOnly"

var outputJSON bool
var outputYAML bool
var outputYML bool

type commandError struct {
	Code    string `json:"code,omitempty" yaml:"code,omitempty"`
	Error   string `json:"error" yaml:"error"`
	Command string `json:"command,omitempty" yaml:"command,omitempty"`
}

type codedError struct {
	code string
	msg  string
}

type emittedError struct {
	err error
}

func (e codedError) Error() string {
	return e.msg
}

func (e emittedError) Error() string {
	return e.err.Error()
}

func (e emittedError) Unwrap() error {
	return e.err
}

func machineCommandError(code string, msg string) error {
	return codedError{code: code, msg: msg}
}

func markErrorEmitted(err error) error {
	if err == nil {
		return nil
	}
	return emittedError{err: err}
}

func currentOutputFormat() outputFormat {
	switch {
	case outputJSON:
		return outputFormatJSON
	case outputYAML || outputYML:
		return outputFormatYAML
	default:
		return outputFormatText
	}
}

func isMachineOutput() bool {
	return currentOutputFormat() != outputFormatText
}

func validateOutputFlags() error {
	selected := 0
	if outputJSON {
		selected++
	}
	if outputYAML {
		selected++
	}
	if outputYML {
		selected++
	}
	if selected <= 1 {
		return nil
	}
	return machineCommandError("output_format_conflict", "choose exactly one of --json, --yaml, or --yml")
}

func markInteractiveOnly(cmd *cobra.Command) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[interactiveOnlyAnnotation] = "true"
}

func isInteractiveOnly(cmd *cobra.Command) bool {
	if cmd == nil || cmd.Annotations == nil {
		return false
	}
	return cmd.Annotations[interactiveOnlyAnnotation] == "true"
}

func ensureMachineOutputAllowed(cmd *cobra.Command) error {
	if !isMachineOutput() || cmd == nil {
		return nil
	}
	if isInteractiveOnly(cmd) {
		return machineCommandError("interactive_command", fmt.Sprintf("%s does not support machine-readable output", cmd.CommandPath()))
	}
	return nil
}

func writeCommandOutput(cmd *cobra.Command, payload any, renderText func(io.Writer) error) error {
	if isMachineOutput() {
		return writeStructuredPayload(cmd.OutOrStdout(), payload)
	}
	return renderText(cmd.OutOrStdout())
}

func writeStructuredPayload(writer io.Writer, payload any) error {
	data, err := marshalPayload(payload, currentOutputFormat(), true)
	if err != nil {
		return err
	}
	if _, err := writer.Write(data); err != nil {
		return err
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		_, err = writer.Write([]byte("\n"))
	}
	return err
}

func writeStreamEvent(writer io.Writer, payload any) error {
	switch currentOutputFormat() {
	case outputFormatJSON:
		data, err := marshalPayload(payload, outputFormatJSON, false)
		if err != nil {
			return err
		}
		if _, err := writer.Write(data); err != nil {
			return err
		}
		_, err = writer.Write([]byte("\n"))
		return err
	case outputFormatYAML:
		data, err := marshalPayload(payload, outputFormatYAML, false)
		if err != nil {
			return err
		}
		if _, err := writer.Write([]byte("---\n")); err != nil {
			return err
		}
		if _, err := writer.Write(data); err != nil {
			return err
		}
		if len(data) == 0 || data[len(data)-1] != '\n' {
			_, err = writer.Write([]byte("\n"))
			return err
		}
		return nil
	default:
		return fmt.Errorf("stream events require machine-readable output")
	}
}

func marshalPayload(payload any, format outputFormat, pretty bool) ([]byte, error) {
	switch format {
	case outputFormatJSON:
		if pretty {
			return json.MarshalIndent(payload, "", "  ")
		}
		return json.Marshal(payload)
	case outputFormatYAML:
		data, err := yaml.Marshal(payload)
		if err != nil {
			return nil, err
		}
		return bytes.TrimRight(data, "\n"), nil
	default:
		return nil, fmt.Errorf("unsupported structured output format: %s", format)
	}
}

func writeCommandError(err error) {
	if err == nil {
		return
	}
	var emitted emittedError
	if errors.As(err, &emitted) {
		return
	}
	if isMachineOutput() {
		payload := commandError{
			Error:   err.Error(),
			Command: rootCmd.CommandPath(),
		}
		if coded, ok := err.(codedError); ok {
			payload.Code = coded.code
			payload.Error = coded.msg
		}
		_ = writeStructuredPayload(rootCmd.OutOrStdout(), payload)
		return
	}
	fmt.Fprintln(rootCmd.ErrOrStderr(), err.Error())
}

func helpOrMachineError(cmd *cobra.Command, msg string) error {
	if isMachineOutput() {
		return machineCommandError("invalid_arguments", msg)
	}
	return cmd.Help()
}

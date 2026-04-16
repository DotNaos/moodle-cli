package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogsCommandSupportsMachineReadableOutput(t *testing.T) {
	reset := saveOutputFlagState()
	defer reset()

	outputJSON = true
	logsFollow = false
	logsErrors = false
	logsLines = 10

	previousStatePath := opts.StatePath
	opts.StatePath = filepath.Join(t.TempDir(), "state.json")
	t.Cleanup(func() { opts.StatePath = previousStatePath })

	if _, err := appendDebugLog("cli", "event: start", "command: list courses"); err != nil {
		t.Fatalf("append debug log: %v", err)
	}

	var out bytes.Buffer
	logsCmd.SetOut(&out)
	t.Cleanup(func() { logsCmd.SetOut(nil) })

	if err := logsCmd.RunE(logsCmd, nil); err != nil {
		t.Fatalf("logs command: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected meta + 1 record event, got %d: %q", len(lines), out.String())
	}

	var meta logsEvent
	if err := json.Unmarshal([]byte(lines[0]), &meta); err != nil {
		t.Fatalf("decode meta: %v", err)
	}
	if meta.Type != "meta" || meta.Path == "" || meta.Label != "debug" {
		t.Fatalf("unexpected meta event: %#v", meta)
	}

	var recordEvent logsEvent
	if err := json.Unmarshal([]byte(lines[1]), &recordEvent); err != nil {
		t.Fatalf("decode record event: %v", err)
	}
	if recordEvent.Type != "record" || recordEvent.Record == nil {
		t.Fatalf("unexpected record event: %#v", recordEvent)
	}
	if recordEvent.Record.Scope != "cli" {
		t.Fatalf("unexpected record scope: %#v", recordEvent.Record)
	}
	if len(recordEvent.Record.Fields) < 2 || recordEvent.Record.Fields[0].Key != "event" || recordEvent.Record.Fields[0].Value != "start" {
		t.Fatalf("unexpected record fields: %#v", recordEvent.Record.Fields)
	}
}

func TestLogsCommandTextOutputRendersWholeRecords(t *testing.T) {
	reset := saveOutputFlagState()
	defer reset()

	logsFollow = false
	logsErrors = false
	logsLines = 10

	previousStatePath := opts.StatePath
	opts.StatePath = filepath.Join(t.TempDir(), "state.json")
	t.Cleanup(func() { opts.StatePath = previousStatePath })

	if _, err := appendDebugLog("open", "target: /tmp/example.pdf", "result: success"); err != nil {
		t.Fatalf("append debug log: %v", err)
	}

	var out bytes.Buffer
	logsCmd.SetOut(&out)
	t.Cleanup(func() { logsCmd.SetOut(nil) })

	if err := logsCmd.RunE(logsCmd, nil); err != nil {
		t.Fatalf("logs command: %v", err)
	}

	text := out.String()
	if !strings.Contains(text, "last 10 entries") {
		t.Fatalf("expected entry-based header, got %q", text)
	}
	if !strings.Contains(text, "[DEBUG] open") {
		t.Fatalf("expected rendered log header, got %q", text)
	}
	if !strings.Contains(text, "target: /tmp/example.pdf") {
		t.Fatalf("expected record fields, got %q", text)
	}
}

func TestReadLogRecordsSupportsLegacyBlocks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cli.log")
	legacy := strings.Join([]string{
		"---",
		"time: 2026-04-16T08:00:00Z",
		"scope: cli",
		"event: start",
		"command: moodle list courses",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatalf("write legacy log: %v", err)
	}

	records, err := readLogRecords(path, "debug")
	if err != nil {
		t.Fatalf("read log records: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected one legacy record, got %#v", records)
	}
	if records[0].Scope != "cli" || records[0].Timestamp != "2026-04-16T08:00:00Z" {
		t.Fatalf("unexpected legacy record: %#v", records[0])
	}
	if len(records[0].Fields) != 2 || records[0].Fields[0].Key != "event" {
		t.Fatalf("unexpected legacy fields: %#v", records[0].Fields)
	}
}

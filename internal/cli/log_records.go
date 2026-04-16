package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type logField struct {
	Key   string `json:"key" yaml:"key"`
	Value string `json:"value" yaml:"value"`
}

type logRecord struct {
	Timestamp string     `json:"timestamp" yaml:"timestamp"`
	Level     string     `json:"level" yaml:"level"`
	Scope     string     `json:"scope" yaml:"scope"`
	Message   string     `json:"message,omitempty" yaml:"message,omitempty"`
	Fields    []logField `json:"fields,omitempty" yaml:"fields,omitempty"`
}

func newLogRecord(level string, scope string, lines []string) logRecord {
	record := logRecord{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     strings.TrimSpace(level),
		Scope:     strings.TrimSpace(scope),
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		key, value, ok := strings.Cut(trimmed, ": ")
		if ok {
			record.Fields = append(record.Fields, logField{
				Key:   strings.TrimSpace(key),
				Value: strings.TrimSpace(value),
			})
			continue
		}
		if record.Message == "" {
			record.Message = trimmed
			continue
		}
		record.Message += "\n" + trimmed
	}
	return record
}

func appendLogRecord(logPath string, record logRecord) (string, error) {
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return "", err
	}
	data, err := json.Marshal(record)
	if err != nil {
		return "", err
	}
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := file.Write(append(data, '\n')); err != nil {
		return "", err
	}
	return logPath, nil
}

func readLogRecords(path string, defaultLevel string) ([]logRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	records := []logRecord{}
	legacyLines := []string{}
	legacyMode := false

	flushLegacy := func() {
		record, ok := parseLegacyLogRecord(legacyLines, defaultLevel)
		if ok {
			records = append(records, record)
		}
		legacyLines = legacyLines[:0]
		legacyMode = false
	}

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if legacyMode {
			if trimmed == "" {
				flushLegacy()
				continue
			}
			legacyLines = append(legacyLines, line)
			continue
		}

		if trimmed == "" {
			continue
		}
		if trimmed == "---" {
			legacyMode = true
			legacyLines = legacyLines[:0]
			continue
		}

		var record logRecord
		if err := json.Unmarshal([]byte(trimmed), &record); err == nil && record.Timestamp != "" {
			if record.Level == "" {
				record.Level = defaultLevel
			}
			records = append(records, record)
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if legacyMode {
		flushLegacy()
	}
	return records, nil
}

func parseLegacyLogRecord(lines []string, defaultLevel string) (logRecord, bool) {
	if len(lines) == 0 {
		return logRecord{}, false
	}
	record := logRecord{Level: defaultLevel}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		key, value, ok := strings.Cut(trimmed, ": ")
		if !ok {
			if record.Message == "" {
				record.Message = trimmed
			} else {
				record.Message += "\n" + trimmed
			}
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch key {
		case "time":
			record.Timestamp = value
		case "scope":
			record.Scope = value
		default:
			record.Fields = append(record.Fields, logField{Key: key, Value: value})
		}
	}
	if record.Timestamp == "" {
		record.Timestamp = time.Now().Format(time.RFC3339)
	}
	return record, record.Scope != "" || record.Message != "" || len(record.Fields) > 0
}

func streamLogRecords(ctx context.Context, path string, level string, limit int, follow bool, emit func(logRecord) error) error {
	emitted := 0
	ticker := time.NewTicker(logTailPollInterval)
	defer ticker.Stop()

	for {
		if err := ensureLogFilePresent(path); err != nil {
			return err
		}
		records, err := readLogRecords(path, level)
		if err != nil {
			return err
		}

		start := emitted
		if emitted == 0 {
			start = len(records) - limit
			if start < 0 {
				start = 0
			}
		} else if len(records) < emitted {
			start = 0
		}

		for _, record := range records[start:] {
			if err := emit(record); err != nil {
				return err
			}
		}
		emitted = len(records)

		if !follow {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func renderLogRecord(w io.Writer, record logRecord) error {
	if _, err := fmt.Fprintf(w, "%s [%s] %s\n", record.Timestamp, strings.ToUpper(record.Level), record.Scope); err != nil {
		return err
	}
	if record.Message != "" {
		for _, line := range strings.Split(record.Message, "\n") {
			if _, err := fmt.Fprintf(w, "  %s\n", line); err != nil {
				return err
			}
		}
	}
	for _, field := range record.Fields {
		if _, err := fmt.Fprintf(w, "  %s: %s\n", field.Key, field.Value); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return err
	}
	return nil
}

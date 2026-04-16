package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
)

func TestSkillCommandPrintsEmbeddedSkill(t *testing.T) {
	reset := saveOutputFlagState()
	defer reset()

	skillInstall = false
	skillInstallAgents = nil

	text, err := readEmbeddedSkill()
	if err != nil {
		t.Fatalf("readEmbeddedSkill: %v", err)
	}

	var out bytes.Buffer
	skillCmd.SetOut(&out)
	t.Cleanup(func() { skillCmd.SetOut(nil) })

	if err := skillCmd.RunE(skillCmd, nil); err != nil {
		t.Fatalf("skill command: %v", err)
	}

	got := out.String()
	if got != text+"\n" && got != text {
		t.Fatalf("unexpected skill output: %q", got)
	}
}

func TestNormalizeAgentsDeduplicatesAndSorts(t *testing.T) {
	input := []string{"Codex", "claude-code", "codex", " gemini-cli "}
	got := normalizeAgents(input)
	want := []string{"claude-code", "codex", "gemini-cli"}
	if len(got) != len(want) {
		t.Fatalf("unexpected length: %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d mismatch: got %q want %q (full %#v)", i, got[i], want[i], got)
		}
	}
}

func TestSkillInstallKeepsMachineReadableStdoutClean(t *testing.T) {
	reset := saveOutputFlagState()
	defer reset()

	outputJSON = true
	skillInstall = true
	skillInstallAgents = nil
	t.Cleanup(func() {
		skillInstall = false
		skillInstallAgents = nil
	})

	previousRunner := skillInstallCommand
	skillInstallCommand = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		commandArgs := append([]string{"-test.run=TestSkillInstallHelperProcess", "--", name}, args...)
		cmd := exec.CommandContext(ctx, os.Args[0], commandArgs...)
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
		return cmd
	}
	t.Cleanup(func() { skillInstallCommand = previousRunner })

	var out bytes.Buffer
	var errOut bytes.Buffer
	skillCmd.SetOut(&out)
	skillCmd.SetErr(&errOut)
	t.Cleanup(func() {
		skillCmd.SetOut(nil)
		skillCmd.SetErr(nil)
	})

	if err := skillCmd.RunE(skillCmd, nil); err != nil {
		t.Fatalf("skill command: %v", err)
	}

	if bytes.HasPrefix(bytes.TrimSpace(out.Bytes()), []byte("installer output")) {
		t.Fatalf("expected stdout to remain machine-readable, got %q", out.String())
	}
	if !bytes.Contains(errOut.Bytes(), []byte("installer output")) {
		t.Fatalf("expected installer output on stderr, got %q", errOut.String())
	}
	if out.Len() == 0 || out.Bytes()[0] != '{' {
		t.Fatalf("expected structured JSON output, got %q", out.String())
	}
}

func TestSkillInstallHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	fmt.Fprintln(os.Stdout, "installer output")
	fmt.Fprintln(os.Stderr, "installer warning")
	os.Exit(0)
}

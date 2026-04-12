package cli

import (
	"bytes"
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

package cli

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/DotNaos/moodle-cli/skills"
	"github.com/spf13/cobra"
)

var (
	skillInstall       bool
	skillInstallAgents []string
)

type skillCommandResult struct {
	Skill     string                `json:"skill" yaml:"skill"`
	Installed *skillInstallResponse `json:"installed,omitempty" yaml:"installed,omitempty"`
}

type skillInstallResponse struct {
	Agents []string `json:"agents" yaml:"agents"`
	Source string   `json:"source" yaml:"source"`
}

var defaultSkillAgents = []string{"codex", "opencode", "claude-code", "gemini-cli"}

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Show or install the moodle-cli skill",
	Long:  "Print the bundled moodle-cli agent skill or install it via the vercel-labs 'skills' CLI.",
	Args:  cobra.NoArgs,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		skillText, err := readEmbeddedSkill()
		if err != nil {
			return err
		}

		result := skillCommandResult{Skill: skillText}
		if skillInstall {
			agents := defaultSkillAgents
			if len(skillInstallAgents) > 0 {
				agents = skillInstallAgents
			}
			installResult, err := installEmbeddedSkill(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), agents)
			if err != nil {
				return err
			}
			result.Installed = installResult
		}

		return writeCommandOutput(cmd, result, func(w io.Writer) error {
			if _, err := io.WriteString(w, result.Skill); err != nil {
				return err
			}
			if !strings.HasSuffix(result.Skill, "\n") {
				if _, err := io.WriteString(w, "\n"); err != nil {
					return err
				}
			}
			if result.Installed != nil {
				if _, err := fmt.Fprintf(w, "\nInstalled moodle-cli skill to agents: %s\n", strings.Join(result.Installed.Agents, ", ")); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(w, "Source: %s\n", result.Installed.Source); err != nil {
					return err
				}
			}
			return nil
		})
	},
}

func init() {
	skillCmd.Flags().BoolVar(&skillInstall, "install", false, "Install the moodle-cli skill using 'npx skills add'")
	skillCmd.Flags().StringSliceVar(&skillInstallAgents, "agent", nil, "Target agents for skill install (default: codex, opencode, claude-code, gemini-cli)")
}

func readEmbeddedSkill() (string, error) {
	data, err := skills.FS.ReadFile(path.Join(skills.RootDir, "SKILL.md"))
	if err != nil {
		return "", fmt.Errorf("embedded skill missing: %w", err)
	}
	return string(data), nil
}

func installEmbeddedSkill(ctx context.Context, stdout io.Writer, stderr io.Writer, agents []string) (*skillInstallResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	uniqueAgents := normalizeAgents(agents)
	if len(uniqueAgents) == 0 {
		uniqueAgents = normalizeAgents(defaultSkillAgents)
	}

	tempDir, err := os.MkdirTemp("", "moodle-skill-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	if err := copyEmbeddedSkill(tempDir); err != nil {
		return nil, err
	}

	args := []string{"skills", "add", tempDir, "-g", "-y"}
	for _, agent := range uniqueAgents {
		args = append(args, "-a", agent)
	}

	cmd := exec.CommandContext(ctx, "npx", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("installing skill failed: %w", err)
	}

	return &skillInstallResponse{
		Agents: uniqueAgents,
		Source: "embedded moodle-cli skill",
	}, nil
}

func copyEmbeddedSkill(targetDir string) error {
	return fs.WalkDir(skills.FS, skills.RootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(skills.RootDir, path)
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			return nil
		}
		dest := filepath.Join(targetDir, rel)
		if d.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		data, readErr := skills.FS.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		return os.WriteFile(dest, data, 0o644)
	})
}

func normalizeAgents(agents []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(agents))
	for _, agent := range agents {
		clean := strings.TrimSpace(strings.ToLower(agent))
		if clean == "" {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	sort.Strings(out)
	return out
}

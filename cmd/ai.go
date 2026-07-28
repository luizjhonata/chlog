package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/luizjhonata/chlog/internal"
	"github.com/spf13/cobra"
)

const (
	aiBlockStart = "<!-- chlog:start -->"
	aiBlockEnd   = "<!-- chlog:end -->"
)

const aiRulesTemplate = "%s\n" +
	"## Changelog (chlog) — MANDATORY\n" +
	"\n" +
	"If the repository you are working in uses chlog (a `.chlog.yaml` or `.chlog.yml`\n" +
	"config file, or a `.changes/` directory, exists at the project root), the\n" +
	"following is binding and ALWAYS applies: whenever you make ANY change, you MUST\n" +
	"create a changelog fragment as part of the same change — automatically, without\n" +
	"being asked, before committing.\n" +
	"\n" +
	"- Do NOT edit CHANGELOG.md directly; it is generated from fragments.\n" +
	"- Create the fragment with:\n" +
	"  `chlog new --kind <Kind> --body \"<imperative description>\"`\n" +
	"- Valid kinds: %s\n" +
	"- Choose the kind that best matches the change (e.g., new feature → Added,\n" +
	"  bug fix → Fixed, behavior change → Changed, removal → Removed, security fix → Security).\n" +
	"- If the change is backward-INCOMPATIBLE with the public API (a breaking\n" +
	"  change), you MUST add the `--breaking` flag:\n" +
	"  `chlog new --kind <Kind> --breaking --body \"<description>\"`.\n" +
	"  This is the ONLY thing that triggers a major version bump — the kind alone\n" +
	"  never does (per SemVer, major = incompatible change). When unsure whether a\n" +
	"  change breaks compatibility, ask the user instead of guessing.\n" +
	"- Fragments are YAML files in `.changes/unreleased/`; stage them with your commit.\n" +
	"- `chlog check` fails the build when a fragment is missing — never skip it.\n" +
	"%s\n"

type assistant struct {
	name        string
	targetFile  string
	isInstalled func() bool
}

func supportedAssistants() []assistant {
	return []assistant{
		{name: "Claude", targetFile: "CLAUDE.md", isInstalled: detectClaude},
		{name: "Codex", targetFile: "AGENTS.md", isInstalled: detectCodex},
		{name: "Cursor", targetFile: "AGENTS.md", isInstalled: detectCursor},
		{name: "Gemini", targetFile: "GEMINI.md", isInstalled: detectGemini},
		{
			name:        "Copilot",
			targetFile:  filepath.Join(".github", "copilot-instructions.md"),
			isInstalled: detectCopilot,
		},
		{
			name:        "Windsurf",
			targetFile:  filepath.Join(".windsurf", "rules", "chlog.md"),
			isInstalled: detectWindsurf,
		},
	}
}

func newAiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai",
		Short: "Manage AI assistant integration",
	}

	cmd.AddCommand(newAiSetupCmd())

	return cmd
}

func newAiSetupCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Inject chlog rules into detected AI assistant instruction files",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAiSetup(cmd, force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "re-inject the chlog block if already present")

	return cmd
}

func runAiSetup(cmd *cobra.Command, force bool) error {
	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	files := detectTargetFiles()
	if len(files) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "no supported AI assistant detected; nothing to do")

		return nil
	}

	block := aiRulesBlock(joinKinds(cfg))

	for _, file := range files {
		action, injectErr := injectAIBlock(file, block, force)
		if injectErr != nil {
			return injectErr
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", action, file)
	}

	return nil
}

func detectTargetFiles() []string {
	var files []string

	seen := make(map[string]bool)

	for _, a := range supportedAssistants() {
		if !a.isInstalled() || seen[a.targetFile] {
			continue
		}

		seen[a.targetFile] = true
		files = append(files, a.targetFile)
	}

	return files
}

func injectAIBlock(file, block string, force bool) (string, error) {
	existing, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return createAIFile(file, block)
		}

		return "", fmt.Errorf("reading %s: %w", file, err)
	}

	content := string(existing)
	hasBlock := strings.Contains(content, aiBlockStart)

	if hasBlock && !force {
		return "already configured:", nil
	}

	if hasBlock {
		content = removeMarkedBlock(content, aiBlockStart, aiBlockEnd)
	}

	err = writeAIFile(file, appendBlock(content, block))
	if err != nil {
		return "", err
	}

	return "updated", nil
}

func createAIFile(file, block string) (string, error) {
	dir := filepath.Dir(file)
	if dir != "." {
		err := os.MkdirAll(dir, 0o750)
		if err != nil {
			return "", fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	err := writeAIFile(file, block)
	if err != nil {
		return "", err
	}

	return "created", nil
}

func writeAIFile(file, content string) error {
	//nolint:gosec // G703 - file path comes from a fixed assistant table, not user input
	err := os.WriteFile(file, []byte(content), 0o600)
	if err != nil {
		return fmt.Errorf("writing %s: %w", file, err)
	}

	return nil
}

func appendBlock(content, block string) string {
	trimmed := strings.TrimRight(content, "\n")
	if trimmed == "" {
		return block
	}

	return trimmed + "\n\n" + block
}

func aiRulesBlock(kinds string) string {
	return fmt.Sprintf(aiRulesTemplate, aiBlockStart, kinds, aiBlockEnd)
}

func joinKinds(cfg *internal.Config) string {
	labels := make([]string, 0, len(cfg.Kinds))
	for _, kind := range cfg.Kinds {
		labels = append(labels, kind.Label)
	}

	return strings.Join(labels, ", ")
}

func detectClaude() bool {
	return binaryExists("claude") || homeDirExists(".claude")
}

func detectCodex() bool {
	return binaryExists("codex") || homeDirExists(".codex")
}

func detectGemini() bool {
	return binaryExists("gemini") || homeDirExists(".gemini")
}

func detectCopilot() bool {
	return configDirExists("github-copilot")
}

func detectCursor() bool {
	return binaryExists("cursor") || homeDirExists(".cursor") || configDirExists("Cursor")
}

func detectWindsurf() bool {
	return binaryExists("windsurf") ||
		homeDirExists(".windsurf") ||
		homeDirExists(filepath.Join(".codeium", "windsurf")) ||
		configDirExists("Windsurf")
}

func binaryExists(name string) bool {
	_, err := exec.LookPath(name)

	return err == nil
}

func homeDirExists(rel string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	return dirExists(filepath.Join(home, rel))
}

func configDirExists(name string) bool {
	dir, err := os.UserConfigDir()
	if err != nil {
		return false
	}

	return dirExists(filepath.Join(dir, name))
}

func dirExists(path string) bool {
	info, err := os.Stat(path)

	return err == nil && info.IsDir()
}

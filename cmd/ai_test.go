//go:build unit

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	claudeFile  = "CLAUDE.md"
	agentsFile  = "AGENTS.md"
	geminiFile  = "GEMINI.md"
	copilotFile = ".github/copilot-instructions.md"
	windsurfRel = ".windsurf/rules/chlog.md"
)

func setupAIEnv(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	binDir := filepath.Join(home, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o750))

	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("PATH", binDir)

	return home
}

func mkHomeDir(t *testing.T, home, name string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(home, name), 0o750))
}

func mkConfigDir(t *testing.T, home, name string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".config", name), 0o750))
}

func writeFakeBinary(t *testing.T, home, name string) {
	t.Helper()
	path := filepath.Join(home, "bin", name)
	//nolint:gosec // G306 - test binary must be executable
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755))
}

func executeAISetup(args ...string) (*bytes.Buffer, error) {
	var out bytes.Buffer

	cmd := newRootCmd("test")
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(append([]string{"ai", "setup"}, args...))

	return &out, cmd.Execute()
}

func TestAISetupCmd(t *testing.T) {
	t.Run("should create CLAUDE.md when claude is installed", func(t *testing.T) {
		// given
		home := setupAIEnv(t)
		mkHomeDir(t, home, ".claude")
		repo := t.TempDir()
		chdir(t, repo)

		// when
		out, err := executeAISetup()

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), claudeFile)

		data, readErr := os.ReadFile(filepath.Join(repo, claudeFile))
		require.NoError(t, readErr)

		content := string(data)
		assert.Contains(t, content, aiBlockStart)
		assert.Contains(t, content, aiBlockEnd)
		assert.Contains(t, content, "MANDATORY")
		assert.Contains(t, content, ".chlog.yaml")
		assert.Contains(t, content, ".chlog.yml")
		assert.Contains(t, content, ".changes/")
		assert.Contains(t, content, "chlog new --kind")
		assert.Contains(t, content, "best matches the change")
		assert.Contains(t, content, "chlog check")
		assert.Contains(t, content, "Added")
	})

	t.Run("should do nothing when no assistant is installed", func(t *testing.T) {
		// given
		setupAIEnv(t)
		repo := t.TempDir()
		chdir(t, repo)

		// when
		out, err := executeAISetup()

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "no supported AI assistant detected")

		entries, readErr := os.ReadDir(repo)
		require.NoError(t, readErr)
		assert.Empty(t, entries)
	})

	t.Run("should append block preserving existing content when file exists", func(t *testing.T) {
		// given
		home := setupAIEnv(t)
		mkHomeDir(t, home, ".claude")
		repo := t.TempDir()
		chdir(t, repo)

		original := "# Project rules\n\nfollow the style guide\n"
		require.NoError(t, os.WriteFile(filepath.Join(repo, claudeFile), []byte(original), 0o600))

		// when
		_, err := executeAISetup()

		// then
		require.NoError(t, err)

		data, readErr := os.ReadFile(filepath.Join(repo, claudeFile))
		require.NoError(t, readErr)

		content := string(data)
		assert.Contains(t, content, "follow the style guide")
		assert.Contains(t, content, aiBlockStart)
	})

	t.Run("should report already configured when block exists without force", func(t *testing.T) {
		// given
		home := setupAIEnv(t)
		mkHomeDir(t, home, ".claude")
		repo := t.TempDir()
		chdir(t, repo)

		require.NoError(t, os.WriteFile(
			filepath.Join(repo, claudeFile),
			[]byte(aiRulesBlock("Added")),
			0o600,
		))

		// when
		out, err := executeAISetup()

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "already configured")

		data, readErr := os.ReadFile(filepath.Join(repo, claudeFile))
		require.NoError(t, readErr)
		assert.Equal(t, 1, strings.Count(string(data), aiBlockStart))
	})

	t.Run("should re-inject single block when force is set", func(t *testing.T) {
		// given
		home := setupAIEnv(t)
		mkHomeDir(t, home, ".claude")
		repo := t.TempDir()
		chdir(t, repo)

		stale := aiBlockStart + "\nold stale rules\n" + aiBlockEnd + "\n"
		require.NoError(t, os.WriteFile(filepath.Join(repo, claudeFile), []byte(stale), 0o600))

		// when
		out, err := executeAISetup("--force")

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "updated")

		data, readErr := os.ReadFile(filepath.Join(repo, claudeFile))
		require.NoError(t, readErr)

		content := string(data)
		assert.Equal(t, 1, strings.Count(content, aiBlockStart))
		assert.NotContains(t, content, "old stale rules")
		assert.Contains(t, content, "chlog new --kind")
	})

	t.Run("should write AGENTS.md once when codex and cursor are both installed", func(t *testing.T) {
		// given
		home := setupAIEnv(t)
		mkHomeDir(t, home, ".codex")
		mkHomeDir(t, home, ".cursor")
		repo := t.TempDir()
		chdir(t, repo)

		// when
		_, err := executeAISetup()

		// then
		require.NoError(t, err)

		data, readErr := os.ReadFile(filepath.Join(repo, agentsFile))
		require.NoError(t, readErr)
		assert.Equal(t, 1, strings.Count(string(data), aiBlockStart))
	})

	t.Run("should write copilot instructions when copilot config dir exists", func(t *testing.T) {
		// given
		home := setupAIEnv(t)
		mkConfigDir(t, home, "github-copilot")
		repo := t.TempDir()
		chdir(t, repo)

		// when
		_, err := executeAISetup()

		// then
		require.NoError(t, err)

		data, readErr := os.ReadFile(filepath.Join(repo, filepath.FromSlash(copilotFile)))
		require.NoError(t, readErr)
		assert.Contains(t, string(data), aiBlockStart)
	})

	t.Run("should write windsurf rules when windsurf dir exists", func(t *testing.T) {
		// given
		home := setupAIEnv(t)
		mkHomeDir(t, home, ".windsurf")
		repo := t.TempDir()
		chdir(t, repo)

		// when
		_, err := executeAISetup()

		// then
		require.NoError(t, err)

		data, readErr := os.ReadFile(filepath.Join(repo, filepath.FromSlash(windsurfRel)))
		require.NoError(t, readErr)
		assert.Contains(t, string(data), aiBlockStart)
	})

	t.Run("should detect assistant via binary in PATH", func(t *testing.T) {
		// given
		home := setupAIEnv(t)
		writeFakeBinary(t, home, "gemini")
		repo := t.TempDir()
		chdir(t, repo)

		// when
		_, err := executeAISetup()

		// then
		require.NoError(t, err)

		data, readErr := os.ReadFile(filepath.Join(repo, geminiFile))
		require.NoError(t, readErr)
		assert.Contains(t, string(data), aiBlockStart)
	})

	t.Run("should reflect configured kinds in the block", func(t *testing.T) {
		// given
		home := setupAIEnv(t)
		mkHomeDir(t, home, ".claude")
		repo := t.TempDir()
		chdir(t, repo)

		writeConfig(t, repo, "kinds:\n  - label: Spice\n  - label: Sauce\n")

		// when
		_, err := executeAISetup()

		// then
		require.NoError(t, err)

		data, readErr := os.ReadFile(filepath.Join(repo, claudeFile))
		require.NoError(t, readErr)

		content := string(data)
		assert.Contains(t, content, "Spice")
		assert.Contains(t, content, "Sauce")
	})

	t.Run("should write block to AGENTS.md when CLAUDE.md only imports it", func(t *testing.T) {
		// given
		home := setupAIEnv(t)
		mkHomeDir(t, home, ".claude")
		repo := t.TempDir()
		chdir(t, repo)

		shim := "@AGENTS.md\n"
		require.NoError(t, os.WriteFile(filepath.Join(repo, claudeFile), []byte(shim), 0o600))

		// when
		_, err := executeAISetup()

		// then
		require.NoError(t, err)

		data, readErr := os.ReadFile(filepath.Join(repo, agentsFile))
		require.NoError(t, readErr)
		assert.Equal(t, 1, strings.Count(string(data), aiBlockStart))

		shimData, shimErr := os.ReadFile(filepath.Join(repo, claudeFile))
		require.NoError(t, shimErr)
		assert.Equal(t, shim, string(shimData))
	})

	for name, args := range map[string][]string{"without force": nil, "with force": {"--force"}} {
		t.Run("should move a legacy shim block to AGENTS.md "+name, func(t *testing.T) {
			// given
			home := setupAIEnv(t)
			mkHomeDir(t, home, ".claude")
			repo := t.TempDir()
			chdir(t, repo)

			shim := "@AGENTS.md\n"
			legacy := appendBlock(shim, aiRulesBlock("Added"))
			require.NoError(t, os.WriteFile(filepath.Join(repo, claudeFile), []byte(legacy), 0o600))

			// when
			out, err := executeAISetup(args...)

			// then
			require.NoError(t, err)
			assert.Contains(t, out.String(), "removed duplicate block from "+claudeFile)

			data, readErr := os.ReadFile(filepath.Join(repo, agentsFile))
			require.NoError(t, readErr)
			assert.Equal(t, 1, strings.Count(string(data), aiBlockStart))

			shimData, shimErr := os.ReadFile(filepath.Join(repo, claudeFile))
			require.NoError(t, shimErr)
			assert.Equal(t, shim, string(shimData))
		})
	}

	t.Run("should keep the CLAUDE.md block when CLAUDE.md has no AGENTS.md import", func(t *testing.T) {
		// given
		home := setupAIEnv(t)
		mkHomeDir(t, home, ".claude")
		mkHomeDir(t, home, ".codex")
		repo := t.TempDir()
		chdir(t, repo)

		full := appendBlock("# Full rules\n", aiRulesBlock("Added"))
		require.NoError(t, os.WriteFile(filepath.Join(repo, claudeFile), []byte(full), 0o600))

		// when
		_, err := executeAISetup()

		// then
		require.NoError(t, err)

		claudeData, claudeErr := os.ReadFile(filepath.Join(repo, claudeFile))
		require.NoError(t, claudeErr)
		assert.Equal(t, full, string(claudeData))
	})

	t.Run("should write one block to AGENTS.md when shim and codex are both detected", func(t *testing.T) {
		// given
		home := setupAIEnv(t)
		mkHomeDir(t, home, ".claude")
		mkHomeDir(t, home, ".codex")
		repo := t.TempDir()
		chdir(t, repo)

		shim := "  @AGENTS.md  \n"
		require.NoError(t, os.WriteFile(filepath.Join(repo, claudeFile), []byte(shim), 0o600))

		// when
		_, err := executeAISetup()

		// then
		require.NoError(t, err)

		data, readErr := os.ReadFile(filepath.Join(repo, agentsFile))
		require.NoError(t, readErr)
		assert.Equal(t, 1, strings.Count(string(data), aiBlockStart))

		shimData, shimErr := os.ReadFile(filepath.Join(repo, claudeFile))
		require.NoError(t, shimErr)
		assert.Equal(t, shim, string(shimData))
	})

	t.Run("should write block to AGENTS.md when only AGENTS.md exists", func(t *testing.T) {
		// given
		home := setupAIEnv(t)
		mkHomeDir(t, home, ".claude")
		repo := t.TempDir()
		chdir(t, repo)

		require.NoError(t, os.WriteFile(filepath.Join(repo, agentsFile), []byte("# Rules\n"), 0o600))

		// when
		_, err := executeAISetup()

		// then
		require.NoError(t, err)

		data, readErr := os.ReadFile(filepath.Join(repo, agentsFile))
		require.NoError(t, readErr)
		assert.Contains(t, string(data), aiBlockStart)

		_, statErr := os.Stat(filepath.Join(repo, claudeFile))
		assert.True(t, os.IsNotExist(statErr))
	})

	t.Run("should write block to CLAUDE.md when it has no AGENTS.md import", func(t *testing.T) {
		// given
		home := setupAIEnv(t)
		mkHomeDir(t, home, ".claude")
		repo := t.TempDir()
		chdir(t, repo)

		require.NoError(t, os.WriteFile(filepath.Join(repo, claudeFile), []byte("# Full rules\n"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(repo, agentsFile), []byte("# Other\n"), 0o600))

		// when
		_, err := executeAISetup()

		// then
		require.NoError(t, err)

		data, readErr := os.ReadFile(filepath.Join(repo, claudeFile))
		require.NoError(t, readErr)
		assert.Contains(t, string(data), aiBlockStart)

		agentsData, agentsErr := os.ReadFile(filepath.Join(repo, agentsFile))
		require.NoError(t, agentsErr)
		assert.NotContains(t, string(agentsData), aiBlockStart)
	})

	t.Run("should create CLAUDE.md when neither CLAUDE.md nor AGENTS.md exists", func(t *testing.T) {
		// given
		home := setupAIEnv(t)
		mkHomeDir(t, home, ".claude")
		repo := t.TempDir()
		chdir(t, repo)

		// when
		_, err := executeAISetup()

		// then
		require.NoError(t, err)
		assert.FileExists(t, filepath.Join(repo, claudeFile))

		_, statErr := os.Stat(filepath.Join(repo, agentsFile))
		assert.True(t, os.IsNotExist(statErr))
	})
}

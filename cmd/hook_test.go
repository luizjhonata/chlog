//go:build unit

package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const chlogManagedMarker = "# chlog:managed"

func setupGlobalEnv(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(tmpDir, ".gitconfig"))
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	return tmpDir
}

func initGitDir(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0o755))
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()

	cmd := exec.Command("git", "init", dir)
	cmd.Env = os.Environ()
	require.NoError(t, cmd.Run())
}

func executeHookInstall(args ...string) (*bytes.Buffer, error) {
	var out bytes.Buffer

	cmd := newRootCmd("test")
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(append([]string{"hook", "install"}, args...))

	return &out, cmd.Execute()
}

func executeHookUninstall(args ...string) (*bytes.Buffer, error) {
	var out bytes.Buffer

	cmd := newRootCmd("test")
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(append([]string{"hook", "uninstall"}, args...))

	return &out, cmd.Execute()
}

func TestHookGlobalInstallCmd(t *testing.T) {
	t.Run("should install global hook with correct script and symlinks", func(t *testing.T) {
		// given
		tmpDir := setupGlobalEnv(t)
		hooksDir := filepath.Join(tmpDir, ".config", "chlog", "hooks")

		// when
		out, err := executeHookInstall()

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "chlog hook installed globally")

		hookPath := filepath.Join(hooksDir, "pre-commit")
		data, readErr := os.ReadFile(hookPath)
		require.NoError(t, readErr)
		assert.Contains(t, string(data), "#!/bin/sh")
		assert.Contains(t, string(data), chlogManagedMarker)
		assert.Contains(t, string(data), "chlog check")

		info, statErr := os.Stat(hookPath)
		require.NoError(t, statErr)
		assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())

		for _, name := range passthroughHookNames() {
			linkPath := filepath.Join(hooksDir, name)
			target, linkErr := os.Readlink(linkPath)
			require.NoError(t, linkErr, "symlink %s should exist", name)
			assert.Equal(t, "pre-commit", target)
		}

		configuredPath := getGlobalHooksPath()
		assert.Equal(t, hooksDir, configuredPath)
	})

	t.Run("should report already installed when global hook exists", func(t *testing.T) {
		// given
		tmpDir := setupGlobalEnv(t)
		hooksDir := filepath.Join(tmpDir, ".config", "chlog", "hooks")
		require.NoError(t, os.MkdirAll(hooksDir, 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(hooksDir, "pre-commit"),
			[]byte(globalHookScript),
			0o755,
		))
		require.NoError(t, exec.Command("git", "config", "--global", "core.hooksPath", hooksDir).Run())

		// when
		out, err := executeHookInstall()

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "chlog hook is already installed")
	})

	t.Run("should fail when core.hooksPath is set to another directory", func(t *testing.T) {
		// given
		setupGlobalEnv(t)
		require.NoError(t, exec.Command("git", "config", "--global", "core.hooksPath", "/some/other/dir").Run())

		// when
		_, err := executeHookInstall()

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "core.hooksPath is already configured")
		assert.Contains(t, err.Error(), "--force")
	})

	t.Run("should override existing core.hooksPath when force flag is set", func(t *testing.T) {
		// given
		tmpDir := setupGlobalEnv(t)
		hooksDir := filepath.Join(tmpDir, ".config", "chlog", "hooks")
		require.NoError(t, exec.Command("git", "config", "--global", "core.hooksPath", "/some/other/dir").Run())

		// when
		out, err := executeHookInstall("--force")

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "chlog hook installed globally")

		configuredPath := getGlobalHooksPath()
		assert.Equal(t, hooksDir, configuredPath)
	})
}

func TestHookGlobalUninstallCmd(t *testing.T) {
	t.Run("should remove global hook and unset core.hooksPath", func(t *testing.T) {
		// given
		tmpDir := setupGlobalEnv(t)
		hooksDir := filepath.Join(tmpDir, ".config", "chlog", "hooks")
		require.NoError(t, os.MkdirAll(hooksDir, 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(hooksDir, "pre-commit"),
			[]byte(globalHookScript),
			0o755,
		))
		require.NoError(t, exec.Command("git", "config", "--global", "core.hooksPath", hooksDir).Run())

		// when
		out, err := executeHookUninstall()

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "chlog hook removed")

		_, statErr := os.Stat(hooksDir)
		assert.True(t, os.IsNotExist(statErr))

		assert.Empty(t, getGlobalHooksPath())
	})

	t.Run("should report not installed when core.hooksPath is not set", func(t *testing.T) {
		// given
		setupGlobalEnv(t)

		// when
		out, err := executeHookUninstall()

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "chlog hook is not installed")
	})

	t.Run("should fail when core.hooksPath is not managed by chlog", func(t *testing.T) {
		// given
		setupGlobalEnv(t)
		require.NoError(t, exec.Command("git", "config", "--global", "core.hooksPath", "/some/other/dir").Run())

		// when
		_, err := executeHookUninstall()

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not managed by chlog")
	})
}

func TestHookLocalInstallCmd(t *testing.T) {
	t.Run("should inject chlog block into existing pre-commit hook", func(t *testing.T) {
		// given
		setupGlobalEnv(t)
		dir := t.TempDir()
		initGitDir(t, dir)
		chdir(t, dir)

		hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
		originalContent := "#!/bin/sh\nnpx lint-staged\n"
		require.NoError(t, os.WriteFile(hookPath, []byte(originalContent), 0o755))

		// when
		out, err := executeHookInstall("--local")

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "chlog hook installed locally")

		data, readErr := os.ReadFile(hookPath)
		require.NoError(t, readErr)

		content := string(data)
		assert.Contains(t, content, "npx lint-staged")
		assert.Contains(t, content, localBlockStart)
		assert.Contains(t, content, "chlog check")
		assert.Contains(t, content, localBlockEnd)
	})

	t.Run("should create pre-commit hook when it does not exist", func(t *testing.T) {
		// given
		setupGlobalEnv(t)
		dir := t.TempDir()
		initGitDir(t, dir)
		chdir(t, dir)

		// when
		out, err := executeHookInstall("--local")

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "chlog hook installed locally")

		hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
		data, readErr := os.ReadFile(hookPath)
		require.NoError(t, readErr)

		content := string(data)
		assert.Contains(t, content, "#!/bin/sh")
		assert.Contains(t, content, localBlockStart)
		assert.Contains(t, content, "chlog check")

		info, statErr := os.Stat(hookPath)
		require.NoError(t, statErr)
		assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())
	})

	t.Run("should report already installed when chlog block exists", func(t *testing.T) {
		// given
		setupGlobalEnv(t)
		dir := t.TempDir()
		initGitDir(t, dir)
		chdir(t, dir)

		hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
		hookContent := "#!/bin/sh\n" + localHookBlock
		require.NoError(t, os.WriteFile(hookPath, []byte(hookContent), 0o755))

		// when
		out, err := executeHookInstall("--local")

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "chlog hook is already installed")
	})

	t.Run("should detect hooks directory from local core.hooksPath", func(t *testing.T) {
		// given
		setupGlobalEnv(t)
		dir := t.TempDir()
		initGitRepo(t, dir)
		chdir(t, dir)

		huskyDir := filepath.Join(dir, ".husky")
		require.NoError(t, os.MkdirAll(huskyDir, 0o755))

		huskyHook := filepath.Join(huskyDir, "pre-commit")
		originalContent := "#!/bin/sh\nnpx lint-staged\n"
		require.NoError(t, os.WriteFile(huskyHook, []byte(originalContent), 0o755))

		require.NoError(t, exec.Command("git", "-C", dir, "config", "--local", "core.hooksPath", ".husky").Run())

		// when
		out, err := executeHookInstall("--local")

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "chlog hook installed locally")
		assert.Contains(t, out.String(), ".husky")

		data, readErr := os.ReadFile(huskyHook)
		require.NoError(t, readErr)

		content := string(data)
		assert.Contains(t, content, "npx lint-staged")
		assert.Contains(t, content, localBlockStart)
	})

	t.Run("should fail when not inside a git repository", func(t *testing.T) {
		// given
		setupGlobalEnv(t)
		dir := t.TempDir()
		chdir(t, dir)

		// when
		_, err := executeHookInstall("--local")

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), ".git directory not found")
	})
}

func TestHookLocalUninstallCmd(t *testing.T) {
	t.Run("should remove chlog block and keep existing hook content", func(t *testing.T) {
		// given
		setupGlobalEnv(t)
		dir := t.TempDir()
		initGitDir(t, dir)
		chdir(t, dir)

		hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
		hookContent := "#!/bin/sh\nnpx lint-staged\n\n" + localHookBlock
		require.NoError(t, os.WriteFile(hookPath, []byte(hookContent), 0o755))

		// when
		out, err := executeHookUninstall("--local")

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "chlog hook removed")

		data, readErr := os.ReadFile(hookPath)
		require.NoError(t, readErr)

		content := string(data)
		assert.Contains(t, content, "npx lint-staged")
		assert.NotContains(t, content, localBlockStart)
		assert.NotContains(t, content, localBlockEnd)
	})

	t.Run("should delete hook file when only chlog block remains", func(t *testing.T) {
		// given
		setupGlobalEnv(t)
		dir := t.TempDir()
		initGitDir(t, dir)
		chdir(t, dir)

		hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
		hookContent := "#!/bin/sh\n" + localHookBlock
		require.NoError(t, os.WriteFile(hookPath, []byte(hookContent), 0o755))

		// when
		out, err := executeHookUninstall("--local")

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "chlog hook removed")

		_, statErr := os.Stat(hookPath)
		assert.True(t, os.IsNotExist(statErr))
	})

	t.Run("should report no hook when pre-commit does not exist", func(t *testing.T) {
		// given
		setupGlobalEnv(t)
		dir := t.TempDir()
		initGitDir(t, dir)
		chdir(t, dir)

		// when
		out, err := executeHookUninstall("--local")

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "no chlog hook installed")
	})

	t.Run("should fail when chlog block is not found in hook", func(t *testing.T) {
		// given
		setupGlobalEnv(t)
		dir := t.TempDir()
		initGitDir(t, dir)
		chdir(t, dir)

		hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
		require.NoError(t, os.WriteFile(hookPath, []byte("#!/bin/sh\necho custom\n"), 0o755))

		// when
		_, err := executeHookUninstall("--local")

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, errHookBlockNotFound)
	})
}

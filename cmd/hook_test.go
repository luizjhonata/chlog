//go:build unit

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const chlogManagedMarker = "# chlog:managed"

func initGitDir(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0o755))
}

func executeHookInstall(args ...string) (*bytes.Buffer, error) {
	var out bytes.Buffer

	cmd := newRootCmd("test")
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(append([]string{"hook", "install"}, args...))

	return &out, cmd.Execute()
}

func executeHookUninstall() (*bytes.Buffer, error) {
	var out bytes.Buffer

	cmd := newRootCmd("test")
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"hook", "uninstall"})

	return &out, cmd.Execute()
}

func TestHookInstallCmd(t *testing.T) {
	t.Run("should create pre-commit hook with chlog check", func(t *testing.T) {
		// given
		dir := t.TempDir()
		initGitDir(t, dir)
		chdir(t, dir)

		// when
		out, err := executeHookInstall()

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "pre-commit hook installed")

		hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
		data, readErr := os.ReadFile(hookPath)
		require.NoError(t, readErr)
		assert.Contains(t, string(data), "#!/bin/sh")
		assert.Contains(t, string(data), chlogManagedMarker)
		assert.Contains(t, string(data), "chlog check")

		info, statErr := os.Stat(hookPath)
		require.NoError(t, statErr)
		assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())
	})

	t.Run("should create hooks directory when it does not exist", func(t *testing.T) {
		// given
		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))
		chdir(t, dir)

		// when
		_, err := executeHookInstall()

		// then
		require.NoError(t, err)

		hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
		_, statErr := os.Stat(hookPath)
		assert.NoError(t, statErr)
	})

	t.Run("should report already installed when chlog hook exists", func(t *testing.T) {
		// given
		dir := t.TempDir()
		initGitDir(t, dir)
		chdir(t, dir)

		hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
		hookContent := "#!/bin/sh\n# chlog:managed\nchlog check\n"
		require.NoError(t, os.WriteFile(hookPath, []byte(hookContent), 0o755))

		// when
		out, err := executeHookInstall()

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "pre-commit hook already installed")
	})

	t.Run("should fail when non-chlog hook exists without force flag", func(t *testing.T) {
		// given
		dir := t.TempDir()
		initGitDir(t, dir)
		chdir(t, dir)

		hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
		require.NoError(t, os.WriteFile(hookPath, []byte("#!/bin/sh\necho custom hook\n"), 0o755))

		// when
		_, err := executeHookInstall()

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, errHookExists)
	})

	t.Run("should overwrite non-chlog hook when force flag is set", func(t *testing.T) {
		// given
		dir := t.TempDir()
		initGitDir(t, dir)
		chdir(t, dir)

		hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
		require.NoError(t, os.WriteFile(hookPath, []byte("#!/bin/sh\necho custom hook\n"), 0o755))

		// when
		out, err := executeHookInstall("--force")

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "pre-commit hook installed")

		data, readErr := os.ReadFile(hookPath)
		require.NoError(t, readErr)
		assert.Contains(t, string(data), chlogManagedMarker)
	})

	t.Run("should fail when not inside a git repository", func(t *testing.T) {
		// given
		dir := t.TempDir()
		chdir(t, dir)

		// when
		_, err := executeHookInstall()

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), ".git directory not found")
	})
}

func TestHookUninstallCmd(t *testing.T) {
	t.Run("should remove chlog-managed hook", func(t *testing.T) {
		// given
		dir := t.TempDir()
		initGitDir(t, dir)
		chdir(t, dir)

		hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
		hookContent := "#!/bin/sh\n# chlog:managed\nchlog check\n"
		require.NoError(t, os.WriteFile(hookPath, []byte(hookContent), 0o755))

		// when
		out, err := executeHookUninstall()

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "pre-commit hook removed")

		_, statErr := os.Stat(hookPath)
		assert.True(t, os.IsNotExist(statErr))
	})

	t.Run("should report no hook installed when hook file does not exist", func(t *testing.T) {
		// given
		dir := t.TempDir()
		initGitDir(t, dir)
		chdir(t, dir)

		// when
		out, err := executeHookUninstall()

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "no pre-commit hook installed")
	})

	t.Run("should fail when hook is not managed by chlog", func(t *testing.T) {
		// given
		dir := t.TempDir()
		initGitDir(t, dir)
		chdir(t, dir)

		hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
		require.NoError(t, os.WriteFile(hookPath, []byte("#!/bin/sh\necho custom\n"), 0o755))

		// when
		_, err := executeHookUninstall()

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, errHookNotOwned)
	})

	t.Run("should fail when not inside a git repository", func(t *testing.T) {
		// given
		dir := t.TempDir()
		chdir(t, dir)

		// when
		_, err := executeHookUninstall()

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), ".git directory not found")
	})
}

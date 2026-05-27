//go:build unit

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeConfig(t *testing.T, dir, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".chlog.yaml"), []byte(content), 0o644))
}

func chdir(t *testing.T, dir string) {
	t.Helper()

	original, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { os.Chdir(original) })
}

func executeNew(args ...string) (*bytes.Buffer, error) {
	var out bytes.Buffer

	cmd := newRootCmd("test")
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(append([]string{"new"}, args...))

	return &out, cmd.Execute()
}

func TestNewCmd(t *testing.T) {
	t.Run("should create fragment file with valid kind and body", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, "kinds:\n  - label: Added\n    auto: minor\n  - label: Fixed\n    auto: patch\n")
		chdir(t, dir)

		// when
		_, err := executeNew("--kind", "Added", "--body", "add new feature")

		// then
		require.NoError(t, err)

		entries, readErr := os.ReadDir(filepath.Join(dir, ".changes", "unreleased"))
		require.NoError(t, readErr)
		require.Len(t, entries, 1)

		data, readErr := os.ReadFile(filepath.Join(dir, ".changes", "unreleased", entries[0].Name()))
		require.NoError(t, readErr)
		assert.Contains(t, string(data), "kind: Added")
		assert.Contains(t, string(data), "body: add new feature")
	})

	t.Run("should reject unknown kind", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, "kinds:\n  - label: Added\n")
		chdir(t, dir)

		// when
		_, err := executeNew("--kind", "Unknown", "--body", "something")

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown kind "Unknown"`)
	})

	t.Run("should match kind case-insensitively and normalize label", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, "kinds:\n  - label: Fixed\n    auto: patch\n")
		chdir(t, dir)

		// when
		_, err := executeNew("--kind", "fixed", "--body", "fix something")

		// then
		require.NoError(t, err)

		entries, readErr := os.ReadDir(filepath.Join(dir, ".changes", "unreleased"))
		require.NoError(t, readErr)
		require.Len(t, entries, 1)

		data, readErr := os.ReadFile(filepath.Join(dir, ".changes", "unreleased", entries[0].Name()))
		require.NoError(t, readErr)
		assert.Contains(t, string(data), "kind: Fixed")
	})

	t.Run("should create directories when they do not exist", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, "changesDir: custom-changes\nunreleasedDir: pending\nkinds:\n  - label: Added\n")
		chdir(t, dir)

		// when
		_, err := executeNew("--kind", "Added", "--body", "add feature")

		// then
		require.NoError(t, err)

		entries, readErr := os.ReadDir(filepath.Join(dir, "custom-changes", "pending"))
		require.NoError(t, readErr)
		assert.Len(t, entries, 1)
	})

	t.Run("should generate unique filenames on consecutive calls", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, "kinds:\n  - label: Added\n")
		chdir(t, dir)

		// when
		_, err := executeNew("--kind", "Added", "--body", "first change")
		require.NoError(t, err)

		_, err = executeNew("--kind", "Added", "--body", "second change")
		require.NoError(t, err)

		// then
		entries, readErr := os.ReadDir(filepath.Join(dir, ".changes", "unreleased"))
		require.NoError(t, readErr)
		require.Len(t, entries, 2)
		assert.NotEqual(t, entries[0].Name(), entries[1].Name())
	})

	t.Run("should use expected filename format", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, "kinds:\n  - label: Added\n")
		chdir(t, dir)

		// when
		_, err := executeNew("--kind", "Added", "--body", "test format")

		// then
		require.NoError(t, err)

		entries, readErr := os.ReadDir(filepath.Join(dir, ".changes", "unreleased"))
		require.NoError(t, readErr)
		require.Len(t, entries, 1)

		pattern := regexp.MustCompile(`^\d+-[0-9a-f]{4}\.yaml$`)
		assert.Regexp(t, pattern, entries[0].Name())
	})

	t.Run("should print created file path to stdout", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, "kinds:\n  - label: Added\n")
		chdir(t, dir)

		// when
		out, err := executeNew("--kind", "Added", "--body", "check output")

		// then
		require.NoError(t, err)

		output := strings.TrimSpace(out.String())
		assert.True(t, strings.HasPrefix(output, ".changes/unreleased/"))
		assert.True(t, strings.HasSuffix(output, ".yaml"))
	})
}

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

func writeChangelog(t *testing.T, dir, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte(content), 0o600))
}

func executeMerge() (*bytes.Buffer, error) {
	var out bytes.Buffer

	cmd := newRootCmd("test")
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"merge"})

	return &out, cmd.Execute()
}

func TestMergeCmd(t *testing.T) {
	defaultKinds := "kinds:\n  - label: Added\n    auto: minor\n"
	v100Content := "## [1.0.0] - 2026-05-27\n\n### Added\n\n- feature one\n"
	v200Content := "## [2.0.0] - 2026-05-28\n\n### Added\n\n- feature two\n"

	changelogPreamble := "# Changelog\n\nAll notable changes.\n\n## [Unreleased]\n"

	t.Run("should insert version content into existing changelog", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, defaultKinds)
		writeChangelog(t, dir, changelogPreamble)
		writeVersionFile(t, dir, "1.0.0", v100Content)
		chdir(t, dir)

		// when
		out, err := executeMerge()

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "merged 1 version(s) into CHANGELOG.md")

		data, readErr := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
		require.NoError(t, readErr)

		content := string(data)
		assert.Contains(t, content, "# Changelog")
		assert.Contains(t, content, "## [Unreleased]")
		assert.Contains(t, content, "## [1.0.0] - 2026-05-27")
		assert.Contains(t, content, "- feature one")
	})

	t.Run("should insert before existing version entries", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, defaultKinds)
		writeChangelog(t, dir, changelogPreamble+"\n"+v100Content)
		writeVersionFile(t, dir, "2.0.0", v200Content)
		chdir(t, dir)

		// when
		_, err := executeMerge()

		// then
		require.NoError(t, err)

		data, readErr := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
		require.NoError(t, readErr)

		content := string(data)
		v2Idx := strings.Index(content, "## [2.0.0]")
		v1Idx := strings.Index(content, "## [1.0.0]")
		assert.Greater(t, v1Idx, v2Idx, "v2.0.0 should appear before v1.0.0")
	})

	t.Run("should create changelog if it does not exist", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, defaultKinds)
		writeVersionFile(t, dir, "1.0.0", v100Content)
		chdir(t, dir)

		// when
		_, err := executeMerge()

		// then
		require.NoError(t, err)

		data, readErr := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
		require.NoError(t, readErr)

		content := string(data)
		assert.True(t, strings.HasPrefix(content, "# Changelog"))
		assert.Contains(t, content, "## [1.0.0]")
	})

	t.Run("should sort multiple versions newest first", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, defaultKinds)
		writeChangelog(t, dir, changelogPreamble)
		writeVersionFile(t, dir, "1.0.0", v100Content)
		writeVersionFile(t, dir, "2.0.0", v200Content)
		chdir(t, dir)

		// when
		out, err := executeMerge()

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "merged 2 version(s)")

		data, readErr := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
		require.NoError(t, readErr)

		content := string(data)
		v2Idx := strings.Index(content, "## [2.0.0]")
		v1Idx := strings.Index(content, "## [1.0.0]")
		assert.Greater(t, v1Idx, v2Idx, "v2.0.0 should appear before v1.0.0")
	})

	t.Run("should fail when no version files exist", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, defaultKinds)
		writeChangelog(t, dir, changelogPreamble)
		chdir(t, dir)

		// when
		_, err := executeMerge()

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no version files found")
	})

	t.Run("should delete consumed version files after merge", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, defaultKinds)
		writeChangelog(t, dir, changelogPreamble)
		writeVersionFile(t, dir, "1.0.0", v100Content)
		chdir(t, dir)

		// when
		_, err := executeMerge()

		// then
		require.NoError(t, err)

		_, statErr := os.Stat(filepath.Join(dir, ".changes", "v1.0.0.md"))
		assert.True(t, os.IsNotExist(statErr))
	})

	t.Run("should preserve unreleased section content", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, defaultKinds)
		unreleasedContent := changelogPreamble + "\n### Added\n\n- work in progress\n"
		writeChangelog(t, dir, unreleasedContent)
		writeVersionFile(t, dir, "1.0.0", v100Content)
		chdir(t, dir)

		// when
		_, err := executeMerge()

		// then
		require.NoError(t, err)

		data, readErr := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
		require.NoError(t, readErr)

		content := string(data)
		assert.Contains(t, content, "- work in progress")
		assert.Contains(t, content, "## [1.0.0]")

		wipIdx := strings.Index(content, "- work in progress")
		v1Idx := strings.Index(content, "## [1.0.0]")
		assert.Greater(t, v1Idx, wipIdx, "unreleased content should remain above version entries")
	})
}

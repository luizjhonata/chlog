//go:build unit

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/luizjhonata/chlog/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFragment(t *testing.T, dir, filename, kind, body string) {
	t.Helper()

	unreleasedDir := filepath.Join(dir, ".changes", "unreleased")
	require.NoError(t, os.MkdirAll(unreleasedDir, 0o750))

	change := &internal.Change{Kind: kind, Body: body, Time: time.Now().UTC()}

	data, err := change.Marshal()
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(unreleasedDir, filename), data, 0o600))
}

func writeVersionFile(t *testing.T, dir, version, content string) {
	t.Helper()

	changesDir := filepath.Join(dir, ".changes")
	require.NoError(t, os.MkdirAll(changesDir, 0o750))

	path := filepath.Join(changesDir, "v"+version+".md")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func executeBatch(args ...string) (*bytes.Buffer, error) {
	var out bytes.Buffer

	cmd := newRootCmd("test")
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(append([]string{"batch"}, args...))

	return &out, cmd.Execute()
}

func TestBatchCmd(t *testing.T) {
	defaultKinds := "kinds:\n  - label: Added\n    auto: minor\n  - label: Fixed\n    auto: patch\n"

	t.Run("should compile fragments into version file and delete them", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, defaultKinds)
		writeFragment(t, dir, "001.yaml", "Added", "add new feature")
		writeFragment(t, dir, "002.yaml", "Fixed", "fix a bug")
		chdir(t, dir)

		// when
		out, err := executeBatch("1.0.0")

		// then
		require.NoError(t, err)

		versionPath := strings.TrimSpace(out.String())
		assert.Equal(t, filepath.Join(".changes", "v1.0.0.md"), versionPath)

		data, readErr := os.ReadFile(filepath.Join(dir, versionPath))
		require.NoError(t, readErr)

		content := string(data)
		assert.Contains(t, content, "## [1.0.0]")
		assert.Contains(t, content, "### Added")
		assert.Contains(t, content, "- add new feature")
		assert.Contains(t, content, "### Fixed")
		assert.Contains(t, content, "- fix a bug")

		entries, readErr := os.ReadDir(filepath.Join(dir, ".changes", "unreleased"))
		require.NoError(t, readErr)
		assert.Empty(t, entries)
	})

	t.Run("should increment from latest existing version", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, defaultKinds)
		writeFragment(t, dir, "001.yaml", "Added", "add feature")
		writeVersionFile(t, dir, "1.2.0", "## [1.2.0]")
		chdir(t, dir)

		// when
		out, err := executeBatch("minor")

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "v1.3.0.md")
	})

	t.Run("should auto-bump based on highest change kind", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, "kinds:\n  - label: Added\n    auto: minor\n  - label: Changed\n    auto: major\n  - label: Fixed\n    auto: patch\n")
		writeFragment(t, dir, "001.yaml", "Added", "add feature")
		writeFragment(t, dir, "002.yaml", "Changed", "breaking change")
		chdir(t, dir)

		// when
		out, err := executeBatch("auto")

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "v1.0.0.md")
	})

	t.Run("should start from 0.0.0 when no previous versions exist", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, "kinds:\n  - label: Fixed\n    auto: patch\n")
		writeFragment(t, dir, "001.yaml", "Fixed", "fix something")
		chdir(t, dir)

		// when
		out, err := executeBatch("patch")

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "v0.0.1.md")
	})

	t.Run("should fail when no fragments exist", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, defaultKinds)
		chdir(t, dir)

		// when
		_, err := executeBatch("1.0.0")

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no unreleased fragments found")
	})

	t.Run("should reject invalid version argument", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, defaultKinds)
		writeFragment(t, dir, "001.yaml", "Added", "feature")
		chdir(t, dir)

		// when
		_, err := executeBatch("invalid")

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid version argument")
	})

	t.Run("should sort changes by kind order in output", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, defaultKinds)
		writeFragment(t, dir, "001.yaml", "Fixed", "fix bug")
		writeFragment(t, dir, "002.yaml", "Added", "add feature")
		chdir(t, dir)

		// when
		out, err := executeBatch("1.0.0")

		// then
		require.NoError(t, err)

		data, readErr := os.ReadFile(filepath.Join(dir, strings.TrimSpace(out.String())))
		require.NoError(t, readErr)

		content := string(data)
		addedIdx := strings.Index(content, "### Added")
		fixedIdx := strings.Index(content, "### Fixed")
		assert.Greater(t, fixedIdx, addedIdx, "Added should appear before Fixed in output")
	})

	t.Run("should not add blank lines between changes of the same kind", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, defaultKinds)
		writeFragment(t, dir, "001.yaml", "Added", "first feature")
		writeFragment(t, dir, "002.yaml", "Added", "second feature")
		chdir(t, dir)

		// when
		out, err := executeBatch("1.0.0")

		// then
		require.NoError(t, err)

		data, readErr := os.ReadFile(filepath.Join(dir, strings.TrimSpace(out.String())))
		require.NoError(t, readErr)

		content := string(data)
		assert.Contains(t, content, "- second feature\n- first feature\n",
			"changes of the same kind should have no blank line between them")
	})
}

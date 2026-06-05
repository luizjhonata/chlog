//go:build unit

package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	t.Run("should return valid config with all six kinds", func(t *testing.T) {
		t.Parallel()

		// when
		cfg := DefaultConfig()

		// then
		assert.Equal(t, ".changes", cfg.ChangesDir)
		assert.Equal(t, "unreleased", cfg.UnreleasedDir)
		assert.Equal(t, "CHANGELOG.md", cfg.ChangelogPath)
		assert.Len(t, cfg.Kinds, 6)
		assert.Equal(t, "Added", cfg.Kinds[0].Label)
		assert.Equal(t, "Security", cfg.Kinds[5].Label)
	})
}

func TestLoadConfigFromPath(t *testing.T) {
	t.Parallel()

	t.Run("should parse valid config file", func(t *testing.T) {
		t.Parallel()

		// given
		dir := t.TempDir()
		content := `
changesDir: .custom-changes
unreleasedDir: pending
changelogPath: CHANGES.md
kinds:
  - label: Added
    auto: minor
  - label: Fixed
    auto: patch
`
		path := filepath.Join(dir, ".chlog.yaml")
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

		// when
		cfg, err := LoadConfigFromPath(path)

		// then
		require.NoError(t, err)
		assert.Equal(t, ".custom-changes", cfg.ChangesDir)
		assert.Equal(t, "pending", cfg.UnreleasedDir)
		assert.Equal(t, "CHANGES.md", cfg.ChangelogPath)
		assert.Len(t, cfg.Kinds, 2)
	})

	t.Run("should return error for invalid yaml", func(t *testing.T) {
		t.Parallel()

		// given
		dir := t.TempDir()
		path := filepath.Join(dir, ".chlog.yaml")
		require.NoError(t, os.WriteFile(path, []byte("invalid: [yaml: {"), 0o644))

		// when
		_, err := LoadConfigFromPath(path)

		// then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "parsing config file")
	})

	t.Run("should return error when kinds list is empty", func(t *testing.T) {
		t.Parallel()

		// given
		dir := t.TempDir()
		content := `
changesDir: .changes
kinds: []
`
		path := filepath.Join(dir, ".chlog.yaml")
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

		// when
		_, err := LoadConfigFromPath(path)

		// then
		assert.ErrorIs(t, err, ErrEmptyKinds)
	})

	t.Run("should preserve default values for unset fields", func(t *testing.T) {
		t.Parallel()

		// given
		dir := t.TempDir()
		content := `
kinds:
  - label: Fixed
    auto: patch
`
		path := filepath.Join(dir, ".chlog.yaml")
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

		// when
		cfg, err := LoadConfigFromPath(path)

		// then
		require.NoError(t, err)
		assert.Equal(t, ".changes", cfg.ChangesDir)
		assert.Equal(t, "CHANGELOG.md", cfg.ChangelogPath)
	})
}

func TestFindConfig(t *testing.T) {
	t.Parallel()

	t.Run("should find config in start directory", func(t *testing.T) {
		t.Parallel()

		// given
		dir := t.TempDir()
		configPath := filepath.Join(dir, ".chlog.yaml")
		require.NoError(t, os.WriteFile(configPath, []byte("kinds:\n  - label: Fixed\n"), 0o644))

		// when
		found, err := FindConfig(dir)

		// then
		require.NoError(t, err)
		assert.Equal(t, configPath, found)
	})

	t.Run("should find config in parent directory when starting from nested subdir", func(t *testing.T) {
		t.Parallel()

		// given
		dir := t.TempDir()
		configPath := filepath.Join(dir, ".chlog.yaml")
		require.NoError(t, os.WriteFile(configPath, []byte("kinds:\n  - label: Fixed\n"), 0o644))

		child := filepath.Join(dir, "sub", "deep")
		require.NoError(t, os.MkdirAll(child, 0o755))

		// when
		found, err := FindConfig(child)

		// then
		require.NoError(t, err)
		assert.Equal(t, configPath, found)
	})

	t.Run("should find .chlog.yml variant", func(t *testing.T) {
		t.Parallel()

		// given
		dir := t.TempDir()
		configPath := filepath.Join(dir, ".chlog.yml")
		require.NoError(t, os.WriteFile(configPath, []byte("kinds:\n  - label: Fixed\n"), 0o644))

		// when
		found, err := FindConfig(dir)

		// then
		require.NoError(t, err)
		assert.Equal(t, configPath, found)
	})

	t.Run("should return error when no config exists", func(t *testing.T) {
		t.Parallel()

		// given
		dir := t.TempDir()

		// when
		_, err := FindConfig(dir)

		// then
		assert.ErrorIs(t, err, ErrConfigNotFound)
	})
}

func TestFindKind(t *testing.T) {
	t.Parallel()

	t.Run("should find kind case-insensitively", func(t *testing.T) {
		t.Parallel()

		// given
		cfg := DefaultConfig()

		// when
		kind, found := cfg.FindKind("added")

		// then
		assert.True(t, found)
		assert.Equal(t, "Added", kind.Label)
	})

	t.Run("should return false for unknown kind", func(t *testing.T) {
		t.Parallel()

		// given
		cfg := DefaultConfig()

		// when
		_, found := cfg.FindKind("unknown")

		// then
		assert.False(t, found)
	})
}

func TestUnreleasedPath(t *testing.T) {
	t.Parallel()

	t.Run("should join changes dir and unreleased dir", func(t *testing.T) {
		t.Parallel()

		// given
		cfg := DefaultConfig()

		// when
		path := cfg.UnreleasedPath()

		// then
		assert.Equal(t, filepath.Join(".changes", "unreleased"), path)
	})
}

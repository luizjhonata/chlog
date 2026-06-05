//go:build unit

package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindGitDir(t *testing.T) {
	t.Parallel()

	t.Run("should find .git directory in start directory", func(t *testing.T) {
		t.Parallel()

		// given
		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))

		// when
		found, err := FindGitDir(dir)

		// then
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(dir, ".git"), found)
	})

	t.Run("should find .git directory in parent when starting from nested subdir", func(t *testing.T) {
		t.Parallel()

		// given
		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))

		child := filepath.Join(dir, "sub", "deep")
		require.NoError(t, os.MkdirAll(child, 0o755))

		// when
		found, err := FindGitDir(child)

		// then
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(dir, ".git"), found)
	})

	t.Run("should return error when no .git directory exists", func(t *testing.T) {
		t.Parallel()

		// given
		dir := t.TempDir()

		// when
		_, err := FindGitDir(dir)

		// then
		assert.ErrorIs(t, err, ErrGitDirNotFound)
	})
}

//go:build unit

package internal

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	timeOne   = time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)
	timeTwo   = time.Date(2026, 5, 26, 11, 0, 0, 0, time.UTC)
	timeThree = time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
)

func TestChangeMarshalRoundtrip(t *testing.T) {
	t.Parallel()

	t.Run("should marshal and unmarshal consistently", func(t *testing.T) {
		t.Parallel()

		// given
		original := &Change{
			Kind: "Fixed",
			Body: "resolve timeout on file uploads",
			Time: timeOne,
		}

		// when
		data, err := original.Marshal()
		require.NoError(t, err)

		assert.Contains(t, string(data), "kind: 'Fixed'")
		assert.Contains(t, string(data), "body: 'resolve timeout on file uploads'")
		assert.Contains(t, string(data), "time: '2026-05-26T10:00:00Z'")

		dir := t.TempDir()
		path := filepath.Join(dir, "change.yaml")
		require.NoError(t, os.WriteFile(path, data, 0o644))

		loaded, err := LoadChange(path)

		// then
		require.NoError(t, err)
		assert.Equal(t, original.Kind, loaded.Kind)
		assert.Equal(t, original.Body, loaded.Body)
		assert.False(t, loaded.Breaking)
		assert.True(t, original.Time.Equal(loaded.Time))
	})

	t.Run("should omit breaking field when change is not breaking", func(t *testing.T) {
		t.Parallel()

		// given
		original := &Change{Kind: "Fixed", Body: "fix a bug", Time: timeOne}

		// when
		data, err := original.Marshal()

		// then
		require.NoError(t, err)
		assert.NotContains(t, string(data), "breaking")
	})

	t.Run("should marshal and unmarshal breaking flag when change is breaking", func(t *testing.T) {
		t.Parallel()

		// given
		original := &Change{Kind: "Changed", Body: "rename public API", Breaking: true, Time: timeOne}

		// when
		data, err := original.Marshal()
		require.NoError(t, err)

		assert.Contains(t, string(data), "breaking: true")

		dir := t.TempDir()
		path := filepath.Join(dir, "change.yaml")
		require.NoError(t, os.WriteFile(path, data, 0o644))

		loaded, err := LoadChange(path)

		// then
		require.NoError(t, err)
		assert.True(t, loaded.Breaking)
	})
}

func TestLoadChange(t *testing.T) {
	t.Parallel()

	t.Run("should load valid change file", func(t *testing.T) {
		t.Parallel()

		// given
		dir := t.TempDir()
		content := "kind: Added\nbody: add new feature\ntime: 2026-05-26T10:00:00Z\n"
		path := filepath.Join(dir, "change.yaml")
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

		// when
		c, err := LoadChange(path)

		// then
		require.NoError(t, err)
		assert.Equal(t, "Added", c.Kind)
		assert.Equal(t, "add new feature", c.Body)
	})

	t.Run("should return error for non-existent file", func(t *testing.T) {
		t.Parallel()

		// when
		_, err := LoadChange("/non/existent/path.yaml")

		// then
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "reading change file")
	})
}

func TestSortChanges(t *testing.T) {
	t.Parallel()

	kinds := []KindConfig{
		{Label: "Added"},
		{Label: "Changed"},
		{Label: "Fixed"},
	}

	t.Run("should sort by kind index then by time newest first", func(t *testing.T) {
		t.Parallel()

		// given
		changes := []Change{
			{Kind: "Fixed", Body: "fix old", Time: timeOne},
			{Kind: "Added", Body: "add new", Time: timeTwo},
			{Kind: "Fixed", Body: "fix new", Time: timeThree},
			{Kind: "Added", Body: "add old", Time: timeOne},
		}

		// when
		SortChanges(changes, kinds)

		// then
		assert.Equal(t, "Added", changes[0].Kind)
		assert.Equal(t, "add new", changes[0].Body)
		assert.Equal(t, "Added", changes[1].Kind)
		assert.Equal(t, "add old", changes[1].Body)
		assert.Equal(t, "Fixed", changes[2].Kind)
		assert.Equal(t, "fix new", changes[2].Body)
		assert.Equal(t, "Fixed", changes[3].Kind)
		assert.Equal(t, "fix old", changes[3].Body)
	})

	t.Run("should place unknown kinds at the end", func(t *testing.T) {
		t.Parallel()

		// given
		changes := []Change{
			{Kind: "Unknown", Body: "mystery", Time: timeOne},
			{Kind: "Added", Body: "add something", Time: timeOne},
		}

		// when
		SortChanges(changes, kinds)

		// then
		assert.Equal(t, "Added", changes[0].Kind)
		assert.Equal(t, "Unknown", changes[1].Kind)
	})

	t.Run("should handle empty slice", func(t *testing.T) {
		t.Parallel()

		// given
		var changes []Change

		// when
		SortChanges(changes, kinds)

		// then
		assert.Empty(t, changes)
	})
}

//go:build unit

package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func executeCheck() (*bytes.Buffer, error) {
	var out bytes.Buffer

	cmd := newRootCmd("test")
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"check"})

	return &out, cmd.Execute()
}

func TestCheckCmd(t *testing.T) {
	defaultKinds := "kinds:\n  - label: Added\n    auto: minor\n"

	t.Run("should succeed when fragments exist", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, defaultKinds)
		writeFragment(t, dir, "001.yaml", "Added", "new feature")
		chdir(t, dir)

		// when
		out, err := executeCheck()

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "1 unreleased fragment(s) found")
	})

	t.Run("should count multiple fragments", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, defaultKinds)
		writeFragment(t, dir, "001.yaml", "Added", "first feature")
		writeFragment(t, dir, "002.yaml", "Added", "second feature")
		writeFragment(t, dir, "003.yaml", "Added", "third feature")
		chdir(t, dir)

		// when
		out, err := executeCheck()

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "3 unreleased fragment(s) found")
	})

	t.Run("should fail when no fragments exist", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, defaultKinds)
		chdir(t, dir)

		// when
		_, err := executeCheck()

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no unreleased fragments found")
	})

	t.Run("should fail when unreleased directory does not exist", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, defaultKinds)
		chdir(t, dir)

		// when
		_, err := executeCheck()

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no unreleased fragments found")
	})

	t.Run("should ignore non-yaml files in unreleased directory", func(t *testing.T) {
		// given
		dir := t.TempDir()
		writeConfig(t, dir, defaultKinds)
		writeFragment(t, dir, "001.yaml", "Added", "real fragment")
		writeFragment(t, dir, "002.yaml", "Added", "another fragment")
		chdir(t, dir)

		// when
		out, err := executeCheck()

		// then
		require.NoError(t, err)
		assert.Contains(t, out.String(), "2 unreleased fragment(s) found")
	})
}

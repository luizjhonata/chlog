package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luizjhonata/chlog/internal"
	"github.com/spf13/cobra"
)

var (
	errHookExists   = errors.New("pre-commit hook already exists and is not managed by chlog; use --force to overwrite")
	errHookNotOwned = errors.New("pre-commit hook is not managed by chlog; refusing to remove")
)

const hookMarker = "# chlog:managed"

const hookScript = `#!/bin/sh
# chlog:managed
chlog check
`

func newHookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hook",
		Short: "Manage Git pre-commit hook for changelog validation",
	}

	cmd.AddCommand(newHookInstallCmd())
	cmd.AddCommand(newHookUninstallCmd())

	return cmd
}

func newHookInstallCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install a pre-commit hook that runs chlog check",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			gitDir, err := internal.FindGitDirUpward()
			if err != nil {
				return err
			}

			hooksDir := filepath.Join(gitDir, "hooks")

			err = os.MkdirAll(hooksDir, 0o750)
			if err != nil {
				return fmt.Errorf("creating hooks directory: %w", err)
			}

			hookPath := filepath.Join(hooksDir, "pre-commit")

			existing, readErr := os.ReadFile(hookPath)
			if readErr == nil {
				if strings.Contains(string(existing), hookMarker) {
					fmt.Fprintln(cmd.OutOrStdout(), "pre-commit hook already installed")
					return nil
				}

				if !force {
					return errHookExists
				}
			}

			err = os.WriteFile(hookPath, []byte(hookScript), 0o755) //nolint:gosec // G306 - hook must be executable
			if err != nil {
				return fmt.Errorf("writing pre-commit hook: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "pre-commit hook installed")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "overwrite an existing non-chlog hook")

	return cmd
}

func newHookUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the chlog-managed pre-commit hook",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			gitDir, err := internal.FindGitDirUpward()
			if err != nil {
				return err
			}

			hookPath := filepath.Join(gitDir, "hooks", "pre-commit")

			existing, readErr := os.ReadFile(hookPath)
			if readErr != nil {
				if os.IsNotExist(readErr) {
					fmt.Fprintln(cmd.OutOrStdout(), "no pre-commit hook installed")
					return nil
				}
				return fmt.Errorf("reading pre-commit hook: %w", readErr)
			}

			if !strings.Contains(string(existing), hookMarker) {
				return errHookNotOwned
			}

			err = os.Remove(hookPath)
			if err != nil {
				return fmt.Errorf("removing pre-commit hook: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "pre-commit hook removed")
			return nil
		},
	}
}

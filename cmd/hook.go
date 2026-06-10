package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/luizjhonata/chlog/internal"
	"github.com/spf13/cobra"
)

var errHookBlockNotFound = errors.New("pre-commit hook does not contain a chlog block")

const (
	hookMarker      = "# chlog:managed"
	localBlockStart = "# chlog:start"
	localBlockEnd   = "# chlog:end"
)

const globalHookScript = `#!/bin/sh
# chlog:managed

hook_name="$(basename "$0")"

git_dir="$(git rev-parse --git-dir 2>/dev/null)"
if [ -n "$git_dir" ] && [ -f "$git_dir/hooks/$hook_name" ] && [ -x "$git_dir/hooks/$hook_name" ]; then
    "$git_dir/hooks/$hook_name" "$@"
    status=$?
    if [ $status -ne 0 ]; then
        exit $status
    fi
fi

if [ "$hook_name" = "pre-commit" ]; then
    root="$(git rev-parse --show-toplevel 2>/dev/null)"
    if [ -f "$root/.chlog.yaml" ] || [ -f "$root/.chlog.yml" ]; then
        if command -v chlog >/dev/null 2>&1; then
            chlog check
        fi
    fi
fi
`

const localHookBlock = `# chlog:start
root="$(git rev-parse --show-toplevel 2>/dev/null)"
if [ -f "$root/.chlog.yaml" ] || [ -f "$root/.chlog.yml" ]; then
    if command -v chlog >/dev/null 2>&1; then
        chlog check
    fi
fi
# chlog:end
`

func passthroughHookNames() []string {
	return []string{
		"prepare-commit-msg",
		"commit-msg",
		"post-commit",
		"pre-rebase",
		"post-checkout",
		"post-merge",
		"pre-push",
	}
}

func newHookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hook",
		Short: "Manage Git hooks for changelog validation",
	}

	cmd.AddCommand(newHookInstallCmd())
	cmd.AddCommand(newHookUninstallCmd())

	return cmd
}

func newHookInstallCmd() *cobra.Command {
	var local, force bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install a hook that runs chlog check before commits",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if local {
				return installLocal(cmd, force)
			}

			return installGlobal(cmd, force)
		},
	}

	cmd.Flags().BoolVar(&local, "local", false, "inject chlog block into the current repository hook")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "override existing hook configuration")

	return cmd
}

func newHookUninstallCmd() *cobra.Command {
	var local bool

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the chlog-managed hook",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if local {
				return uninstallLocal(cmd)
			}

			return uninstallGlobal(cmd)
		},
	}

	cmd.Flags().BoolVar(&local, "local", false, "remove chlog block from the current repository hook")

	return cmd
}

func installGlobal(cmd *cobra.Command, force bool) error {
	dir, err := globalHooksDir()
	if err != nil {
		return err
	}

	currentPath := getGlobalHooksPath()

	if currentPath == dir {
		hookPath := filepath.Join(dir, "pre-commit")
		if data, readErr := os.ReadFile(hookPath); readErr == nil && strings.Contains(string(data), hookMarker) {
			fmt.Fprintln(cmd.OutOrStdout(), "chlog hook is already installed")

			return nil
		}
	}

	if currentPath != "" && currentPath != dir && !force {
		return fmt.Errorf(
			"core.hooksPath is already configured to %q; use --force to override",
			currentPath,
		)
	}

	err = os.MkdirAll(dir, 0o750)
	if err != nil {
		return fmt.Errorf("creating hooks directory: %w", err)
	}

	hookPath := filepath.Join(dir, "pre-commit")

	//nolint:gosec // G306 - hook must be executable
	err = os.WriteFile(hookPath, []byte(globalHookScript), 0o755)
	if err != nil {
		return fmt.Errorf("writing pre-commit hook: %w", err)
	}

	for _, name := range passthroughHookNames() {
		linkPath := filepath.Join(dir, name)
		_ = os.Remove(linkPath)

		if linkErr := os.Symlink("pre-commit", linkPath); linkErr != nil {
			return fmt.Errorf("creating symlink for %s: %w", name, linkErr)
		}
	}

	setCmd, err := gitCommand("config", "--global", "core.hooksPath", dir)
	if err != nil {
		return err
	}

	err = setCmd.Run()
	if err != nil {
		return fmt.Errorf("setting core.hooksPath: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "chlog hook installed globally (%s)\n", dir)

	return nil
}

func uninstallGlobal(cmd *cobra.Command) error {
	dir, err := globalHooksDir()
	if err != nil {
		return err
	}

	currentPath := getGlobalHooksPath()

	if currentPath == "" {
		fmt.Fprintln(cmd.OutOrStdout(), "chlog hook is not installed")

		return nil
	}

	if currentPath != dir {
		return fmt.Errorf("core.hooksPath is set to %q, which is not managed by chlog", currentPath)
	}

	unsetCmd, err := gitCommand("config", "--global", "--unset", "core.hooksPath")
	if err != nil {
		return err
	}

	err = unsetCmd.Run()
	if err != nil {
		return fmt.Errorf("unsetting core.hooksPath: %w", err)
	}

	err = os.RemoveAll(dir)
	if err != nil {
		return fmt.Errorf("removing hooks directory: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "chlog hook removed")

	return nil
}

func installLocal(cmd *cobra.Command, force bool) error {
	gitDir, err := internal.FindGitDirUpward()
	if err != nil {
		return err
	}

	hooksDir := resolveLocalHooksDir(gitDir)
	hookPath := filepath.Join(hooksDir, "pre-commit")

	existing, readErr := os.ReadFile(hookPath)

	if readErr == nil {
		content := string(existing)

		if strings.Contains(content, localBlockStart) && !force {
			fmt.Fprintln(cmd.OutOrStdout(), "chlog hook is already installed")

			return nil
		}

		content = removeChlogBlock(content)
		trimmed := strings.TrimRight(content, "\n")

		//nolint:gosec // G306 - hook must be executable
		err = os.WriteFile(hookPath, []byte(trimmed+"\n\n"+localHookBlock), 0o755)
		if err != nil {
			return fmt.Errorf("writing pre-commit hook: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "chlog hook installed locally (%s)\n", hookPath)

		return nil
	}

	err = os.MkdirAll(hooksDir, 0o750)
	if err != nil {
		return fmt.Errorf("creating hooks directory: %w", err)
	}

	//nolint:gosec // G306 - hook must be executable
	err = os.WriteFile(hookPath, []byte("#!/bin/sh\n"+localHookBlock), 0o755)
	if err != nil {
		return fmt.Errorf("writing pre-commit hook: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "chlog hook installed locally (%s)\n", hookPath)

	return nil
}

func uninstallLocal(cmd *cobra.Command) error {
	gitDir, err := internal.FindGitDirUpward()
	if err != nil {
		return err
	}

	hooksDir := resolveLocalHooksDir(gitDir)
	hookPath := filepath.Join(hooksDir, "pre-commit")

	existing, readErr := os.ReadFile(hookPath)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			fmt.Fprintln(cmd.OutOrStdout(), "no chlog hook installed")

			return nil
		}

		return fmt.Errorf("reading pre-commit hook: %w", readErr)
	}

	content := string(existing)

	if !strings.Contains(content, localBlockStart) {
		return errHookBlockNotFound
	}

	remaining := strings.TrimRight(removeChlogBlock(content), "\n")

	if remaining == "" || remaining == "#!/bin/sh" {
		err = os.Remove(hookPath)
		if err != nil {
			return fmt.Errorf("removing pre-commit hook: %w", err)
		}
	} else {
		//nolint:gosec // G306 - hook must be executable
		err = os.WriteFile(hookPath, []byte(remaining+"\n"), 0o755)
		if err != nil {
			return fmt.Errorf("writing pre-commit hook: %w", err)
		}
	}

	fmt.Fprintln(cmd.OutOrStdout(), "chlog hook removed")

	return nil
}

func gitCommand(args ...string) (*exec.Cmd, error) {
	gitPath, lookErr := exec.LookPath("git")
	if lookErr != nil {
		return nil, fmt.Errorf("git executable not found: %w", lookErr)
	}

	return exec.CommandContext(context.Background(), gitPath, args...), nil
}

func globalHooksDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("getting user config directory: %w", err)
	}

	return filepath.Join(configDir, "chlog", "hooks"), nil
}

func getGlobalHooksPath() string {
	cmd, err := gitCommand("config", "--global", "--get", "core.hooksPath")
	if err != nil {
		return ""
	}

	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}

func resolveLocalHooksDir(gitDir string) string {
	repoRoot := filepath.Dir(gitDir)

	cmd, err := gitCommand("-C", repoRoot, "config", "--local", "--get", "core.hooksPath")
	if err != nil {
		return filepath.Join(gitDir, "hooks")
	}

	out, err := cmd.Output()
	if err == nil {
		localPath := strings.TrimSpace(string(out))
		if localPath != "" {
			if filepath.IsAbs(localPath) {
				return localPath
			}

			return filepath.Join(repoRoot, localPath)
		}
	}

	return filepath.Join(gitDir, "hooks")
}

func removeChlogBlock(content string) string {
	before, rest, found := strings.Cut(content, localBlockStart)
	if !found {
		return content
	}

	_, after, found := strings.Cut(rest, localBlockEnd)
	if !found {
		return content
	}

	return before + strings.TrimPrefix(after, "\n")
}

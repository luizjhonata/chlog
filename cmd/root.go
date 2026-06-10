package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "chlog",
		Short:        "Fragment-based changelog management",
		Long:         "chlog eliminates merge conflicts on CHANGELOG.md by using individual fragment files per change.",
		Version:      version,
		SilenceUsage: true,
	}
	cmd.SetVersionTemplate(fmt.Sprintf("chlog version: %s\n", version))
	cmd.AddCommand(newNewCmd())
	cmd.AddCommand(newBatchCmd())
	cmd.AddCommand(newMergeCmd())
	cmd.AddCommand(newCheckCmd())
	cmd.AddCommand(newHookCmd())
	cmd.AddCommand(newAiCmd())

	return cmd
}

func Execute(version string) {
	if err := newRootCmd(version).Execute(); err != nil {
		os.Exit(1)
	}
}

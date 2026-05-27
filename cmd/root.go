package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "chlog",
		Short:   "Fragment-based changelog management",
		Long:    "chlog eliminates merge conflicts on CHANGELOG.md by using individual fragment files per change.",
		Version: version,
	}
	cmd.SetVersionTemplate(fmt.Sprintf("chlog version: %s\n", version))

	return cmd
}

func Execute(version string) {
	if err := newRootCmd(version).Execute(); err != nil {
		os.Exit(1)
	}
}

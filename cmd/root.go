package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "chlog",
	Short: "Fragment-based changelog management",
	Long:  "chlog eliminates merge conflicts on CHANGELOG.md by using individual fragment files per change.",
}

func Execute(version string) {
	rootCmd.Version = version
	rootCmd.SetVersionTemplate(fmt.Sprintf("chlog version: %s\n", version))

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

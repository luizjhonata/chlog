package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/luizjhonata/chlog/internal"
	"github.com/spf13/cobra"
)

func newCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Verify that unreleased fragments exist",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			count, err := countFragments(cfg.UnreleasedPath())
			if err != nil {
				return err
			}

			if count == 0 {
				return errNoFragments
			}

			fmt.Fprintf(cmd.OutOrStdout(), "%d unreleased fragment(s) found\n", count)
			return nil
		},
	}
}

func countFragments(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("reading unreleased directory: %w", err)
	}

	count := 0

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			count++
		}
	}

	return count, nil
}

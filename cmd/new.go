package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/luizjhonata/chlog/internal"
	"github.com/spf13/cobra"
)

func newNewCmd() *cobra.Command {
	var kind, body string

	var breaking bool

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new changelog fragment",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			matched, found := cfg.FindKind(kind)
			if !found {
				return fmt.Errorf("unknown kind %q", kind)
			}

			change := &internal.Change{
				Kind:     matched.Label,
				Body:     body,
				Breaking: breaking,
				Time:     time.Now().UTC(),
			}

			data, err := change.Marshal()
			if err != nil {
				return err
			}

			dir := cfg.UnreleasedPath()

			err = os.MkdirAll(dir, 0o750)
			if err != nil {
				return fmt.Errorf("creating directory %s: %w", dir, err)
			}

			filename, err := generateFragmentFilename()
			if err != nil {
				return err
			}

			path := filepath.Join(dir, filename)

			err = os.WriteFile(path, data, 0o600)
			if err != nil {
				return fmt.Errorf("writing fragment: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		},
	}

	cmd.Flags().StringVarP(&kind, "kind", "k", "", "kind of change (e.g., Added, Fixed)")
	cmd.Flags().StringVarP(&body, "body", "b", "", "description of the change")
	cmd.Flags().BoolVar(&breaking, "breaking", false,
		"mark the change as a backward-incompatible (breaking) change; forces a major bump")
	_ = cmd.MarkFlagRequired("kind")
	_ = cmd.MarkFlagRequired("body")

	return cmd
}

const randomSuffixBytes = 2

func generateFragmentFilename() (string, error) {
	b := make([]byte, randomSuffixBytes)

	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("generating random bytes: %w", err)
	}

	return fmt.Sprintf("%d-%s.yaml", time.Now().UnixNano(), hex.EncodeToString(b)), nil
}

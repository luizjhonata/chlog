package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/luizjhonata/chlog/internal"
	"github.com/spf13/cobra"
)

var errNoVersionFiles = errors.New("no version files found")

type versionFile struct {
	Path    string
	Version *semver.Version
	Content string
}

func newMergeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "merge",
		Short: "Insert version files into CHANGELOG.md",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			files, err := readVersionFiles(cfg.ChangesDir)
			if err != nil {
				return err
			}

			content := buildVersionContent(files)

			err = insertIntoChangelog(cfg.ChangelogPath, content)
			if err != nil {
				return err
			}

			for _, f := range files {
				err = os.Remove(f.Path)
				if err != nil {
					return fmt.Errorf("deleting version file %s: %w", f.Path, err)
				}
			}

			fmt.Fprintf(cmd.OutOrStdout(), "merged %d version(s) into %s\n", len(files), cfg.ChangelogPath)
			return nil
		},
	}
}

func readVersionFiles(changesDir string) ([]versionFile, error) {
	entries, err := os.ReadDir(changesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errNoVersionFiles
		}
		return nil, fmt.Errorf("reading changes directory: %w", err)
	}

	var files []versionFile

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "v") || !strings.HasSuffix(name, ".md") {
			continue
		}

		vStr := strings.TrimSuffix(strings.TrimPrefix(name, "v"), ".md")

		v, parseErr := semver.NewVersion(vStr)
		if parseErr != nil {
			continue
		}

		path := filepath.Join(changesDir, name)

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("reading version file %s: %w", path, readErr)
		}

		files = append(files, versionFile{
			Path:    path,
			Version: v,
			Content: string(data),
		})
	}

	if len(files) == 0 {
		return nil, errNoVersionFiles
	}

	slices.SortFunc(files, func(a, b versionFile) int {
		return b.Version.Compare(a.Version)
	})

	return files, nil
}

func buildVersionContent(files []versionFile) string {
	parts := make([]string, len(files))

	for i, f := range files {
		parts[i] = strings.TrimRight(f.Content, "\n")
	}

	return strings.Join(parts, "\n\n") + "\n"
}

func insertIntoChangelog(changelogPath string, versionContent string) error {
	cleanPath := filepath.Clean(changelogPath)

	existing, err := os.ReadFile(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			skeleton := "# Changelog\n\n" + versionContent
			return os.WriteFile(cleanPath, []byte(skeleton), 0o600)
		}
		return fmt.Errorf("reading changelog: %w", err)
	}

	content := string(existing)
	insertIdx := findVersionInsertionPoint(content)

	var result string

	if insertIdx < 0 {
		result = strings.TrimRight(content, "\n") + "\n\n" + versionContent
	} else {
		result = content[:insertIdx] + versionContent + "\n" + content[insertIdx:]
	}

	return os.WriteFile(cleanPath, []byte(result), 0o600) //nolint:gosec // path comes from config, not user input
}

func findVersionInsertionPoint(content string) int {
	offset := 0

	for line := range strings.SplitSeq(content, "\n") {
		if isVersionHeader(line) {
			return offset
		}

		offset += len(line) + 1
	}

	return -1
}

func isVersionHeader(line string) bool {
	if !strings.HasPrefix(line, "## [") {
		return false
	}

	rest := strings.TrimPrefix(line, "## [")

	return len(rest) > 0 && rest[0] >= '0' && rest[0] <= '9'
}

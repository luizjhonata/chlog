package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/luizjhonata/chlog/internal"
	"github.com/spf13/cobra"
)

const (
	bumpMajor = "major"
	bumpMinor = "minor"
	bumpPatch = "patch"
	bumpAuto  = "auto"
)

var errNoFragments = errors.New("no unreleased fragments found")

func newBatchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "batch <version|major|minor|patch|auto>",
		Short: "Compile unreleased fragments into a version file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			changes, fragmentPaths, err := readFragments(cfg)
			if err != nil {
				return err
			}

			version, err := resolveVersion(args[0], cfg, changes)
			if err != nil {
				return err
			}

			internal.SortChanges(changes, cfg.Kinds)

			content, err := renderVersionFile(cfg, version, time.Now().UTC(), changes)
			if err != nil {
				return err
			}

			versionPath := filepath.Join(cfg.ChangesDir, "v"+version.String()+".md")

			err = os.WriteFile(versionPath, []byte(content), 0o600)
			if err != nil {
				return fmt.Errorf("writing version file: %w", err)
			}

			for _, path := range fragmentPaths {
				err = os.Remove(path)
				if err != nil {
					return fmt.Errorf("deleting fragment %s: %w", path, err)
				}
			}

			fmt.Fprintln(cmd.OutOrStdout(), versionPath)
			return nil
		},
	}
}

func readFragments(cfg *internal.Config) ([]internal.Change, []string, error) {
	dir := cfg.UnreleasedPath()

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, errNoFragments
		}
		return nil, nil, fmt.Errorf("reading unreleased directory: %w", err)
	}

	var changes []internal.Change
	var paths []string

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		path := filepath.Join(dir, entry.Name())

		change, loadErr := internal.LoadChange(path)
		if loadErr != nil {
			return nil, nil, loadErr
		}

		changes = append(changes, *change)
		paths = append(paths, path)
	}

	if len(changes) == 0 {
		return nil, nil, errNoFragments
	}

	return changes, paths, nil
}

func resolveVersion(input string, cfg *internal.Config, changes []internal.Change) (*semver.Version, error) {
	if v, err := semver.NewVersion(input); err == nil {
		return v, nil
	}

	latest := findLatestVersion(cfg)

	var bumpType string

	switch input {
	case bumpMajor, bumpMinor, bumpPatch:
		bumpType = input
	case bumpAuto:
		bumpType = inferBumpType(cfg, changes)
	default:
		return nil, fmt.Errorf("invalid version argument %q: use a semver, major, minor, patch, or auto", input)
	}

	return bumpVersion(latest, bumpType)
}

// findLatestVersion resolves the current version by taking the highest across
// every source that may know it: leftover version files, git tags, and the
// CHANGELOG.md headings. Version files alone are insufficient because [merge]
// deletes them, which would otherwise reset relative bumps to 0.0.0.
func findLatestVersion(cfg *internal.Config) *semver.Version {
	var candidates []*semver.Version

	candidates = append(candidates, versionsFromChangesDir(cfg.ChangesDir)...)
	candidates = append(candidates, versionsFromChangelog(cfg.ChangelogPath)...)
	candidates = append(candidates, versionsFromGitTags()...)

	latest := highestVersion(candidates)
	if latest == nil {
		return semver.New(0, 0, 0, "", "")
	}

	return latest
}

func highestVersion(versions []*semver.Version) *semver.Version {
	var latest *semver.Version

	for _, v := range versions {
		if latest == nil || v.GreaterThan(latest) {
			latest = v
		}
	}

	return latest
}

func versionsFromChangesDir(changesDir string) []*semver.Version {
	entries, err := os.ReadDir(changesDir)
	if err != nil {
		return nil
	}

	var versions []*semver.Version

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

		versions = append(versions, v)
	}

	return versions
}

func versionsFromChangelog(changelogPath string) []*semver.Version {
	data, err := os.ReadFile(changelogPath)
	if err != nil {
		return nil
	}

	var versions []*semver.Version

	for line := range strings.SplitSeq(string(data), "\n") {
		if !strings.HasPrefix(line, "## [") {
			continue
		}

		start := strings.Index(line, "[")
		end := strings.Index(line, "]")
		if start < 0 || end <= start+1 {
			continue
		}

		v, parseErr := semver.NewVersion(line[start+1 : end])
		if parseErr != nil {
			continue
		}

		versions = append(versions, v)
	}

	return versions
}

func versionsFromGitTags() []*semver.Version {
	cmd, err := gitCommand("tag")
	if err != nil {
		return nil
	}

	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	var versions []*semver.Version

	for line := range strings.SplitSeq(string(out), "\n") {
		tag := strings.TrimSpace(line)
		if tag == "" {
			continue
		}

		v, parseErr := semver.NewVersion(tag)
		if parseErr != nil {
			continue
		}

		versions = append(versions, v)
	}

	return versions
}

func inferBumpType(cfg *internal.Config, changes []internal.Change) string {
	highest := bumpPatch

	for _, c := range changes {
		kind, found := cfg.FindKind(c.Kind)
		if !found || kind.Auto == "" {
			continue
		}

		if kind.Auto == bumpMajor {
			return bumpMajor
		}

		if kind.Auto == bumpMinor {
			highest = bumpMinor
		}
	}

	return highest
}

func bumpVersion(v *semver.Version, bumpType string) (*semver.Version, error) {
	var next semver.Version

	switch bumpType {
	case bumpMajor:
		next = v.IncMajor()
	case bumpMinor:
		next = v.IncMinor()
	case bumpPatch:
		next = v.IncPatch()
	default:
		return nil, fmt.Errorf("unknown bump type: %s", bumpType)
	}

	return &next, nil
}

func renderVersionFile(
	cfg *internal.Config,
	version *semver.Version,
	batchTime time.Time,
	changes []internal.Change,
) (string, error) {
	var buf bytes.Buffer

	versionTmpl, err := template.New("version").Parse(cfg.VersionFormat)
	if err != nil {
		return "", fmt.Errorf("parsing version format template: %w", err)
	}

	versionData := struct {
		Version string
		Time    time.Time
	}{
		Version: version.String(),
		Time:    batchTime,
	}

	err = versionTmpl.Execute(&buf, versionData)
	if err != nil {
		return "", fmt.Errorf("rendering version header: %w", err)
	}

	buf.WriteString("\n")

	kindTmpl, err := template.New("kind").Parse(cfg.KindFormat)
	if err != nil {
		return "", fmt.Errorf("parsing kind format template: %w", err)
	}

	changeTmpl, err := template.New("change").Parse(cfg.ChangeFormat)
	if err != nil {
		return "", fmt.Errorf("parsing change format template: %w", err)
	}

	lastKind := ""

	for _, c := range changes {
		if c.Kind != lastKind {
			buf.WriteString("\n")

			err = kindTmpl.Execute(&buf, struct{ Kind string }{Kind: c.Kind})
			if err != nil {
				return "", fmt.Errorf("rendering kind header: %w", err)
			}

			buf.WriteString("\n\n")
			lastKind = c.Kind
		}

		err = changeTmpl.Execute(&buf, struct{ Body string }{Body: c.Body})
		if err != nil {
			return "", fmt.Errorf("rendering change line: %w", err)
		}

		buf.WriteString("\n")
	}

	return buf.String(), nil
}

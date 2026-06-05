package internal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	ErrConfigNotFound = errors.New("config file not found")
	ErrEmptyKinds     = errors.New("kinds list must not be empty")
)

func configFileNames() []string {
	return []string{".chlog.yaml", ".chlog.yml"}
}

type Config struct {
	ChangesDir    string       `yaml:"changesDir"`
	UnreleasedDir string       `yaml:"unreleasedDir"`
	ChangelogPath string       `yaml:"changelogPath"`
	VersionFormat string       `yaml:"versionFormat"`
	KindFormat    string       `yaml:"kindFormat"`
	ChangeFormat  string       `yaml:"changeFormat"`
	Kinds         []KindConfig `yaml:"kinds"`
}

type KindConfig struct {
	Label string `yaml:"label"`
	Auto  string `yaml:"auto,omitempty"`
}

func DefaultConfig() *Config {
	return &Config{
		ChangesDir:    ".changes",
		UnreleasedDir: "unreleased",
		ChangelogPath: "CHANGELOG.md",
		VersionFormat: `## [{{.Version}}] - {{.Time.Format "2006-01-02"}}`,
		KindFormat:    `### {{.Kind}}`,
		ChangeFormat:  `- {{.Body}}`,
		Kinds: []KindConfig{
			{Label: "Added", Auto: "minor"},
			{Label: "Changed", Auto: "major"},
			{Label: "Deprecated", Auto: "minor"},
			{Label: "Removed", Auto: "major"},
			{Label: "Fixed", Auto: "patch"},
			{Label: "Security", Auto: "patch"},
		},
	}
}

func LoadConfig() (*Config, error) {
	path, err := FindConfigUpward()
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	return LoadConfigFromPath(path)
}

func LoadConfigFromPath(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	cfg := DefaultConfig()

	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	err = cfg.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

func FindConfig(startDir string) (string, error) {
	dir := startDir

	for {
		for _, name := range configFileNames() {
			path := filepath.Join(dir, name)
			if _, statErr := os.Stat(path); statErr == nil {
				return path, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrConfigNotFound
		}
		dir = parent
	}
}

func FindConfigUpward() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}

	return FindConfig(dir)
}

func (c *Config) Validate() error {
	if len(c.Kinds) == 0 {
		return ErrEmptyKinds
	}
	return nil
}

func (c *Config) FindKind(name string) (*KindConfig, bool) {
	for i := range c.Kinds {
		if strings.EqualFold(c.Kinds[i].Label, name) {
			return &c.Kinds[i], true
		}
	}
	return nil, false
}

func (c *Config) UnreleasedPath() string {
	return filepath.Join(c.ChangesDir, c.UnreleasedDir)
}

package internal

import (
	"cmp"
	"fmt"
	"os"
	"slices"
	"time"

	"gopkg.in/yaml.v3"
)

type Change struct {
	Kind     string    `yaml:"kind"`
	Body     string    `yaml:"body"`
	Breaking bool      `yaml:"breaking,omitempty"`
	Time     time.Time `yaml:"time"`
}

func LoadChange(path string) (*Change, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading change file %s: %w", path, err)
	}

	var c Change

	err = yaml.Unmarshal(data, &c)
	if err != nil {
		return nil, fmt.Errorf("parsing change file %s: %w", path, err)
	}

	return &c, nil
}

func (c *Change) Marshal() ([]byte, error) {
	timeStr := c.Time.UTC().Format(time.RFC3339Nano)

	content := []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "kind"},
		{Kind: yaml.ScalarNode, Value: c.Kind, Style: yaml.SingleQuotedStyle},
		{Kind: yaml.ScalarNode, Value: "body"},
		{Kind: yaml.ScalarNode, Value: c.Body, Style: yaml.SingleQuotedStyle},
	}

	if c.Breaking {
		content = append(content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "breaking"},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"},
		)
	}

	content = append(content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "time"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: timeStr, Style: yaml.SingleQuotedStyle},
	)

	doc := &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{Kind: yaml.MappingNode, Tag: "!!map", Content: content},
		},
	}

	data, err := yaml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("marshalling change: %w", err)
	}

	return data, nil
}

func SortChanges(changes []Change, kinds []KindConfig) {
	kindIndex := make(map[string]int, len(kinds))
	for i, k := range kinds {
		kindIndex[k.Label] = i
	}

	slices.SortStableFunc(changes, func(a, b Change) int {
		ai, aOk := kindIndex[a.Kind]
		bi, bOk := kindIndex[b.Kind]

		if !aOk {
			ai = len(kinds)
		}
		if !bOk {
			bi = len(kinds)
		}

		if v := cmp.Compare(ai, bi); v != 0 {
			return v
		}

		return b.Time.Compare(a.Time)
	})
}

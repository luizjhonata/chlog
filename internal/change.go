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
	Kind string    `yaml:"kind"`
	Body string    `yaml:"body"`
	Time time.Time `yaml:"time"`
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
	data, err := yaml.Marshal(c)
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

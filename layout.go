package parcours

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Layout struct {
	Columns []Column `yaml:"columns"`
	Filter  *Filter  `yaml:"filter,omitempty"`
}

func LoadLayout(path string) (*Layout, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var layout Layout
	if err := yaml.Unmarshal(data, &layout); err != nil {
		return nil, err
	}

	return &layout, nil
}

package parcours

import (
	"os"

	"gopkg.in/yaml.v3"

	nt "parcours/entity"
)

type Layout struct {
	Columns []nt.Column `yaml:"columns"`
	Filter  nt.Filter   `yaml:"filter,omitempty"`
}

func loadLayout(path string) (*Layout, error) {
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

// promote promotes fields in layout
// Todo: cols not layout here yeah?
func (layout *Layout) promote(store Store) (err error) {

	fields, _, err := store.GetView()
	if err != nil {
		return
	}

	promoted := make(map[string]bool)
	for _, f := range fields {
		promoted[f.Name] = true
	}

	for _, col := range layout.Columns {
		if promoted[col.Field] || col.Demote {
			continue
		}

		err = store.Promote(col.Field)
		if err != nil {
			return
		}
	}
	return
}

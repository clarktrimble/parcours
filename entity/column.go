package entity

type Column struct {
	Field  string `yaml:"field"`
	Width  int    `yaml:"width"`
	Format string `yaml:"format,omitempty"`
	Hidden bool   `yaml:"hidden,omitempty"`
	Demote bool   `yaml:"demote,omitempty"`
	Json   bool   `yaml:"json,omitempty"`

	// Resolved at layout time
	// Todo: dehax
	//FieldIdx  int                `yaml:"-"`
	//Formatter func(Value) string `yaml:"-"`
}

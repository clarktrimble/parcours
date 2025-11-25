package parcours

type Column struct {
	Field  string `yaml:"field"`
	Width  int    `yaml:"width"`
	Format string `yaml:"format,omitempty"`
	Hidden bool   `yaml:"hidden,omitempty"`
	Demote bool   `yaml:"demote,omitempty"`
	Json   bool   `yaml:"json,omitempty"`
}

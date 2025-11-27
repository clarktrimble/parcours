package parcours

import (
	"context"
)

// Todo: look a delish remote_ip logging, borken with "["?
// Todo: better nav ijkl and page up/down
// Todo: full page of lines yeah?

// Logger specifies a contextual, structured logger.
type Logger interface {
	Info(ctx context.Context, msg string, kv ...any)
	Error(ctx context.Context, msg string, err error, kv ...any)
}

// Field represents metadata about a log field.
type Field struct {
	Name string
	Type string
}

// Store specifies a backing datastore.
type Store interface {
	// Name returns the name of the data source
	Name() string
	// Load a file
	Load(path string, last int) (err error)
	// Follow a file
	Follow(ctx context.Context, path string, last int) (err error)
	// Promote a field
	Promote(field string) (err error)
	//SetView Filter and Sort(s)
	SetView(filter *Filter, sorts []Sort) (err error)
	// GetView fields and count
	GetView() (fields []Field, count int, err error)
	// GetPage of log lines
	GetPage(offset, size int) (lines []Line, err error)
	// GetJson returns raw json for a log line
	GetLine(id string) (data map[string]any, err error)
	// Tail streams log lines
	Tail(ctx context.Context) (lines <-chan Line, err error)
}

type Config struct{}

type Parcours struct {
	store  Store
	logger Logger
}

func (cfg *Config) New(store Store, lgr Logger) *Parcours {

	return &Parcours{
		store:  store,
		logger: lgr,
	}
}

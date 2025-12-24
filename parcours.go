package parcours

import (
	"context"

	nt "parcours/entity"
)

// Todo: look at delish remote_ip logging, borken with "["?
// Todo: better nav ijkl and page up/down
// Todo: full page of lines yeah?
// Todo: fold per column (when value repeats)

// Store specifies a backing datastore.
// Todo: rename Get/Set View
// Todo: we now rely on col order from Store, arrange to set
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
	SetView(filter nt.Filter, sorts []nt.Sort) (err error)
	// GetView fields and count
	GetView() (fields []nt.Field, count int, err error)
	// GetPage of log lines
	GetPage(offset, size int) (lines []nt.Line, err error)
	// GetJson returns raw json for a log line
	GetLine(id string) (data map[string]any, err error)
	// Tail streams log lines
	Tail(ctx context.Context) (lines <-chan nt.Line, err error)
}

type Config struct{}

type Parcours struct {
	store  Store
	logger nt.Logger
}

func (cfg *Config) New(store Store, lgr nt.Logger) *Parcours {

	return &Parcours{
		store:  store,
		logger: lgr,
	}
}

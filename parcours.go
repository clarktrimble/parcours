package parcours

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	nt "parcours/entity"
	"parcours/util"
)

// Todo: look at delish remote_ip logging, borken with "["?
// Todo: better nav ijkl and page up/down
// Todo: full page of lines yeah?
// Todo: fold per column (when value repeats)

// Todo: why do I need to reset term after?  (less page/scroll up/down is broken in git diff)
// Todo: linepanel out of range errors after changing font size,  do we get a size msg?
// Todo: try to "stick" to current line id when applying/removing filters
// Todo: slower accel ramp, sensible params
// Todo: look over filter -> ducksql,
// Todo: bug "board requires non-zero ranks and files" when  filtering to 0 lines
// Todo: rethink key-binds holistically
// Todo: look for ignored cmd's thruout
// Todo: disable accel for non-lp

// Todo: cannot paste into piece textinput

// Todo: failed to create table: Catalog Error: Table with name "logs" already exists! -- on new file
// Todo: if last col is partly shown, we failed to adjust col view to show all of it on scroll

const (
	stateFile = "parcours.yaml"
	stateMode = 0600
)

// Store specifies a backing datastore.
// Todo: rename Get/Set View
// Todo: we now rely on col order from Store, arrange to set
type Store interface {
	// Name returns the name of the data source
	Name() string
	// Load loads a file
	Load(path string, last int) (err error)
	// Follow follows a file
	Follow(ctx context.Context, path string, last int) (err error)
	// Promote promotes a field
	Promote(field string) (err error)
	// SetView sets filter and sorts
	SetView(filter nt.Filter, sorts []nt.Sort) (err error)
	// GetView returns fields and count
	GetView() (fields []nt.Field, count int, err error)
	// GetPage returns a page of log lines
	GetPage(offset, size int) (lines []nt.Line, err error)
	// GetLine returns field data for a log line
	GetLine(id string) (data map[string]any, err error)
	// Tail streams log lines
	Tail(ctx context.Context) (lines <-chan nt.Line, err error)
}

// State holds persisted state.
type State struct {
	LastFile string `yaml:"last_file"`
}

// Config holds configuration.
type Config struct {
	WorkDir string `json:"work_dir" desc:"working directory" default:"."`
}

// Parcours is a log viewer.
type Parcours struct {
	workDir string
	store   Store
	logger  nt.Logger
}

// New creates a Parcours from Config.
func (cfg *Config) New(store Store, lgr nt.Logger) *Parcours {

	return &Parcours{
		workDir: cfg.WorkDir,
		store:   store,
		logger:  lgr,
	}
}

// Run runs the app.
func (p *Parcours) Run(ctx context.Context) (err error) {

	state := State{}
	path := filepath.Join(p.workDir, stateFile)
	err = util.Load(&state, path)
	if errors.Is(err, os.ErrNotExist) {
		p.logger.Info(ctx, "no saved state", "path", path)
	} else if err != nil {
		return
	}

	model, err := NewModel(ctx, p.store, p.logger, state.LastFile)
	if err != nil {
		return
	}

	model, err = tea.NewProgram(model).Run()
	if err != nil {
		return
	}

	state.LastFile = model.(Model).lastFile

	err = util.Save(state, path, stateMode)
	return err
}

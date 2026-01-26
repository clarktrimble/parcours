package parcours

import (
	"context"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	nt "parcours/entity"
	"parcours/util"
)

const (
	stateFile = "parcours.yml"
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

// Todo: failed to create table: Catalog Error: Table with name "logs" already exists! -- on new file
// Todo: if last col is partly shown, we failed to adjust col view to show all of it on scroll

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

type State struct {
	LastFile string `yaml:"last_file"`
}

// Todo: move to util
func workDir(dir string) string {
	if dir == "" {
		dir, _ = os.Getwd()
	}
	return dir
}

type Config struct {
	workDir string
}

type Parcours struct {
	state   State
	workDir string
	store   Store
	logger  nt.Logger
}

func (cfg *Config) New(ctx context.Context, store Store, lgr nt.Logger) *Parcours {

	dir := workDir(cfg.workDir)
	statePath := filepath.Join(dir, stateFile)
	state := State{}

	err := util.LoadConfig(&state, statePath)
	if err != nil {
		lgr.Info(ctx, "no saved state", "path", statePath)
	}

	return &Parcours{
		state:   state,
		workDir: dir,
		store:   store,
		logger:  lgr,
	}
}

func (p *Parcours) Run(ctx context.Context) error {

	model, err := NewModel(ctx, p.store, p.logger, p.state.LastFile)
	if err != nil {
		return err
	}

	finalModel, err := tea.NewProgram(model).Run()
	if err != nil {
		return err
	}

	p.state.LastFile = finalModel.(Model).lastFile

	err = util.WriteConfig(p.state, filepath.Join(p.workDir, stateFile), 0600)
	return err
}

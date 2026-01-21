package intake

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"

	"parcours/board"
	"parcours/board/piece"
	nt "parcours/entity"
	"parcours/message"
)

// IntakePanel displays a file browser for selecting log files
type IntakePanel struct {
	board tea.Model

	currentPath string
	entries     []os.DirEntry

	// Current selection (derived from board position)
	selectedIdx int

	// Size
	width  int
	height int

	ctx    context.Context
	logger nt.Logger
}

// Todo: consider filtering to common log file extensions (.log, .json, .ndjson)
// Todo: remember current path across useages, maybe by stashing elsewhere via msg
// Todo: remember last opened file and try to reopen, sometimes?
// Todo: remember previously viewed files in this session, longer?
// Todo: multi-file, woah -- mind blown
// Todo: consider sorting options (name, date, size, dirs first)
// Todo: add separator line below panel, fp as well!
// Todo: look for any ignored error conditions
// Todo: fix "failed to create table: Catalog Error: Table with name "logs" already exists!" on re-open

var intakeFiles = []board.File{
	{Header: "Name", Width: 40},
	{Header: "Size", Width: 10},
	{Header: "Modified", Width: 12},
}

func New(ctx context.Context, lgr nt.Logger, size tea.WindowSizeMsg) (IntakePanel, error) {

	// Todo: pass cwd in
	cwd, err := os.Getwd()
	if err != nil {
		return IntakePanel{}, err
	}

	pnl := IntakePanel{
		currentPath: cwd,
		ctx:         ctx,
		logger:      lgr,
		width:       size.Width,
		height:      size.Height,
	}

	err = pnl.readDir()
	if err != nil {
		return pnl, err
	}

	pnl.board, err = pnl.buildBoard()
	return pnl, err
}

func (pnl IntakePanel) Init() tea.Cmd {
	return nil
}

func (pnl IntakePanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		pnl.width = msg.Width
		pnl.height = msg.Height
		return pnl.rebuildBoard()

	case board.PositionMsg:
		pnl.selectedIdx = msg.Rank
		return pnl, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return pnl, func() tea.Msg { return message.CloseMsg{} }
		case "enter":
			return pnl.handleEnter()
		}
	}

	var cmd tea.Cmd
	pnl.board, cmd = pnl.board.Update(msg)
	return pnl, cmd
}

func (pnl IntakePanel) View() tea.View {
	return pnl.board.View()
}

// Todo: factor helpers -- some repetition here, maybe reimagine?

// handleEnter navigates into directories or selects files
func (pnl IntakePanel) handleEnter() (IntakePanel, tea.Cmd) {
	// Account for ".." entry at index 0
	if pnl.selectedIdx == 0 {
		// Go up to parent directory
		parent := filepath.Dir(pnl.currentPath)
		if parent != pnl.currentPath {
			pnl.currentPath = parent
			if err := pnl.readDir(); err != nil {
				return pnl, func() tea.Msg { return err }
			}
			pnl.selectedIdx = 0
			return pnl.rebuildBoard()
		}
		return pnl, nil
	}

	// Adjust for ".." offset
	entryIdx := pnl.selectedIdx - 1
	if entryIdx < 0 || entryIdx >= len(pnl.entries) {
		return pnl, nil
	}

	entry := pnl.entries[entryIdx]
	fullPath := filepath.Join(pnl.currentPath, entry.Name())

	if entry.IsDir() {
		// Navigate into directory
		pnl.currentPath = fullPath
		if err := pnl.readDir(); err != nil {
			return pnl, func() tea.Msg { return err }
		}
		pnl.selectedIdx = 0
		return pnl.rebuildBoard()
	}

	// File selected - emit message
	return pnl, func() tea.Msg {
		return message.FileSelectedMsg{Path: fullPath}
	}
}

func (pnl *IntakePanel) readDir() error {
	entries, err := os.ReadDir(pnl.currentPath)
	if err != nil {
		return err
	}
	pnl.entries = entries
	return nil
}

func (pnl IntakePanel) rebuildBoard() (IntakePanel, tea.Cmd) {
	var err error
	pnl.board, err = pnl.buildBoard()
	if err != nil {
		return pnl, func() tea.Msg { return err }
	}
	return pnl, nil
}

func (pnl IntakePanel) buildBoard() (tea.Model, error) {
	var ranks []board.Rank

	// Add ".." entry for parent navigation
	ranks = append(ranks, board.NewRank([]tea.Model{
		piece.NewLabel(".."),
		piece.NewLabel(""),
		piece.NewLabel(""),
	}))

	// Add directory entries
	for _, entry := range pnl.entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}

		size := "-"
		modified := ""

		if info, err := entry.Info(); err == nil {
			if !entry.IsDir() {
				size = formatSize(info.Size())
			}
			modified = formatTime(info.ModTime())
		}

		ranks = append(ranks, board.NewRank([]tea.Model{
			piece.NewLabel(name),
			piece.NewLabel(size),
			piece.NewLabel(modified),
		}))
	}

	// Clamp selectedIdx
	// Todo: is this what we want?
	if pnl.selectedIdx >= len(ranks) {
		pnl.selectedIdx = len(ranks) - 1
	}

	return board.New(ranks, intakeFiles, pnl.selectedIdx, 0, pnl.width)
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func formatTime(t time.Time) string {
	now := time.Now()
	if t.Year() == now.Year() {
		if t.YearDay() == now.YearDay() {
			return t.Format("15:04")
		}
		return t.Format("Jan 02")
	}
	return t.Format("2006-01-02")
}

package linepanel

import (
	tea "charm.land/bubbletea/v2"
	"github.com/pkg/errors"

	"parcours/board"
	"parcours/board/piece"
)

// buildRankAndFile builds board files and ranks together in one pass
// Todo: iterate over lp.columns (layout order) instead of lp.fields (store order)
// to respect column ordering from layout file
// Orrrrrrrrrrrrrrrrrrr: we depend on order from Store and can "fix" there??
func (lp LinePanel) buildRankAndFile() (files board.Files, ranks board.Ranks, err error) {
	ranks = make(board.Ranks, len(lp.lines))

	for i, field := range lp.fields {
		col, exists := lp.colMap[field.Name]
		if !exists || col.Hidden || col.Demote {
			continue
		}
		files = append(files, board.File{Header: field.Name, Width: col.Width, SrcIdx: i})
		formatter := getFormatter(col.Format, col.Width)

		for lineIdx, line := range lp.lines {
			if i >= len(line.Values) {
				err = errors.Errorf("field %s out of range for line %d", field.Name, lineIdx)
				return
			}
			ranks[lineIdx].Append(piece.NewLabel(formatter(line.Values[i])))
		}
	}

	return
}

// applyPage builds rank and file from current lines and updates the board
func (lp LinePanel) applyPage() (LinePanel, tea.Cmd) {
	// Build files (columns) and ranks (rows) together from current lines
	files, ranks, err := lp.buildRankAndFile()
	if err != nil {
		return lp, func() tea.Msg { return err }
	}

	// Column structure unchanged (just scrolled) - replace ranks only,
	// preserving board state like cursor position
	if lp.files.Equal(files) {
		var cmd tea.Cmd
		lp.board, cmd = lp.board.Update(board.ReplaceMsg{Ranks: ranks})
		return lp, cmd
	}

	// Column structure changed (initial load, columns added/removed/resized),
	// rebuild the entire board
	lp.files = files
	// Position cursor at bottom if we were scrolling down, else top
	startRank := 0
	if lp.scrollingDown {
		startRank = len(ranks) - 1
	}
	lp.board, err = board.New(ranks, files, startRank, 0, lp.width)
	if err != nil {
		return lp, func() tea.Msg { return err }
	}
	return lp, nil
}

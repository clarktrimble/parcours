package linepanel

import (
	"parcours/board"
	"parcours/board/piece"

	tea "charm.land/bubbletea/v2"
)

// buildFiles builds board files from column config
func (lp LinePanel) buildFiles() []board.File {
	var files []board.File
	for i, field := range lp.fields {
		col, exists := lp.colMap[field.Name]
		if !exists || col.Hidden || col.Demote {
			continue
		}
		files = append(files, board.File{Name: field.Name, Width: col.Width, SrcIdx: i})
	}
	return files
}

// buildRanks converts current lines into board Ranks
func (lp LinePanel) buildRanks(files []board.File) []board.Rank {
	ranks := make([]board.Rank, 0, len(lp.lines))
	for _, line := range lp.lines {
		pieces := make([]tea.Model, 0, len(files))
		for _, f := range files {
			if f.SrcIdx >= len(line.Values) {
				continue
			}
			field := lp.fields[f.SrcIdx]
			col := lp.colMap[field.Name]
			formatter := makeFormatter(field.Type, col.Format)
			pieces = append(pieces, piece.NewValue(line.Values[f.SrcIdx], formatter))
		}
		ranks = append(ranks, board.NewRank(pieces))
	}
	return ranks
}

// buildBoard converts current lines and columns into a Board
func (lp LinePanel) buildBoard() (board.Board, []board.File, error) {
	files := lp.buildFiles()
	ranks := lp.buildRanks(files)

	// Position board based on last navigation direction
	startRank := 0
	if lp.scrollingDown {
		startRank = len(ranks) - 1
	}

	brd, err := board.New(ranks, files, startRank, 0, lp.width, lp.height)
	return brd, files, err
}

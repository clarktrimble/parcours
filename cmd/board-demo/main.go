package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"parcours/board"
	"parcours/board/piece"
)

type model struct {
	//board board.Board
	board tea.Model
}

func initialModel() model {
	// Build files (column headers)
	files := []board.File{
		board.NewFile(piece.NewLabel("Name")),
		board.NewFile(piece.NewLabel("Active")),
		board.NewFile(piece.NewLabel("Level")),
		board.NewFile(piece.NewLabel("Score")),
	}

	// Create multiple rows to test navigation (g/G, pgup/pgdown)
	var ranks []board.Rank

	// Header row
	ranks = append(ranks, board.NewRank([]board.Piece{
		piece.NewLabel("Name"),
		piece.NewLabel("Active"),
		piece.NewLabel("Difficulty"),
		piece.NewLabel("Points"),
	}))

	// Data rows - make enough to test page navigation
	names := []string{"Alice", "Bob", "Carol", "Dave", "Eve", "Frank", "Grace", "Hank", "Ivy", "Jack",
		"Karen", "Leo", "Maya", "Noah", "Olivia", "Paul", "Quinn", "Ruby", "Sam", "Tina"}

	for i, name := range names {
		ranks = append(ranks, board.NewRank([]board.Piece{
			piece.NewTextInput(name, 20),
			piece.NewCheckbox(i%2 == 0),
			piece.NewOperator([]string{"Easy", "Medium", "Hard"}, i%3),
			piece.NewLabel(fmt.Sprintf("%d", (i+1)*10)),
		}))
	}

	// Action row at bottom
	ranks = append(ranks, board.NewRank([]board.Piece{
		piece.NewButton("Submit", "enter"),
		piece.NewButton("Cancel", "esc"),
		piece.NewLabel("---"),
		piece.NewLabel("Total"),
	}))

	brd, err := board.New(ranks, files, 0, 0)
	if err != nil {
		panic(err)
	}

	return model{board: brd}
}

func (m model) Init() tea.Cmd {
	return m.board.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	// Pass all messages to the board
	var cmd tea.Cmd
	m.board, cmd = m.board.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	return m.board.View()
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

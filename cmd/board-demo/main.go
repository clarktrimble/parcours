package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"parcours/board"
	"parcours/board/cell"
)

type model struct {
	//board board.Board
	board tea.Model
}

func initialModel() model {
	// Create some demo cells
	row1 := []tea.Model{
		cell.NewLabel("Name"),
		cell.NewLabel("Status"),
		cell.NewLabel("Count"),
	}

	row2 := []tea.Model{
		cell.NewTextInput("Alice", 20),
		cell.NewCheckbox(true),
		cell.NewLabel("42"),
	}

	row3 := []tea.Model{
		cell.NewTextInput("Bob", 20),
		cell.NewCheckbox(false),
		cell.NewLabel("17"),
	}

	row4 := []tea.Model{
		cell.NewButton("Submit", "enter"),
		cell.NewOperator([]string{"Easy", "Medium", "Hard"}, 1),
		cell.NewLabel("99"),
	}

	// Build ranks
	ranks := []board.Rank{
		board.NewRank(row1),
		board.NewRank(row2),
		board.NewRank(row3),
		board.NewRank(row4),
	}

	// Build files (column headers)
	files := []board.File{
		board.NewFile(cell.NewLabel("Column A")),
		board.NewFile(cell.NewLabel("Column B")),
		board.NewFile(cell.NewLabel("Column C")),
	}

	brd, err := board.New(ranks, files)
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

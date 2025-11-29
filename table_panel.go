package parcours

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/pkg/errors"
)

// Todo: handle cell overflow
// Todo: cell selection
// Todo: handle columns overflow
// Todo: search
// Todo: extend last column to edge of panel

const (
	headerHeight = 2
)

// TablePanel handles the table view display and navigation state
type TablePanel struct {
	Selected int // Absolute position (0 to TotalLines-1) of selected line
	Offset   int // Offset of page shown
	Total    int // Total log lines after filtering

	Width   int
	Height  int
	Focused bool

	columns []Column
	table   *table.Table
}

// pageSize returns the number of rows that fit on screen
func (pnl TablePanel) pageSize() int {
	return pnl.Height - headerHeight
}

// selectedLine returns the index of the currently selected line
func (pnl TablePanel) selectedLine() int {
	return pnl.Selected - pnl.Offset
}

func NewTablePanel(columns []Column, fields []Field, count int) TablePanel {

	lgt := table.New()
	styleTable(lgt)

	tablePane := TablePanel{
		Focused: true, // Todo: elsewhere
		table:   lgt,
		Total:   count,
	}

	tablePane = tablePane.SetColumns(columns, fields)

	return tablePane
}

func (pnl TablePanel) SetColumns(columns []Column, fields []Field) TablePanel {

	pnl.columns = columns

	// Build field index
	idxByName := map[string]int{}
	for i, field := range fields {
		idxByName[field.Name] = i
	}

	// Resolve each column against fields
	for i := range columns {
		col := &columns[i]

		idx := idxByName[col.Field]
		field := fields[idx]

		col.fieldIdx = idx
		col.formatter = makeFormatter(field.Type, col.Format)
	}

	// Set headers (padded to width+1 for spacing)
	var headers []string
	for _, col := range pnl.columns {
		if col.Hidden || col.Demote {
			continue
		}
		padded := fmt.Sprintf("%-*s", col.Width+1, col.Field)
		headers = append(headers, padded)
	}
	pnl.table.Headers(headers...)

	return pnl
}

func (pnl TablePanel) Update(msg tea.Msg) (TablePanel, tea.Cmd) {
	switch msg := msg.(type) {

	case resetMsg:
		pnl.Selected = 0
		pnl.Offset = 0
		return pnl, nil

	case pageMsg:
		pnl.Total = msg.count
		return pnl, nil

	case tea.KeyPressMsg:

		if !pnl.Focused {
			return pnl, nil
		}

		pageSize := pnl.pageSize()

		switch msg.String() {
		case "up", "k":
			if pnl.Selected > 0 {
				pnl.Selected--
			}

		case "down", "j":
			if pnl.Selected < pnl.Total-1 {
				pnl.Selected++
			}

		case "pgup", "ctrl+u":
			pnl.Selected -= pageSize
			if pnl.Selected < 0 {
				pnl.Selected = 0
			}

		case "pgdown", "ctrl+d":
			pnl.Selected += pageSize
			if pnl.Selected >= pnl.Total {
				pnl.Selected = pnl.Total - 1
			}

		case "g":
			pnl.Selected = 0

		case "G":
			pnl.Selected = pnl.Total - 1
		}

		// Adjust ScrollOffset to keep SelectedLine visible
		oldScrollOffset := pnl.Offset
		if pnl.Selected < pnl.Offset {
			pnl.Offset = pnl.Selected
		} else if pnl.Selected >= pnl.Offset+pageSize {
			pnl.Offset = pnl.Selected - pageSize + 1
		}

		// If we've scrolled to a different page, request new data
		if pnl.Offset != oldScrollOffset {
			return pnl, func() tea.Msg {
				return getPageMsg{
					offset: pnl.Offset,
					size:   pageSize,
				}
			}
		}

	case panelSizeMsg:
		pnl.Width = msg.width
		pnl.Height = msg.height

		// Request data with new page size
		pageSize := pnl.pageSize()
		if pageSize > 0 {
			return pnl, func() tea.Msg {
				return getPageMsg{
					offset: pnl.Offset,
					size:   pageSize,
				}
			}
		}
	}

	return pnl, nil
}

// SelectedId returns the id of the currently selected line
func (pnl TablePanel) SelectedId(lines []Line) (id string, err error) {
	selected := pnl.selectedLine()

	if len(lines) == 0 || selected >= len(lines) {
		err = errors.Errorf("index %d is out of bounds of %d lines", selected, len(lines))
		return
	}

	id = lines[selected][0].String() //Todo: add Id() method to Line?
	return
}

// Render renders the table with the given data
func (pnl TablePanel) Render(lines []Line) string {

	// style selected row
	selected := pnl.selectedLine() // Todo: neede for closure?
	pnl.table.StyleFunc(func(row, col int) lipgloss.Style {
		if row == selected {
			return hlRowStyle
		}
		return unStyle
	})

	// repopulate table
	pnl.table.ClearRows()
	for _, line := range lines {
		var row []string
		for _, col := range pnl.columns {
			if col.Hidden || col.Demote {
				continue
			}

			formatted := col.formatter(line[col.fieldIdx]) // Todo: dont crash
			row = append(row, truncate(formatted, col.Width))
		}
		pnl.table.Row(row...)
	}

	return pnl.table.Render()
}

// help

func makeFormatter(fieldType, format string) func(Value) string {
	if format != "" && fieldType == "TIMESTAMP" {
		return func(v Value) string {
			t, err := v.Time()
			if err == nil {
				return t.Format(format)
			}
			return v.String()
		}
	}

	return func(v Value) string {
		return v.String()
	}
}

func truncate(in string, width int) string {

	if len(in) <= width {
		return in
	}

	truncated := in[:width-1]
	ellipsis := mutedStyle.Render("…")
	return truncated + ellipsis
}

func styleTable(tbl *table.Table) {

	tbl.Border(lipgloss.Border{
		Top:         "─", // Horizontal parts of separator
		Middle:      "─", // Between columns in separator
		MiddleLeft:  "─", // Left edge of separator
		MiddleRight: "─", // Right edge of separator
	}).
		BorderTop(false).    // Disable top border
		BorderBottom(false). // Disable bottom border
		BorderLeft(false).   // Disable left border
		BorderRight(false).  // Disable right border
		BorderColumn(false). // Disable column separators
		BorderStyle(tableBorderStyle)

}

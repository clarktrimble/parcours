package table

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/pkg/errors"

	nt "parcours/entity"
	"parcours/message"
	"parcours/style"
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
	// Todo: privatize
	Selected int // Absolute position (0 to TotalLines-1) of selected line
	Offset   int // Offset of page shown
	Total    int // Total log lines after filtering

	Width   int
	height  int
	Focused bool

	colFmts []colFmt
	lines   []nt.Line
	table   *table.Table
}

func NewTablePanel(columns []nt.Column, fields []nt.Field, count int) TablePanel {

	lgt := table.New()
	styleTable(lgt)

	tablePanel := TablePanel{
		Focused: true, // Todo: elsewhere
		table:   lgt,
		Total:   count,
	}

	tablePanel = tablePanel.setColumns(columns, fields)

	return tablePanel
}

type colFmt struct {
	lineIdx   int
	width     int
	fieldName string
	formatter func(nt.Value) string
}

func (pnl TablePanel) Update(msg tea.Msg) (TablePanel, tea.Cmd) {
	switch msg := msg.(type) {

	case SizeMsg:
		pnl.Width = msg.Width
		pnl.height = msg.Height

		pageSize := pnl.PageSize() // Todo: fur closure??
		if pageSize > 0 {
			return pnl, func() tea.Msg {
				return message.GetPageMsg{
					Offset: pnl.Offset,
					Size:   pageSize,
				}
			}
		}

	case ColumnsMsg:
		pnl = pnl.setColumns(msg.Columns, msg.Fields)
		return pnl, func() tea.Msg {
			return message.GetPageMsg{
				Offset: pnl.Offset,
				Size:   pnl.PageSize(),
			}
		}

	case PageMsg:
		pnl.lines = msg.Lines
		pnl.Total = msg.Count
		return pnl, nil

	case ResetMsg:
		pnl.Selected = 0
		pnl.Offset = 0
		return pnl, nil

	case tea.KeyPressMsg:
		pageSize := pnl.PageSize()

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
				return message.GetPageMsg{
					Offset: pnl.Offset,
					Size:   pageSize,
				}
			}
		}
	}

	return pnl, nil
}

// Render renders the table with the given data
func (pnl TablePanel) View() tea.View {

	// style selected row
	selected := pnl.selectedLine() // Todo: neede for closure?
	pnl.table.StyleFunc(func(row, col int) lipgloss.Style {
		if row == selected {
			return style.HlRowStyle
		}
		return style.UnStyle
	})

	// repopulate table
	pnl.table.ClearRows()
	for _, line := range pnl.lines {
		row := pnl.row(line)
		pnl.table.Row(row...)
	}

	//return pnl.table.Render()
	return tea.NewView(pnl.table)
}

// SelectedId returns the id of the currently selected line
func (pnl TablePanel) SelectedId() (id string, err error) {

	selected := pnl.selectedLine()
	ln := len(pnl.lines)

	if ln == 0 || selected >= ln {
		err = errors.Errorf("index %d is out of bounds of %d lines", selected, ln)
		return
	}

	id = pnl.lines[selected][0].String() //Todo: add Id() method to Line?
	return
}

// PageSize returns the number of rows that fit on panel
func (pnl TablePanel) PageSize() int {
	return pnl.height - headerHeight
}

// unexported

func (pnl TablePanel) selectedLine() int {
	return pnl.Selected - pnl.Offset
}

func (pnl TablePanel) row(line nt.Line) []string {
	row := make([]string, len(pnl.colFmts))
	for i, colFmt := range pnl.colFmts {
		formatted := colFmt.formatter(line[colFmt.lineIdx]) // Todo: dont crash
		row[i] = truncate(formatted, colFmt.width)
	}
	return row
}

func (pnl TablePanel) setColumns(columns []nt.Column, fields []nt.Field) TablePanel {

	// colFmts tracks order and format of columns to be shown
	colFmts := []colFmt{}

	idxByName := map[string]int{}
	for i, field := range fields {
		idxByName[field.Name] = i
	}

	for _, col := range columns {
		if col.Hidden || col.Demote {
			continue
		}

		idx := idxByName[col.Field]
		field := fields[idx] // Todo: dont crash

		colFmts = append(colFmts, colFmt{
			lineIdx:   idx,
			width:     col.Width,
			fieldName: col.Field,
			formatter: makeFormatter(field.Type, col.Format),
		})
	}

	var headers []string
	for _, colFmt := range colFmts {
		padded := fmt.Sprintf("%-*s", colFmt.width+1, colFmt.fieldName)
		headers = append(headers, padded)
	}

	pnl.table.Headers(headers...)
	pnl.colFmts = colFmts
	pnl.lines = nil // lines we had no longer match colFmts

	return pnl
}

// help

func makeFormatter(fieldType, format string) func(nt.Value) string {
	if format != "" && fieldType == "TIMESTAMP" {
		return func(val nt.Value) string {
			t, err := val.Time()
			if err == nil {
				return t.Format(format)
			}
			return val.String()
		}
	}

	return func(v nt.Value) string {
		return v.String()
	}
}

func truncate(in string, width int) string {

	if len(in) <= width {
		return in
	}

	truncated := in[:width-1]
	ellipsis := style.MutedStyle.Render("…")
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
		BorderStyle(style.TableBorderStyle)

}

package table

import (
	"context"
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
	selected int // Absolute position (0 to TotalLines-1) of selected line
	offset   int // Offset of page shown
	total    int // Total log lines after filtering

	width  int
	height int

	colFmts []colFmt
	lines   []nt.Line
	table   *table.Table

	ctx    context.Context
	logger nt.Logger
}

func NewTablePanel(ctx context.Context, columns []nt.Column, fields []nt.Field, count int, lgr nt.Logger) (pnl TablePanel, err error) {

	lgt := table.New()
	styleTable(lgt)

	pnl = TablePanel{
		table:  lgt,
		total:  count,
		ctx:    ctx,
		logger: lgr,
	}

	pnl, err = pnl.setColumns(columns, fields)
	return
}

func (pnl TablePanel) Init() tea.Cmd {
	return nil
}

func (pnl TablePanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case SizeMsg:
		pnl.width = msg.Width
		pnl.height = msg.Height

		pageSize := pnl.PageSize()
		if pageSize > 0 {
			return pnl, func() tea.Msg {
				return message.GetPageMsg{
					Offset: pnl.offset,
					Size:   pageSize,
				}
			}
		}

	case ColumnsMsg:
		var err error
		pnl, err = pnl.setColumns(msg.Columns, msg.Fields)
		if err != nil {
			return pnl, message.ErrorCmd(err)
		}
		return pnl, func() tea.Msg {
			return message.GetPageMsg{
				Offset: pnl.offset,
				Size:   pnl.PageSize(),
			}
		}

	case PageMsg:
		pnl.lines = msg.Lines
		pnl.total = msg.Count
		pnl.populate()

		return pnl, pnl.selectedCmd()

	case ResetMsg:
		pnl.selected = 0
		pnl.offset = 0
		return pnl, func() tea.Msg {
			return message.GetPageMsg{
				Offset: pnl.offset,
				Size:   pnl.PageSize(),
			}
		}

	case tea.KeyPressMsg:
		pageSize := pnl.PageSize()
		selected := handleNavKey(msg.String(), pnl.selected, pnl.total, pageSize)
		offset := adjustOffset(selected, pnl.offset, pageSize)

		// If we've scrolled to a different page, request new data
		if pnl.offset != offset {
			pnl.offset = offset
			pnl.selected = selected
			return pnl, func() tea.Msg {
				return message.GetPageMsg{
					Offset: pnl.offset,
					Size:   pageSize,
				}
			}
		}

		// Selection changed, tell the world
		if pnl.selected != selected {
			pnl.selected = selected
			return pnl, pnl.selectedCmd()
		}
	}

	return pnl, nil
}

func (pnl TablePanel) View() tea.View {
	pnl.table.StyleFunc(style.RowStyler(pnl.selectedLocal()))
	return tea.NewView(pnl.table)
}

// PageSize returns the number of rows that fit on panel
func (pnl TablePanel) PageSize() int {
	return pnl.height - headerHeight
}

// unexported

type colFmt struct {
	lineIdx   int
	width     int
	fieldName string
	formatter func(nt.Value) string
}

func (pnl TablePanel) populate() {
	pnl.table.ClearRows()
	for _, line := range pnl.lines {
		row := pnl.row(line)
		pnl.table.Row(row...)
	}
}

func (pnl TablePanel) selectedLocal() int {
	return pnl.selected - pnl.offset
}

func handleNavKey(key string, selected, total, pageSize int) int {
	switch key {
	case "up", "k":
		if selected > 0 {
			selected--
		}

	case "down", "j":
		if selected < total-1 {
			selected++
		}

	case "pgup", "ctrl+u":
		selected -= pageSize
		if selected < 0 {
			selected = 0
		}

	case "pgdown", "ctrl+d":
		selected += pageSize
		if selected >= total {
			selected = total - 1
		}

	case "g":
		selected = 0

	case "G":
		selected = total - 1
	}

	return selected
}

func adjustOffset(selected, offset, pageSize int) int {
	if selected < offset {
		return selected
	} else if selected >= offset+pageSize {
		return selected - pageSize + 1
	}
	return offset
}

func (pnl TablePanel) row(line nt.Line) []string {
	row := make([]string, len(pnl.colFmts))
	for i, colFmt := range pnl.colFmts {
		formatted := colFmt.formatter(line.Values[colFmt.lineIdx]) // Todo: dont crash
		row[i] = truncate(formatted, colFmt.width)
	}
	return row
}

func (pnl TablePanel) setColumns(columns []nt.Column, fields []nt.Field) (TablePanel, error) {

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

		idx, ok := idxByName[col.Field]
		if !ok {
			return pnl, errors.Errorf("column %q not found in fields", col.Field)
		}
		if idx < 0 || idx >= len(fields) {
			return pnl, errors.Errorf("column %q has invalid index %d", col.Field, idx)
		}

		field := fields[idx]

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
	pnl.populate()

	return pnl, nil
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

package edible

// EditableRow represents a row in an editable table
type EditableRow interface {
	// NumColumns returns the number of columns in this row
	NumColumns() int

	// GetCell returns the string value of a cell
	GetCell(col int) string

	// SetCell updates the value of a cell
	SetCell(col int, value string)

	// Render returns the rendered string for this row
	// selectedCol indicates which column is currently selected (-1 if none)
	Render(selectedCol int) string
}

// EditTable manages a list of editable rows with navigation and selection
type EditTable struct {
	rows        []EditableRow
	selectedRow int // Currently selected row index
	selectedCol int // Currently selected column index
}

// NewEditTable creates a new editable table
func NewEditTable(rows []EditableRow) EditTable {
	return EditTable{
		rows:        rows,
		selectedRow: 0,
		selectedCol: 0,
	}
}

// SelectedRow returns the currently selected row index
func (t EditTable) SelectedRow() int {
	return t.selectedRow
}

// SelectedCol returns the currently selected column index
func (t EditTable) SelectedCol() int {
	return t.selectedCol
}

// NumRows returns the total number of rows
func (t EditTable) NumRows() int {
	return len(t.rows)
}

// GetRow returns the row at the given index
func (t EditTable) GetRow(idx int) EditableRow {
	if idx < 0 || idx >= len(t.rows) {
		return nil
	}
	return t.rows[idx]
}

// SelectedCell returns the currently selected row and column
func (t EditTable) SelectedCell() (row EditableRow, col int, ok bool) {
	if t.selectedRow < 0 || t.selectedRow >= len(t.rows) {
		return nil, -1, false
	}
	return t.rows[t.selectedRow], t.selectedCol, true
}

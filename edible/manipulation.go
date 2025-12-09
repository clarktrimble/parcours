package edible

// AddRow appends a new row to the table
func (t EditTable) AddRow(row EditableRow) EditTable {
	t.rows = append(t.rows, row)
	return t
}

// InsertRow inserts a row at the specified index
func (t EditTable) InsertRow(idx int, row EditableRow) EditTable {
	if idx < 0 {
		idx = 0
	}
	if idx >= len(t.rows) {
		return t.AddRow(row)
	}

	t.rows = append(t.rows[:idx+1], t.rows[idx:]...)
	t.rows[idx] = row
	return t
}

// DeleteRow removes the row at the specified index
func (t EditTable) DeleteRow(idx int) EditTable {
	if idx < 0 || idx >= len(t.rows) {
		return t
	}

	t.rows = append(t.rows[:idx], t.rows[idx+1:]...)

	// Adjust selection if needed
	if t.selectedRow >= len(t.rows) && len(t.rows) > 0 {
		t.selectedRow = len(t.rows) - 1
	}
	if len(t.rows) == 0 {
		t.selectedRow = -1
		t.selectedCol = -1
	}

	return t
}

// DeleteSelectedRow removes the currently selected row
func (t EditTable) DeleteSelectedRow() EditTable {
	return t.DeleteRow(t.selectedRow)
}

// SetRows replaces all rows
func (t EditTable) SetRows(rows []EditableRow) EditTable {
	t.rows = rows
	// Reset selection if out of bounds
	if t.selectedRow >= len(rows) {
		if len(rows) > 0 {
			t.selectedRow = len(rows) - 1
		} else {
			t.selectedRow = -1
			t.selectedCol = -1
		}
	}
	return t
}

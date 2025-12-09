package edible

// MoveUp moves selection up one row
func (t EditTable) MoveUp() EditTable {
	if t.selectedRow > 0 {
		t.selectedRow--
	}
	return t
}

// MoveDown moves selection down one row
func (t EditTable) MoveDown() EditTable {
	if t.selectedRow < len(t.rows)-1 {
		t.selectedRow++
	}
	return t
}

// MoveLeft moves selection left one column
func (t EditTable) MoveLeft() EditTable {
	if t.selectedRow < 0 || t.selectedRow >= len(t.rows) {
		return t
	}
	if t.selectedCol > 0 {
		t.selectedCol--
	}
	return t
}

// MoveRight moves selection right one column
func (t EditTable) MoveRight() EditTable {
	if t.selectedRow < 0 || t.selectedRow >= len(t.rows) {
		return t
	}
	row := t.rows[t.selectedRow]
	if t.selectedCol < row.NumColumns()-1 {
		t.selectedCol++
	}
	return t
}

// NextField cycles to the next field (wrapping to next row if needed)
func (t EditTable) NextField() EditTable {
	if t.selectedRow < 0 || t.selectedRow >= len(t.rows) {
		return t
	}

	row := t.rows[t.selectedRow]
	if t.selectedCol < row.NumColumns()-1 {
		// Move to next column in same row
		t.selectedCol++
	} else {
		// Wrap to first column of next row (or same row if last)
		t.selectedCol = 0
		if t.selectedRow < len(t.rows)-1 {
			t.selectedRow++
		}
	}
	return t
}

// PrevField cycles to the previous field (wrapping to previous row if needed)
func (t EditTable) PrevField() EditTable {
	if t.selectedRow < 0 || t.selectedRow >= len(t.rows) {
		return t
	}

	if t.selectedCol > 0 {
		// Move to previous column in same row
		t.selectedCol--
	} else {
		// Wrap to last column of previous row (or same row if first)
		if t.selectedRow > 0 {
			t.selectedRow--
			row := t.rows[t.selectedRow]
			t.selectedCol = row.NumColumns() - 1
		}
	}
	return t
}

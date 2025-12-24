# Navigation Improvements - Next Steps

## What We Fixed for PageDown

### The Problem
- PageDown would jump cursor to bottom after loading new data
- At end of dataset, PageDown did nothing
- Replace() would fail due to dimension mismatches (varying line counts)

### The Solution
1. **Board.Replace()** - Preserves cursor position when updating data with same dimensions
2. **Removed cursor movement from movePageDown()** - Board no longer moves cursor before sending NavMsg
3. **Always request full pages** - LinesPanel adjusts offset to ensure `offset + pageSize <= total`
4. **MoveToMsg at end** - When already at last page, send `MoveToMsg{MoveTo: Bottom}` to position cursor

### How It Works Now
- PageDown preserves cursor position (e.g., rank 5 stays rank 5) with new data underneath
- When at end of dataset, cursor moves to bottom line
- Replace() succeeds because we always get full pageSize of data

## Remaining Work

### PageUp
**Current behavior:** Needs investigation
**Desired behavior:** Mirror of PageDown
- Preserve cursor position when paging up
- At top of dataset (offset=0), move cursor to top (rank 0)
- Should use Replace() same as PageDown

**Implementation notes:**
- Check if movePageUp() needs same treatment as movePageDown()
- Likely needs MoveToMsg{MoveTo: Top} when already at beginning
- May need offset adjustment like PageDown to ensure full pages

### Top (g key)
**Current behavior:** moveTop() sets rank=0 and sends NavMsg{NavTop}
**Issues to check:**
- Does it properly go to absolute top of dataset?
- NavTop handler calculates offset=0, should work correctly
- Cursor positioning at top after data loads?

**Implementation notes:**
- LinesPanel NavTop sets offset=0 and requests page
- Should send MoveToMsg{MoveTo: Top} or rely on buildBoard positioning?
- Test if cursor ends up at top after reload

### Bottom (G key)
**Current behavior:** moveBottom() sets rank=height-1 and sends NavMsg{NavBottom}
**Issues to check:**
- NavBottom calculates `((total-1)/pageSize)*pageSize` for last page offset
- Does cursor end up at bottom line of dataset?
- With our full-page guarantee, should work correctly now

**Implementation notes:**
- Should send MoveToMsg{MoveTo: Bottom} after loading last page
- Or does scrollingDown=true in NavBottom handler position correctly?
- Test if cursor ends up at actual last line of data

## Key Principles Established

1. **Board doesn't move cursor before requesting data** (for Page navigation)
   - Lets Replace() preserve position naturally
   - Only move cursor when explicitly positioning (MoveToMsg)

2. **Always request full pages**
   - Prevents Replace() dimension mismatches
   - No blank lines at end of dataset

3. **Use MoveToMsg for explicit positioning**
   - When we know exactly where cursor should be
   - Board handles it cleanly without coupling to data state

4. **LinesPanel knows dataset boundaries, Board knows cursor**
   - LinesPanel decides when to send MoveToMsg
   - Board just positions based on message

## Testing Checklist

- [ ] PageUp from various positions preserves cursor
- [ ] PageUp at top (offset=0) moves cursor to rank 0
- [ ] Top (g) goes to first line of dataset, cursor at rank 0
- [ ] Bottom (G) goes to last line of dataset, cursor at last rank
- [ ] All navigation works correctly with Replace() (no rebuilds except dimension changes)
- [ ] scrollingDown behavior - may be obsolete now?

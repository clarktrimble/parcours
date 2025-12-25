# Board Reusability - Session Notes

## What We Did

### Horizontal Scrolling (Board)
- Added `viewportWidth` and `fileOffset` to Board
- Added `SizeMsg` handling
- Ported `visibleFiles()` and `adjustFileOffset()` from TablePanel
- Updated `moveLeft()`/`moveRight()` to adjust fileOffset
- Updated `View()` to render only visible files
- LinePanel forwards SizeMsg to Board

### Piece Messages
- Added `board.PieceMsg` interface with `IsPieceMsg()` and `SetPosition(rank, file int)`
- Created `piece.CheckedMsg`, `piece.OperatorChangedMsg`, `piece.ValueChangedMsg`
- Pieces send messages when state changes (Checkbox on toggle, Operator on cycle, TextInput on value change)
- Board wraps piece cmds to inject position via `SetPosition()`
- `Square.position` is set in `board.New()` and `Replace()` via `setSquarePositions()`

### FilterPanelToo
- New filter panel using Board instead of manual rendering
- Holds `board.Board` and `filters []nt.Filter`
- Builds Board with Checkbox, Label, Operator, TextInput pieces per row
- Handles `piece.*Msg` to update `filters` slice
- On "p" applies the filter
- Model routes `board.PieceMsg` to active panel

## Known Issues / TODOs

### Edit Mode (Blocker for TextInput)
- TextInput handles left/right for cursor movement
- Board handles left/right for file navigation
- They conflict - currently can't edit text values
- Need: focus/edit mode where keys go to piece instead of Board nav
- Ideas: Enter to toggle edit mode, or piece signals it wants keys

### Delete Row
- Back-burnered
- Not a piece operation - removes entire rank
- Options: special key FilterPanelToo intercepts, or a delete button piece

### Type Assertion in LinePanel
- `linepanel.go:310` has `brd = sized.(board.Board) // Todo: unfuck`
- bt/elm pattern: Update() returns tea.Model, we assert back to concrete type
- This is the cost of nested models in bubbletea - live with it

### Board.Replace Error Signaling
- `linepanel.go:96-100` uses `cmd != nil` to detect Replace failure
- Fragile - should use explicit error signal instead

## bt/elm Patterns Learned

1. Messages flow DOWN (runtime -> Model -> child -> grandchild)
2. Commands flow UP (returned from Update, collected by runtime)
3. No "bubbling" - that's not a thing
4. Pieces own state, send messages on change, parent tracks via messages
5. For routing: marker interfaces (like `PieceMsg`) + type switch in Model
6. Type assertions after Update() are the tax for nested models

## Misc

- `cmd.go:26-37` has `tea.Batch(...)()` - immediate invocation pattern with trailing `()`. Marked with Todo, worth revisiting.
- Operator uses left/right to cycle, which works but might confuse: at first option, left wraps to last rather than moving to previous file. May be fine, may need edit mode to resolve.
- Didn't test: multiple filter rows, scrolling with many filters, empty TextInput value.
- `Square.position` is now in use - can remove the `// Todo: use/lose` comment.

## Next Steps

1. Implement edit mode for TextInput
2. Handle delete row
3. Test thoroughly
4. Consider removing old FilterPanel once FilterPanelToo is solid

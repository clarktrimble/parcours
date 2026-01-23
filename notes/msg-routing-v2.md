# Message Routing

## Definition

Messages are defined in the pkg that handle them.
This is a little arbitrary, but some rule to follow seems to simplify.

Messages handled by `model` are defined in the `message` pkg which acts as a sort of bt/elm entity.

## Messages

| Message | Description |
|---------|-------------|
| `detail.ColumnsMsg` | delivers column config |
| `detail.LineMsg` | delivers full line data |
| `linepanel.ColumnsMsg` | delivers column config |
| `linepanel.PageMsg` | delivers page data |
| `linepanel.ResetMsg` | reset to initial state |
| `message.CloseMsg` | panel wants to close |
| `message.CountMsg` | track total count |
| `message.FileSelectedMsg` | file selected for loading |
| `message.GetPageMsg` | fetch page from Store |
| `message.OpenDetailMsg` | open detail view for a line |
| `message.OpenFilterMsg` | open filter panel |
| `message.OpenIntakeMsg` | open intake panel |
| `message.ReloadColumnsMsg` | reload columns from layout file |
| `message.SelectedMsg` | track selected row |
| `message.SetFilterMsg` | apply filter |

## Stack

Messages are routed to the top of model's stack, excepting:
- `WindowSizeMsg` is broadcast to all children in stack
- `message.*`, `error`, and global keys are handled by model

All this is a bit of a high-wire act.
Way better than previous approaches but consider:
- Stack: simple, positional, modal-friendly, can't address below top
- Tagged: type-safe, one instance per type, compile-time routing (called this wrapped in this repo)
- ID-based: flexible, multiple instances, runtime routing, more boilerplate

In any case, we get a very tidy render in View with stack.

## Keypress Map

| Key | Panel | Message |
|-----|-------|---------|
| `esc` | any | `message.CloseMsg` |
| `c` | linepanel | `message.OpenFilterMsg` (with cell) |
| `f` | linepanel | `message.OpenFilterMsg` (view) |
| `r` | linepanel | `message.ReloadColumnsMsg` |
| `o` | linepanel | `message.OpenIntakeMsg` |
| `enter` | linepanel | `message.OpenDetailMsg` |
| `enter` | intake | `message.FileSelectedMsg` |
| `p` | filterpanel | `message.SetFilterMsg` |
| `delete` | filterpanel | delete selected filter |
| `ctrl+c`, `q` | (global) | quit |

## Board / Piece

Internal messaging for grid system. Panels use board, board uses pieces.

### Board Messages

| Message | Direction | Notes |
|---------|-----------|-------|
| `board.PositionMsg` | board→panel | cursor moved |
| `board.NavMsg` | board→panel | hit boundary |
| `board.MoveToMsg` | panel→board | move cursor |
| `board.ReplaceMsg` | panel→board | replace ranks |
| `board.AppendMsg` | panel→board | add rank |
| `board.RemoveMsg` | panel→board | remove rank |

### Piece Messages

| Message | Emitter | Notes |
|---------|---------|-------|
| `piece.CheckedMsg` | checkbox | toggled |
| `piece.OperatorChangedMsg` | operator | selection changed |
| `piece.ValueChangedMsg` | textinput | text changed |


## Board / Piece

Internal messaging for the grid system. Panels use board, board uses pieces.

### Board Messages

| Message | Emitter | Handler | Notes |
|---------|---------|---------|-------|
| `board.PositionMsg` | board | panel | cursor moved |
| `board.NavMsg` | board | panel | hit boundary, need scroll |
| `board.MoveToMsg` | panel | board | move cursor to top/bottom |
| `board.ReplaceMsg` | panel | board | replace ranks (data changed) |
| `board.AppendMsg` | panel | board | add rank |
| `board.RemoveMsg` | panel | board | remove rank |

### Piece Messages

| Message | Emitter | Handler | Notes |
|---------|---------|---------|-------|
| `piece.PressedMsg` | button | filterpanel | button pressed |
| `piece.CheckedMsg` | checkbox | filterpanel | toggled |
| `piece.OperatorChangedMsg` | operator | filterpanel | selection changed |
| `piece.ValueChangedMsg` | textinput | filterpanel | text changed |

## Keypress Map

| Key | Panel | Message | Action |
|-----|-------|---------|--------|
| `esc` | linepanel | `CloseMsg` | quit |
| `esc` | filterpanel | `CloseMsg` | pop |
| `esc` | detail | `CloseMsg` | pop |
| `esc` | intake | `CloseMsg` | pop or quit |
| `c` | linepanel | `OpenFilterMsg` | push filter with cell data |
| `f` | linepanel | `OpenFilterMsg` | push filter (view existing) |
| `r` | linepanel | `ReloadColumnsMsg` | reload from layout file |
| `o` | linepanel | `OpenIntakeMsg` | push intake |
| `enter` | linepanel | `OpenDetailMsg` | push detail |
| `enter` | intake | `FileSelectedMsg` | load file, replace stack |
| `p` | filterpanel | `SetFilterMsg` | apply filter, pop |
| `delete` | filterpanel | - | delete selected filter |
| `ctrl+c`, `q` | (global) | - | quit |

### Filter Pieces (board/piece)

| Key | Piece | Action |
|-----|-------|--------|
| `t`, `space` | checkbox | toggle checked |
| `tab` | operator | next operator |
| `shift+tab` | operator | prev operator |
| `backspace` | textinput | delete char before cursor |
| `delete` | textinput | delete char at cursor |
| `left` | textinput | move cursor left |
| `right` | textinput | move cursor right |
| `home`, `ctrl+a` | textinput | cursor to start |
| `end`, `ctrl+e` | textinput | cursor to end |
| (chars) | textinput | insert character |



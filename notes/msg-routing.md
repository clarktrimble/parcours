# Message Routing

## Package Rule

```
message = panel→model (requests going up)
panel   = model→panel (data coming down)
```

When adding a message, ask: who handles it?
- Model handles → `message` package
- Panel handles → panel package

## Message Packages

### message (panel→model)

| Message | Emitter | Notes |
|---------|---------|-------|
| `CloseMsg` | any panel | quit or pop based on stack |
| `OpenDetailMsg` | linepanel | push detail |
| `OpenFilterMsg` | linepanel | push filter |
| `OpenIntakeMsg` | linepanel | push intake |
| `ReloadColumnsMsg` | linepanel | reload from layout file |
| `FileSelectedMsg` | intake | load file, replace stack |
| `GetPageMsg` | linepanel | fetch page from Store |
| `SetFilterMsg` | filterpanel | apply filter, pop |
| `SelectedMsg` | linepanel | track selection |
| `CountMsg` | linepanel | track total |

### linepanel (model→linepanel)

| Message | Notes |
|---------|-------|
| `Msg` | wrapper for routing |
| `PageMsg` | delivers page data |
| `ColumnsMsg` | delivers column config |
| `ResetMsg` | resets to initial state |

### detail (model→detail)

| Message | Notes |
|---------|-------|
| `LineMsg` | delivers full line data |
| `ColumnsMsg` | delivers column config |

### filterpanel (model→filterpanel)

| Message | Notes |
|---------|-------|
| `Msg` | wrapper for routing |

### intake (model→intake)

| Message | Notes |
|---------|-------|
| `Msg` | wrapper for routing |

## Patterns

### Panels own their keys

Panel catches key, emits semantic message. Model receives intent, decides action.

```go
// linepanel
case "f":
    return lp, func() tea.Msg { return message.OpenFilterMsg{} }
```

Model only handles global keys (ctrl+c, q). Everything else flows to top of stack.

### Model orchestrates stack and Store

Panels don't push/pop or touch Store directly. They emit messages; model mediates.

### Wrapper Msg routes back to panel

`linepanel.Msg{Wrapped: innerMsg}` tells model "route this to linepanel".
Used when commands produce messages that need to find their panel.

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

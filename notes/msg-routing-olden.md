# Message Routing

## Todos
- Todo: consider user stories to follow msg flows

## Potential Refactor

Consolidate all panel→model messages into `message` package:

```
message (panel→model):
  CloseMsg           - unified, all panels
  OpenDetailMsg      - from linepanel
  OpenFilterMsg      - from linepanel
  OpenIntakeMsg      - from linepanel
  ReloadColumnsMsg   - from linepanel
  FileSelectedMsg    - from intake
  SetFilterMsg       - already here
  GetPageMsg         - already here
  SelectedMsg        - already here
  CountMsg           - already here

panel (model→panel + routing):
  Msg                - wrapper for routing
  PageMsg            - linepanel handles
  ColumnsMsg         - linepanel/detail handle
  ResetMsg           - linepanel handles
  LineMsg            - detail handles
```

Why: `message` package exists to avoid circular deps. If it's the
panel→model bucket, make it consistently so. Panel packages become
the panel's input interface only.

Clear rule: message = panel→model, panel = model→panel

## Patterns We Like

### Messages live with whoever uses them

`linepanel.OpenDetailMsg` lives in linepanel because linepanel emits it.
detail never sees this message - it's constructed by model, then receives `detail.LineMsg`.
The message is linepanel's request to model, not detail's interface.

### Panels own their keys, emit semantic messages

Panel catches "f", emits `OpenFilterMsg`. Panel doesn't know about stack.
Model receives semantic intent, decides what to do.

No `switch m.active` for key routing - panels own their keys.

### Model orchestrates stack and Store access

Panels don't push/pop or touch Store directly.
They emit messages; model mediates.

### Wrapper Msg routes messages back to specific panel

`linepanel.Msg{Wrapped: innerMsg}` - tells model "this goes to linepanel".
Used when commands produce messages that need to find their panel.

### CloseMsg is universal

Every panel has a `CloseMsg`. Model handles them all the same way:
if bottom of stack, quit; otherwise pop.

## Message Map

| Message | Emitter | Handler | Notes |
|---------|---------|---------|-------|
| `linepanel.Msg` | model | linepanel | wrapper for routing |
| `linepanel.PageMsg` | model | linepanel | delivers page data |
| `linepanel.ColumnsMsg` | model | linepanel | delivers column config |
| `linepanel.ResetMsg` | model | linepanel | resets to initial state |
| `linepanel.OpenDetailMsg` | linepanel | model | pushes detail |
| `linepanel.OpenFilterMsg` | linepanel | model | pushes filter |
| `linepanel.OpenIntakeMsg` | linepanel | model | pushes intake |
| `linepanel.ReloadColumnsMsg` | linepanel | model | reloads from layout file |
| `linepanel.CloseMsg` | linepanel | model | quit |
| `filterpanel.Msg` | model | filterpanel | wrapper for routing |
| `filterpanel.CloseMsg` | filterpanel | model | pop |
| `detail.LineMsg` | model | detail | delivers full line data |
| `detail.ColumnsMsg` | model | detail | delivers column config |
| `detail.CloseMsg` | detail | model | pop |
| `intake.Msg` | model | intake | wrapper for routing |
| `intake.FileSelectedMsg` | intake | model | loads file, replaces stack |
| `intake.CloseMsg` | intake | model | pop or quit |
| `message.GetPageMsg` | linepanel | model | Store access |
| `message.CountMsg` | model | model | internal state |
| `message.SelectedMsg` | linepanel | model | internal state |
| `message.SetFilterMsg` | filterpanel | model | Store access, pop |

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


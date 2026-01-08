# Parcours Development Context

## Recent Work: Message Routing Refactor

### What We Did

1. **Added Intake Panel** - File browser for selecting log files
   - `intake/intake.go` - Uses Board for file listing
   - `intake/message.go` - FileSelectedMsg, Msg wrapper, Wrap()
   - Triggered on startup or via `o` keybinding

2. **Established Wrapping Pattern** for child-to-parent message routing
   - Panels wrap outgoing cmds: `return pnl, Wrap(cmd)`
   - Wrapped messages self-route: `case intake.Msg:` in model.go
   - Model unwraps before forwarding: `m.intakePanel.Update(msg.Wrapped)`
   - Wrapping is for messages FROM panel's children (board) back to panel
   - Parent should NOT wrap messages going TO children (that's backwards)

3. **Cleaned Up SizeMsg**
   - Deleted all custom SizeMsg types
   - All panels now handle `tea.WindowSizeMsg` directly
   - model.go sends standard `panelSize` to all children

4. **Added Wrapping to linepanel**
   - `linepanel/message.go` - Added Msg wrapper, Wrap()
   - `linepanel/linepanel.go` - Wraps all board commands
   - Removed dead `board.PositionMsg`, `board.NavMsg`, `board.PieceMsg` routing from model.go

5. **Removed Old Marker Interfaces**
   - Deleted `linepanel.LinesMsg` interface - use concrete types instead
   - Deleted `detail.DetailMsg` interface - use concrete types instead
   - model.go now uses: `case linepanel.PageMsg, linepanel.ColumnsMsg, linepanel.ResetMsg:`

6. **Renamed tablePanel/tableActive**
   - `tablePanel` → `linePanel`
   - `tableActive` → `lineActive`

### Current State

**All panels with boards now wrap:**
- `intake` - wraps board messages
- `filterpanel` - wraps board and piece messages
- `linepanel` - wraps board messages

**Panels without boards (no wrapping needed):**
- `detail` - simple scroll view, no board, returns nil cmds

**Message routing in model.go:**
| Message Type | Routing Method |
|--------------|----------------|
| Wrapped (intake.Msg, linepanel.Msg, filterpanel.Msg) | By wrapper type, unwrap and forward |
| Concrete (linepanel.PageMsg, detail.LineMsg, etc.) | By type, forward directly |
| tea.KeyPressMsg | By `active` (focus) |
| tea.WindowSizeMsg | Broadcast to all panels |

### Next Steps: Keypress Refactor

The `tea.KeyPressMsg` handling in model.go is the last area needing cleanup. Issues:

1. **`r`, `f` keys** - Check `active == lineActive` before acting. Could be handled by linePanel itself.

2. **Focus changes in message handlers** - `OpenFilterMsg` sets `m.active = filterActive`. Focus management leaking into message handling.

3. **Default key routing** - `switch m.active` to forward keys. With Focus/Blur, panels would know their state.

4. **enter/fallthrough** - Awkward pattern where enter from non-lineActive falls through to default.

**Solution: Focus/Blur Messages**
- Panels receive FocusMsg/BlurMsg when focus changes
- Panels know their own focus state
- Keys could broadcast to all, only focused panel responds
- OR model still routes by active, but focus-dependent behavior lives in panels

### Key Files

```
model.go          - Main routing hub
intake/           - File browser panel
filterpanel/      - Filter editor
linepanel/        - Log lines table
detail/           - Full record view (no board)
board/            - Grid widget used by panels
```

### Patterns

**Wrapping a panel's child messages:**
```go
// In panel's message.go
type Msg struct {
    Wrapped tea.Msg
}

func Wrap(cmd tea.Cmd) tea.Cmd {
    if cmd == nil {
        return nil
    }
    return func() tea.Msg {
        msg := cmd()
        if msg == nil {
            return nil
        }
        return Msg{Wrapped: msg}
    }
}

// In panel's Update, when forwarding to board:
pnl.board, cmd = pnl.board.Update(msg)
return pnl, Wrap(cmd)

// In model.go:
case mypanel.Msg:
    m.myPanel, cmd = m.myPanel.Update(msg.Wrapped)
    return m, cmd
```

**Concrete types for parent-to-child messages:**
```go
// In model.go - route by concrete type, not interface
case linepanel.PageMsg, linepanel.ColumnsMsg, linepanel.ResetMsg:
    m.linePanel, cmd = m.linePanel.Update(msg)
    return m, cmd
```

### TODOs in Code

- See intake/intake.go for feature TODOs (filtering, sorting, multi-file, etc.)
- Add ID to wrappers for multi-instance support

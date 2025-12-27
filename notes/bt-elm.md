# Bubbletea/Elm Patterns

Patterns learned while building Board and FilterPanelToo.

## Core Principles

1. **Messages flow DOWN** - runtime -> Model -> child -> grandchild
2. **Commands flow UP** - returned from Update(), collected by runtime
3. **No bubbling** - children can't send messages to parents directly; they return commands that produce messages
4. **Type assertions are the tax** - nested models require asserting `tea.Model` back to concrete types after `Update()`

## Messages Should Be Values

- Return value types, not pointers: `return CheckedMsg{...}` not `&CheckedMsg{...}`
- If a message needs mutation (like `SetPosition`), have it return a new message instead:
  ```go
  func (m CheckedMsg) SetPosition(rank, file int) board.PieceMsg {
      m.Rank = rank
      m.File = file
      return m  // Copy with new values
  }
  ```
- This keeps messages immutable and plays nice with type switches

## State Ownership

- **Pieces own their state** - they update themselves and emit messages on change
- **Parents track via messages** - don't reach into children, listen to what they emit
- **Avoid caching child state** - if you must cache (like `currentField`), keep it fresh via messages

## Patterns for Common Problems

### Edit Mode
- Board tracks `editMode bool`
- `i` enters edit mode, `enter` exits
- In edit mode, keys go to focused piece; in nav mode, keys navigate
- Different pieces may want different keys (TextInput wants arrows, Operator wants tab)

### Position/Selection Tracking
- Board sends `PositionMsg` when cursor moves
- Board also sends `PositionMsg` after `Replace` (data changed under cursor)
- Parents listen to stay in sync - no accessor methods needed

### Cancel/Commit (Snapshot Pattern)
- Keep two slices: `filters` (working) and `filtersSnapshot` (committed)
- On open: copy snapshot to working state
- On apply: swap working to snapshot (`snapshot = filters`)
- On cancel: just close - next open restores from snapshot
- No cancel message needed; cancel is free

### Duplicate Detection
- Check before adding: same field + value + op = duplicate
- If duplicate found, select existing instead of adding

### Parent Intercepts Keys
- If Model handles a key (like `esc`, `enter`) before routing to panel, panel never sees it
- Solutions: `fallthrough` to route anyway, or restructure key handling
- Be aware of this when adding new key handlers

## Anti-Patterns

- **Accessors to query child state** - use messages instead
- **Pointer messages** - use values
- **Mutating messages** - return new ones
- **Caching without refresh** - listen to messages to stay in sync
- **`cmd != nil` to detect errors** - fragile; use explicit error types

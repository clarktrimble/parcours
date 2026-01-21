
# Wrapped
Panels built on `board` can wrap messages that should be returned to them.
Note!!: they do not currently, stack based routing obviates.

Panel delegates to board:
```go
  lp.board, cmd = lp.board.Update(msg)
  return lp, Wrap(cmd)
  ```
`board` emits nav messages intended for its panel.
By wrapping them, `model` knows which panel should get the message.

```go
  case linepanel.Msg:
      m.linePanel, cmd = m.linePanel.Update(msg.Wrapped)
      return m, cmd
```

Probably this is bust in any case, as typed routing is a straight jacket(one per type).
Leaving for now just in case ...

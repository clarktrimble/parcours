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

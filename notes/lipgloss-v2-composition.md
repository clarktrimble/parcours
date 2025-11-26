# Lipgloss v2 Layer Composition

**Version:** `charm.land/lipgloss/v2@v2.0.0-beta.3.0.20251121225325-f6fbdf23b0ff`

## Key Concepts

### Canvas
- A drawing surface (screen buffer) with fixed width × height
- Provides the coordinate system origin at (0, 0)
- Created with: `lipgloss.NewCanvas(width, height)`
- Renders to string with: `canvas.Render()`

### Layer
- Content positioned in 2D space with optional z-index
- Created with: `lipgloss.NewLayer(id, content)`
- Properties:
  - `x, y` - position relative to parent (default 0, 0)
  - `z` - z-index relative to parent (default 0)
  - `content` - the styled string to draw
  - `layers` - child layers (positioned relative to this layer)

### View (BubbleTea)
- Terminal-level wrapper with metadata (AltScreen, Cursor, WindowTitle, etc.)
- Created with: `tea.NewView(canvas)` or `tea.NewView(string)`
- Only top-level Model returns View - child components return Layers

## Positioning

**Coordinates are relative to parent:**
```go
layer.X(10).Y(5)  // Sets position relative to parent
```

**Absolute position calculated recursively:**
```go
// From layer.go
func (l *Layer) absolutePosition(parentX, parentY, parentZ int) (x, y, z int) {
    return l.x + parentX, l.y + parentY, l.z + parentZ
}
```

**Canvas is at (0, 0, 0)**, so layers added to canvas:
- `layer.X(10).Y(5)` → absolute position (10, 5)
- No `.X()` or `.Y()` → defaults to (0, 0)

## Composition Order

**Compose() draws layers onto canvas:**
```go
func (c *Canvas) Compose(drawer uv.Drawable) {
    drawer.Draw(c, c.Bounds())
}
```

**Draw order determined by:**
1. **Z-index** (lowest to highest) - layers sorted globally by absolute z-index
2. **Compose() call order** - when z-index is equal, later calls draw on top

**Example:**
```go
canvas := lipgloss.NewCanvas(width, height)
canvas.Compose(screenLayer)   // Drawn first (underneath)
canvas.Compose(footerLayer)   // Drawn second (on top)
```

Both have default z=0, so footer draws over screen where they overlap.

## Child Components Pattern

**In v2, child components are NOT tea.Models:**

```go
// Child component
type TableModel struct {
    // state...
}

func (m *TableModel) Update(msg tea.Msg) (*TableModel, tea.Cmd) {
    // Returns concrete type, not tea.Model
}

func (m *TableModel) View() *lipgloss.Layer {
    // Returns Layer, not tea.View
    return lipgloss.NewLayer("table", content)
}
```

**Only top-level Model is a tea.Model:**
```go
func (m Model) View() tea.View {
    // Get layers from children
    screenLayer := m.TableModel.View()
    footerLayer := lipgloss.NewLayer("footer", footerStr).Y(m.Height - 2)

    // Compose on canvas
    canvas := lipgloss.NewCanvas(m.Width, m.Height)
    canvas.Compose(screenLayer)
    canvas.Compose(footerLayer)

    // Wrap in View with metadata
    v := tea.NewView(canvas)
    v.AltScreen = true
    return v
}
```

## Sizing Considerations

**Canvas doesn't clip automatically** - layers can draw outside bounds.

**Child components need to know their available space:**
```go
case tea.WindowSizeMsg:
    m.Width = msg.Width
    m.Height = msg.Height

    // Adjust height for children (reserve space for footer)
    adjustedMsg := tea.WindowSizeMsg{
        Width:  msg.Width,
        Height: msg.Height - 2,  // Reserve 2 lines for footer
    }

    m.TableModel, cmd1 = m.TableModel.Update(adjustedMsg)
```

**Child must use height to limit content:**
- TableModel should render only `height` lines
- Otherwise content extends under footer and gets covered

## Important Details

1. **Layer.Draw() flattens hierarchy** - all layers collected with absolute positions, sorted by z-index, drawn in order

2. **Position setters return pointer** - allows chaining:
   ```go
   layer.X(10).Y(5).Z(1)
   ```

3. **No automatic layout** - you must explicitly position everything

4. **Content size from lipgloss.Width/Height** - layers sized based on actual content dimensions

5. **Overlapping layers** - later z-index (or later Compose()) wins

## Example: Footer at Bottom

```go
func (m Model) View() tea.View {
    // Get screen content
    screenLayer := m.TableModel.View()  // defaults to (0, 0)

    // Create footer and position at bottom
    footerStr := RenderFooter(current, total, filename, m.Width)
    footerLayer := lipgloss.NewLayer("footer", footerStr).Y(m.Height - 2)

    // Compose
    canvas := lipgloss.NewCanvas(m.Width, m.Height)
    canvas.Compose(screenLayer)   // Covers (0,0) to (width, contentHeight)
    canvas.Compose(footerLayer)   // Draws at (0, height-2)

    // Wrap
    v := tea.NewView(canvas)
    v.AltScreen = true
    return v
}
```

## Side-by-Side Layout Example

```go
tableLayer := lipgloss.NewLayer("table", tableContent)  // X defaults to 0
detailLayer := lipgloss.NewLayer("detail", detailContent).X(60)  // Positioned right

canvas := lipgloss.NewCanvas(width, height)
canvas.Compose(tableLayer)
canvas.Compose(detailLayer)
```

## Design Question: Should Children Create Layers?

**Current implementation:**
```go
func (m *TableModel) View() *lipgloss.Layer {
    content := RenderTable(...)
    return lipgloss.NewLayer("table", content)
}
```

**Alternative approach:**
```go
func (m *TableModel) View() string {
    return RenderTable(...)
}

// Parent creates layers:
func (m Model) View() tea.View {
    tableContent := m.TableModel.View()
    tableLayer := lipgloss.NewLayer("table", tableContent)

    footerContent := RenderFooter(...)
    footerLayer := lipgloss.NewLayer("footer", footerContent).Y(m.Height - 2)
    // ...
}
```

**Advantages of children returning strings:**
1. **Separation of concerns** - children generate content, parent handles layout
2. **Parent owns positioning** - only parent knows "footer at Y(height-2)" or "detail at X(60)"
3. **Simpler children** - no lipgloss dependency, just build strings
4. **More reusable** - same content can be positioned differently in different contexts
5. **Layout knowledge centralized** - parent has complete picture of composition

**When children should create layers:**
- Child needs to create complex nested layer hierarchies
- Child manages its own internal layout composition
- Child layer needs specific z-index relative to siblings

For most cases (simple content components), **children returning strings** is cleaner.

## Resources

- Layer implementation: `lipgloss/v2/layer.go`
- Canvas implementation: `lipgloss/v2/canvas.go`
- BubbleTea v2: `charm.land/bubbletea/v2@v2.0.0-rc.2`

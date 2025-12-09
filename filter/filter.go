package filter

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	nt "parcours/entity"
	"parcours/message"
	"parcours/style"
)

// Todo: look at flash after filter apply

// FilterPanel displays a modal dialog for editing filters
type FilterPanel struct {
	filters             []nt.Filter
	selectedFilterIndex int       // Which filter is selected
	selectedField       fieldType // Which field within row is selected

	cursorPos int // Cursor position in value field

	width  int
	height int

	ctx    context.Context
	logger nt.Logger
}

type fieldType int

const (
	fieldEnabled fieldType = iota // Only for filter rows
	fieldDelete                   // Only for filter rows
	fieldOperator
	fieldValue
)

// Todo: put these in nt?
var opNames = map[nt.FilterOp]string{
	nt.Eq:       "==",
	nt.Ne:       "!=",
	nt.Gt:       ">",
	nt.Gte:      ">=",
	nt.Lt:       "<",
	nt.Lte:      "<=",
	nt.Contains: "contains",
	nt.Match:    "matches",
}

var opList = []nt.FilterOp{
	nt.Eq,
	nt.Ne,
	nt.Contains,
	nt.Match,
	nt.Gt,
	nt.Gte,
	nt.Lt,
	nt.Lte,
}

func NewFilterPanel(ctx context.Context, lgr nt.Logger) FilterPanel {
	return FilterPanel{
		ctx:                 ctx,
		logger:              lgr,
		selectedFilterIndex: 0,            // Start with first filter selected
		selectedField:       fieldEnabled, // Start on enabled field
	}
}

func (pnl FilterPanel) Init() tea.Cmd {
	return nil
}

func (pnl FilterPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case message.OpenFilterMsg: // invoked, not routed msg
		// Automatically add the new filter to the list
		newFilter := nt.Filter{
			Op:      nt.Eq, // Default operator
			Field:   msg.Field,
			Value:   msg.Value,
			Enabled: true,
		}
		pnl.filters = append(pnl.filters, newFilter)

		// Select the newly added filter
		pnl.selectedFilterIndex = len(pnl.filters) - 1
		pnl.selectedField = fieldEnabled

	case SizeMsg:
		pnl.width = msg.Width
		pnl.height = msg.Height

	case tea.KeyPressMsg:
		return pnl.handleKey(msg)
	}

	return pnl, nil
}

func (pnl FilterPanel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "p":
		// Apply all enabled filters
		var enabledFilters []nt.Filter
		for _, f := range pnl.filters {
			if f.Enabled {
				enabledFilters = append(enabledFilters, f)
			}
		}

		// Create composite AND filter if multiple enabled filters
		var filterToApply nt.Filter
		if len(enabledFilters) == 0 {
			// No filters enabled, apply nil
			filterToApply = nt.Filter{}
		} else if len(enabledFilters) == 1 {
			// Single filter, use it directly
			filterToApply = enabledFilters[0]
		} else {
			// Multiple filters, combine with AND
			filterToApply = nt.Filter{
				Op:       nt.And,
				Children: enabledFilters,
			}
		}

		return pnl, func() tea.Msg {
			return message.SetFilterMsg{Filter: filterToApply}
		}

	case "tab":
		// Cycle through fields: enabled → delete → operator → value
		switch pnl.selectedField {
		case fieldEnabled:
			pnl.selectedField = fieldDelete
		case fieldDelete:
			pnl.selectedField = fieldOperator
		case fieldOperator:
			pnl.selectedField = fieldValue
		case fieldValue:
			pnl.selectedField = fieldEnabled
		}

	case "left":
		if pnl.selectedField == fieldOperator {
			pnl.prevFilterOperator()
		}

	case "right":
		if pnl.selectedField == fieldOperator {
			pnl.nextFilterOperator()
		}

	case "up":
		// Navigate up in filter list
		if pnl.selectedFilterIndex > 0 {
			pnl.selectedFilterIndex--
			pnl.selectedField = fieldEnabled // Reset to first field
		}

	case "down":
		// Navigate down in filter list
		if pnl.selectedFilterIndex < len(pnl.filters)-1 {
			pnl.selectedFilterIndex++
			pnl.selectedField = fieldEnabled // Reset to first field
		}

	case "d":
		if pnl.selectedField == fieldDelete && pnl.selectedFilterIndex >= 0 {
			// Delete the selected filter
			pnl.filters = append(pnl.filters[:pnl.selectedFilterIndex], pnl.filters[pnl.selectedFilterIndex+1:]...)
			// Adjust selection
			if pnl.selectedFilterIndex >= len(pnl.filters) {
				if len(pnl.filters) > 0 {
					pnl.selectedFilterIndex = len(pnl.filters) - 1
				} else {
					pnl.selectedFilterIndex = -1
					pnl.selectedField = fieldOperator
				}
			}
		}

	case "t":
		if pnl.selectedField == fieldEnabled && pnl.selectedFilterIndex >= 0 {
			// Toggle enabled state
			pnl.filters[pnl.selectedFilterIndex].Enabled = !pnl.filters[pnl.selectedFilterIndex].Enabled
		}
	}

	return pnl, nil
}

func (pnl *FilterPanel) nextFilterOperator() {
	if pnl.selectedFilterIndex < 0 || pnl.selectedFilterIndex >= len(pnl.filters) {
		return
	}
	f := &pnl.filters[pnl.selectedFilterIndex]
	for i, op := range opList {
		if op == f.Op {
			f.Op = opList[(i+1)%len(opList)]
			return
		}
	}
}

func (pnl *FilterPanel) prevFilterOperator() {
	if pnl.selectedFilterIndex < 0 || pnl.selectedFilterIndex >= len(pnl.filters) {
		return
	}
	f := &pnl.filters[pnl.selectedFilterIndex]
	for i, op := range opList {
		if op == f.Op {
			f.Op = opList[(i-1+len(opList))%len(opList)]
			return
		}
	}
}

func (pnl FilterPanel) View() tea.View {
	var content strings.Builder

	// Show existing filters list
	if len(pnl.filters) > 0 {
		content.WriteString("Filters:\n")
		for i, f := range pnl.filters {
			isSelected := i == pnl.selectedFilterIndex

			// Build each field with highlighting
			enabledStr := " "
			if f.Enabled {
				enabledStr = "x"
			}
			if isSelected && pnl.selectedField == fieldEnabled {
				enabledStr = lipgloss.NewStyle().Background(lipgloss.Color("240")).Render("[" + enabledStr + "]")
			} else {
				enabledStr = "[" + enabledStr + "]"
			}

			deleteStr := "[del]"
			if isSelected && pnl.selectedField == fieldDelete {
				deleteStr = lipgloss.NewStyle().Background(lipgloss.Color("240")).Render(deleteStr)
			}

			opStr := opNames[f.Op]
			if isSelected && pnl.selectedField == fieldOperator {
				opStr = lipgloss.NewStyle().Background(lipgloss.Color("240")).Render(opStr)
			}

			valStr := fmt.Sprintf("%v", f.Value)
			if isSelected && pnl.selectedField == fieldValue {
				valStr = lipgloss.NewStyle().Background(lipgloss.Color("240")).Render(valStr)
			}

			rowPrefix := "  "
			if isSelected {
				rowPrefix = "> "
			}

			content.WriteString(fmt.Sprintf("%s%s %s %s %s %s\n", rowPrefix, enabledStr, deleteStr, f.Field, opStr, valStr))
		}
	}

	// Context-aware help text
	var helpText string
	switch pnl.selectedField {
	case fieldEnabled:
		helpText = "t: toggle  Tab: next field  ↑↓: change row  p: apply  Esc: cancel"
	case fieldDelete:
		helpText = "d: delete  Tab: next field  ↑↓: change row  p: apply  Esc: cancel"
	case fieldOperator:
		helpText = "←→: change  Tab: next field  ↑↓: change row  p: apply  Esc: cancel"
	case fieldValue:
		helpText = "Tab: next field  ↑↓: change row  p: apply  Esc: cancel"
	}
	content.WriteString("\n" + style.MutedStyle.Render(helpText))

	// Create a bordered box
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(60)

	dialog := dialogStyle.Render(content.String())

	// Center the dialog
	if pnl.width > 0 && pnl.height > 0 {
		dialogHeight := strings.Count(dialog, "\n") + 1
		dialogWidth := 64 // Approximate width with border

		vPad := (pnl.height - dialogHeight) / 2
		hPad := (pnl.width - dialogWidth) / 2

		if vPad < 0 {
			vPad = 0
		}
		if hPad < 0 {
			hPad = 0
		}

		dialogLayer := lipgloss.NewLayer("filter", dialog).
			X(hPad).
			Y(vPad)

		return tea.NewView(dialogLayer)
	}

	dialogLayer := lipgloss.NewLayer("filter", dialog)
	return tea.NewView(dialogLayer)
}

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/pkg/errors"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"

	"parcours"
)

type Model struct {
	db          *sql.DB
	layout      *parcours.Layout
	fields      []parcours.Field
	fieldIndex  map[string]int
	lines       []parcours.Line
	totalLines  int
	selectedRow int
	scrollOffset int
	width       int
	height      int
}

type loadDataMsg struct {
	fields []parcours.Field
	lines  []parcours.Line
	count  int
	err    error
}

func main() {
	// Open duck db
	db, err := sql.Open("duckdb", "")
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to open duckdb"))
	}
	defer db.Close()

	// Load test data
	logFile := "test/data/smar.log"
	if len(os.Args) > 1 {
		logFile = os.Args[1]
	}

	err = loadTestData(db, logFile)
	if err != nil {
		log.Fatal(err)
	}

	// Load layout
	layout, err := parcours.LoadLayout("layout.yaml")
	if err != nil {
		log.Fatal(err)
	}

	m := Model{
		db:     db,
		layout: layout,
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadData()
}

func (m Model) loadData() tea.Cmd {
	return func() tea.Msg {
		fields, err := getFields(m.db)
		if err != nil {
			return loadDataMsg{err: err}
		}

		var count int
		err = m.db.QueryRow("SELECT COUNT(*) FROM logs").Scan(&count)
		if err != nil {
			return loadDataMsg{err: err}
		}

		lines, err := getLines(m.db, m.scrollOffset, 20)
		if err != nil {
			return loadDataMsg{err: err}
		}

		return loadDataMsg{
			fields: fields,
			lines:  lines,
			count:  count,
		}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loadDataMsg:
		if msg.err != nil {
			// TODO: handle error
			return m, nil
		}
		m.fields = msg.fields
		m.lines = msg.lines
		m.totalLines = msg.count

		// Build field index
		m.fieldIndex = make(map[string]int)
		for i, f := range m.fields {
			m.fieldIndex[f.Name] = i
		}
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.selectedRow > 0 {
				m.selectedRow--
			} else if m.scrollOffset > 0 {
				m.scrollOffset--
				return m, m.loadData()
			}
		case "down", "j":
			if m.selectedRow < len(m.lines)-1 {
				m.selectedRow++
			} else if m.scrollOffset+len(m.lines) < m.totalLines {
				m.scrollOffset++
				return m, m.loadData()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m Model) View() tea.View {
	if len(m.lines) == 0 {
		return tea.NewView("Loading...")
	}

	// Create lipgloss table
	t := table.New()

	// Add headers
	headers := []string{}
	for _, col := range m.layout.Columns {
		if col.Hidden || col.Demote {
			continue
		}
		headers = append(headers, col.Field)
	}
	t.Headers(headers...)

	// Add rows with padded cells and highlight selection
	t.StyleFunc(func(row, col int) lipgloss.Style {
		if row == m.selectedRow {
			return lipgloss.NewStyle().Background(lipgloss.Color("63"))
		}
		return lipgloss.NewStyle()
	})

	for _, line := range m.lines {
		row := []string{}
		for _, col := range m.layout.Columns {
			if col.Hidden || col.Demote {
				continue
			}
			idx := m.fieldIndex[col.Field]
			val := line[idx].String()
			// Pad/truncate to exact width
			padded := fmt.Sprintf("%-*.*s", col.Width, col.Width, val)
			row = append(row, padded)
		}
		t.Row(row...)
	}

	// Style the table
	t.Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240")))

	v := tea.NewView(t.Render())
	v.AltScreen = true
	return v
}

func loadTestData(db *sql.DB, logFile string) error {
	// Create logs table
	createLogs := fmt.Sprintf(`
		CREATE TABLE logs AS
		SELECT
			ROW_NUMBER() OVER () as id,
			ts as timestamp,
			level,
			msg as message
		FROM read_json('%s',
			columns={ts: 'TIMESTAMP', level: 'VARCHAR', msg: 'VARCHAR'},
			format='newline_delimited',
			maximum_object_size=16777216)
	`, logFile)

	_, err := db.Exec(createLogs)
	if err != nil {
		return errors.Wrap(err, "failed to create logs table")
	}

	// Promote fields from layout
	fields := []string{"app_id", "run_id"}
	for _, field := range fields {
		_, err = db.Exec(fmt.Sprintf("ALTER TABLE logs ADD COLUMN IF NOT EXISTS %s VARCHAR", field))
		if err != nil {
			return errors.Wrapf(err, "failed to add column %s", field)
		}

		createRaw := fmt.Sprintf(`
			CREATE TEMP TABLE tmp_raw AS
			SELECT
				ROW_NUMBER() OVER () as id,
				json_extract_string(json_text, '$.%s') as %s
			FROM read_json_objects('%s', format='newline_delimited') AS t(json_text)
		`, field, field, logFile)

		_, err = db.Exec(createRaw)
		if err != nil {
			return errors.Wrapf(err, "failed to create tmp_raw for %s", field)
		}

		_, err = db.Exec(fmt.Sprintf(`
			UPDATE logs
			SET %s = tmp_raw.%s
			FROM tmp_raw
			WHERE logs.id = tmp_raw.id
		`, field, field))
		if err != nil {
			return errors.Wrapf(err, "failed to update %s", field)
		}

		_, err = db.Exec("DROP TABLE tmp_raw")
		if err != nil {
			return errors.Wrapf(err, "failed to drop tmp_raw")
		}
	}

	return nil
}

func getFields(db *sql.DB) ([]parcours.Field, error) {
	rows, err := db.Query(`
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE table_name = 'logs'
		ORDER BY ordinal_position
	`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query schema")
	}
	defer rows.Close()

	var fields []parcours.Field
	for rows.Next() {
		var field parcours.Field
		if err = rows.Scan(&field.Name, &field.Type); err != nil {
			return nil, errors.Wrap(err, "failed to scan field")
		}
		fields = append(fields, field)
	}

	return fields, nil
}

func getLines(db *sql.DB, offset, size int) ([]parcours.Line, error) {
	query := fmt.Sprintf("SELECT * FROM logs LIMIT %d OFFSET %d", size, offset)

	rows, err := db.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query logs")
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get columns")
	}
	count := len(cols)

	var lines []parcours.Line
	for rows.Next() {
		vals := make([]any, count)
		ptrs := make([]any, count)
		for i := range vals {
			ptrs[i] = &vals[i]
		}

		err := rows.Scan(ptrs...)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan row")
		}

		line := make(parcours.Line, count)
		for i, val := range vals {
			line[i] = parcours.Value{Raw: val}
		}

		lines = append(lines, line)
	}

	return lines, rows.Err()
}

package duck

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pkg/errors"

	"parcours"
)

type Duck struct {
	db     *sql.DB
	logger parcours.Logger
	filter parcours.Filter
	sorts  []parcours.Sort
}

func New(lgr parcours.Logger) (dk *Duck, err error) {

	db, err := sql.Open("duckdb", "")
	if err != nil {
		err = errors.Wrapf(err, "failed to open memo duck")
		return
	}

	dk = &Duck{
		db:     db,
		logger: lgr,
	}

	return
}

func (dk *Duck) Close() {
	dk.db.Close()
}

// Load a file
func (dk *Duck) Load(path string, last int) (err error) {
	err = loadDualTable(dk.db, path)
	return
}

// Follow a file
func (dk *Duck) Follow(ctx context.Context, path string, last int) (err error) {
	// Todo:
	return errors.New("not implemented")
}

// Promote a field
func (dk *Duck) Promote(field string) (err error) {
	err = PromoteField(dk.db, field)
	if err != nil {
		return
	}
	err = IndexField(dk.db, field)
	return
}

// SetView Filter and Sort(s)
func (dk *Duck) SetView(filter parcours.Filter, sorts []parcours.Sort) (err error) {
	dk.filter = filter
	dk.sorts = sorts
	return nil
}

// GetView fields and count
func (dk *Duck) GetView() (fields []parcours.Field, count int, err error) {
	// Get fields from schema
	rawFields, err := getFields(dk.db)
	if err != nil {
		return nil, 0, err
	}

	// Convert to parcours.Field
	fields = make([]parcours.Field, len(rawFields))
	for i, f := range rawFields {
		fields[i] = parcours.Field{
			Name: f.Name,
			Type: f.Type,
		}
	}

	// Get count (TODO: apply filter)
	err = dk.db.QueryRow("SELECT COUNT(*) FROM logs").Scan(&count)
	if err != nil {
		err = errors.Wrapf(err, "failed to count logs")
		return nil, 0, err
	}

	return fields, count, nil
}

// GetPage of log lines
func (dk *Duck) GetPage(offset, size int) (lines []parcours.Line, err error) {
	// Get fields to know column order
	fields, _, err := dk.GetView()
	if err != nil {
		return nil, err
	}

	// Build query (TODO: apply filter and sort)
	query := fmt.Sprintf("SELECT * FROM logs LIMIT %d OFFSET %d", size, offset)

	rows, err := dk.db.Query(query)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query logs")
	}
	defer rows.Close()

	// Scan rows into Lines
	for rows.Next() {
		// Create slice of interface{} for scanning
		values := make([]any, len(fields))
		valuePtrs := make([]any, len(fields))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, errors.Wrapf(err, "failed to scan row")
		}

		// Convert to parcours.Line
		line := make(parcours.Line, len(values))
		for i, v := range values {
			line[i] = parcours.Value{Raw: v}
		}

		lines = append(lines, line)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "error iterating rows")
	}

	return lines, nil
}

// Tail streams log lines
func (dk *Duck) Tail(ctx context.Context) (lines <-chan parcours.Line, err error) {
	// Todo:
	return nil, errors.New("not implemented")
}

// unexported

func loadDualTable(db *sql.DB, logFile string) (err error) {
	// Todo: ID alignment issue
	// Both tables use ROW_NUMBER() OVER () assuming both functions read file in same order
	// This works but is implicit - better approaches:
	//   1. Use MD5(raw) as content hash for joining
	//   2. Use natural key from logs (ts + some unique field)
	//   3. Read file once in Go, insert to both tables with same ID
	// Current approach is simple and works for single-file bulk loads

	// Table 1: Structured data with ONLY core fields
	// Use columns parameter to extract only ts, level, msg
	// Other fields stay in raw JSON for controlled promotion later
	// Todo: Consider ENUM type for level (info, debug, warn, error, etc)
	createStructured := fmt.Sprintf(`
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

	_, err = db.Exec(createStructured)
	if err != nil {
		err = errors.Wrapf(err, "failed to create table")
		return
	}

	// Table 2: Raw JSON text
	createRaw := fmt.Sprintf(`
		CREATE TABLE logs_raw AS
		SELECT
			ROW_NUMBER() OVER () as id,
			json_text as raw
		FROM read_json_objects('%s', format='newline_delimited') AS t(json_text)
	`, logFile)

	_, err = db.Exec(createRaw)
	if err != nil {
		err = errors.Wrapf(err, "failed to create table")
		return
	}

	_, err = db.Exec("CREATE INDEX idx_timestamp ON logs(timestamp)")
	err = errors.Wrapf(err, "failed to create index")
	return
}

// PromoteField promotes a field from logs_raw to a column in logs table
func PromoteField(db *sql.DB, fieldName string) (err error) {

	_, err = db.Exec(fmt.Sprintf(
		"ALTER TABLE logs ADD COLUMN IF NOT EXISTS %s VARCHAR",
		fieldName))
	if err != nil {
		err = errors.Wrapf(err, "failed to add column")
		return
	}

	// Step 2: Backfill from logs_raw (extracting from JSON)
	_, err = db.Exec(fmt.Sprintf(`
		UPDATE logs
		SET %s = json_extract_string(logs_raw.raw, '$.%s')
		FROM logs_raw
		WHERE logs.id = logs_raw.id
	`, fieldName, fieldName))
	err = errors.Wrapf(err, "failed to backfill column")
	return
}

func IndexField(db *sql.DB, fieldName string) (err error) {

	_, err = db.Exec(fmt.Sprintf(
		"CREATE INDEX IF NOT EXISTS idx_%s ON logs(%s)",
		fieldName, fieldName))
	err = errors.Wrapf(err, "failed to index column")
	return
}

func getFields(db *sql.DB) (fields []struct {
	Name string
	Type string
}, err error) {

	rows, err := db.Query(`
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE table_name = 'logs'
		ORDER BY ordinal_position
	`)
	if err != nil {
		err = errors.Wrapf(err, "failed to query schema")
		return
	}
	defer rows.Close()

	for rows.Next() {
		var field struct {
			Name string
			Type string
		}
		if err = rows.Scan(&field.Name, &field.Type); err != nil {
			err = errors.Wrapf(err, "failed to scan field")
			return
		}
		fields = append(fields, field)
	}

	return
}

package duck

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	nt "parcours/entity"
)

// Todo: use uptodate lib from duckdb in main

type Duck struct {
	db       *sql.DB
	logger   nt.Logger
	filter   nt.Filter
	sorts    []nt.Sort
	filename string
}

func New(lgr nt.Logger) (dk *Duck, err error) {

	db, err := sql.Open("duckdb", "")
	if err != nil {
		err = errors.Wrapf(err, "failed to open memo duck")
		return
	}

	dk = &Duck{
		db:     db,
		sorts:  []nt.Sort{},
		logger: lgr,
	}

	return
}

func (dk *Duck) Close() {
	dk.db.Close()
}

// Load a file
func (dk *Duck) Load(path string, last int) (err error) {
	dk.filename = path
	err = loadDualTable(dk.db, path)
	return
}

// Name returns the name of the loaded file
func (dk *Duck) Name() string {
	return dk.filename
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
func (dk *Duck) SetView(filter nt.Filter, sorts []nt.Sort) (err error) {
	dk.filter = filter
	dk.sorts = sorts
	return nil
}

// buildWhereClause converts a Filter to SQL WHERE clause
func (dk *Duck) buildWhereClause() string {

	clause := dk.buildFilterExpr(dk.filter)
	if clause == "" {
		return ""
	}
	return "WHERE " + clause
}

// buildFilterExpr recursively builds filter expression (without WHERE prefix)
func (dk *Duck) buildFilterExpr(f nt.Filter) string {
	switch f.Op {
	case nt.Eq:
		return fmt.Sprintf("%s = '%v'", f.Field, f.Value)
	case nt.Ne:
		return fmt.Sprintf("%s != '%v'", f.Field, f.Value)
	case nt.Gt:
		return fmt.Sprintf("%s > %v", f.Field, f.Value)
	case nt.Gte:
		return fmt.Sprintf("%s >= %v", f.Field, f.Value)
	case nt.Lt:
		return fmt.Sprintf("%s < %v", f.Field, f.Value)
	case nt.Lte:
		return fmt.Sprintf("%s <= %v", f.Field, f.Value)
	case nt.Contains:
		return fmt.Sprintf("%s LIKE '%%%v%%'", f.Field, f.Value)
	case nt.And:
		var clauses []string
		for _, child := range f.Children {
			if expr := dk.buildFilterExpr(child); expr != "" {
				clauses = append(clauses, expr)
			}
		}
		if len(clauses) == 0 {
			return ""
		}
		return "(" + strings.Join(clauses, " AND ") + ")"
	case nt.Or:
		var clauses []string
		for _, child := range f.Children {
			if expr := dk.buildFilterExpr(child); expr != "" {
				clauses = append(clauses, expr)
			}
		}
		if len(clauses) == 0 {
			return ""
		}
		return "(" + strings.Join(clauses, " OR ") + ")"
	case nt.Not:
		if len(f.Children) > 0 {
			if expr := dk.buildFilterExpr(f.Children[0]); expr != "" {
				return "NOT (" + expr + ")"
			}
		}
		return ""
	// TODO: handle Match (regex)
	default:
		return ""
	}
}

// GetView fields and count
func (dk *Duck) GetView() (fields []nt.Field, count int, err error) {
	// Get fields from schema
	rawFields, err := getFields(dk.db)
	if err != nil {
		return nil, 0, err
	}

	// Convert to nt.Field
	fields = make([]nt.Field, len(rawFields))
	for i, f := range rawFields {
		fields[i] = nt.Field{
			Name: f.Name,
			Type: f.Type,
		}
	}

	// Get count with filter
	whereClause := dk.buildWhereClause()
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM logs %s", whereClause)
	err = dk.db.QueryRow(countQuery).Scan(&count)
	if err != nil {
		err = errors.Wrapf(err, "failed to count logs")
		return nil, 0, err
	}

	return fields, count, nil
}

// GetPage of log lines
func (dk *Duck) GetPage(offset, size int) (lines []nt.Line, err error) {

	// Apply filter (TODO: apply sort)
	whereClause := dk.buildWhereClause()
	query := fmt.Sprintf("SELECT * FROM logs %s LIMIT %d OFFSET %d", whereClause, size, offset)

	rows, err := dk.db.Query(query)
	if err != nil {
		err = errors.Wrapf(err, "failed to query logs")
		return
	}
	defer rows.Close()

	count, err := columnCount(rows)
	if err != nil {
		return
	}

	for rows.Next() {
		var vals []any
		vals, err = scanRow(rows, count)
		if err != nil {
			err = errors.Wrapf(err, "failed to scan row")
			return
		}

		values := make([]nt.Value, count)
		for i, val := range vals {
			values[i] = nt.Value{Raw: val}
		}

		// Todo: can do better than first col is id?
		// Todo: want to think about specifying order?  would untangle table a little
		// Todo: dont crash

		line := nt.Line{
			Id:     values[0].String(),
			Values: values,
		}
		lines = append(lines, line)
	}

	err = rows.Err()
	err = errors.Wrapf(err, "error iterating rows")
	return
}

func columnCount(rows *sql.Rows) (int, error) {
	cols, err := rows.Columns()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get cols from query rows")
	}
	return len(cols), nil
}

func scanRow(rows *sql.Rows, columnCount int) ([]any, error) {
	vals := make([]any, columnCount)
	ptrs := make([]any, columnCount)
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	err := rows.Scan(ptrs...)
	return vals, err
}

// GetLine returns raw json for a line
func (dk *Duck) GetLine(id string) (data map[string]any, err error) {

	query := "SELECT raw FROM logs_raw WHERE id = ?"

	var raw any
	err = dk.db.QueryRow(query, id).Scan(&raw)
	if err != nil {
		err = errors.Wrapf(err, "failed to query raw JSON")
		return
	}

	var ok bool
	data, ok = raw.(map[string]any)
	if !ok {
		err = errors.Errorf("expected map[string]any from driver, got %T", raw)
	}
	return
}

// Tail streams log lines
func (dk *Duck) Tail(ctx context.Context) (lines <-chan nt.Line, err error) {
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

	// Table 2: Raw JSON
	createRaw := fmt.Sprintf(`
		CREATE TABLE logs_raw AS
		SELECT
			ROW_NUMBER() OVER () as id,
			json_text::JSON as raw
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

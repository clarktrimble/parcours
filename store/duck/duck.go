package parcours

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
)

func newDuck() (db *sql.DB, err error) {

	db, err = sql.Open("duckdb", "")
	err = errors.Wrapf(err, "failed to open memo duck")
	return
	//Todo: where? defer db.Close()
}

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

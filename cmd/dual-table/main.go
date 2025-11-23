package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/marcboeker/go-duckdb"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <log-file.log>\n", os.Args[0])
		os.Exit(1)
	}

	logFile := os.Args[1]
	absPath, err := filepath.Abs(logFile)
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		log.Fatalf("Log file not found: %s", absPath)
	}

	fmt.Printf("Loading logs from: %s\n\n", absPath)

	// Open in-memory DuckDB database
	db, err := sql.Open("duckdb", "")
	if err != nil {
		log.Fatalf("Failed to open DuckDB: %v", err)
	}
	defer db.Close()

	// Load logs using dual-table approach
	start := time.Now()
	if err := loadDualTable(db, absPath); err != nil {
		log.Fatalf("Failed to load logs: %v", err)
	}
	elapsed := time.Since(start)

	// Get row count
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM logs").Scan(&count)
	if err != nil {
		log.Fatalf("Failed to count logs: %v", err)
	}

	fmt.Printf("✓ Loaded %d log entries in %v\n\n", count, elapsed)

	// Show what we got
	showSchema(db)
	showSampleData(db)

	// Test field promotion
	fmt.Println("\n=== Testing Field Promotion ===")
	if err := testPromotion(db); err != nil {
		log.Printf("Promotion test failed: %v", err)
	}
}

func loadDualTable(db *sql.DB, logFile string) error {
	// TODO: ID alignment issue
	// Both tables use ROW_NUMBER() OVER () assuming both functions read file in same order
	// This works but is implicit - better approaches:
	//   1. Use MD5(raw) as content hash for joining
	//   2. Use natural key from logs (ts + some unique field)
	//   3. Read file once in Go, insert to both tables with same ID
	// Current approach is simple and works for single-file bulk loads

	// Table 1: Structured data with ONLY core fields
	// Use columns parameter to extract only ts, level, msg
	// Other fields stay in raw JSON for controlled promotion later
	// TODO: Consider ENUM type for level (info, debug, warn, error, etc)
	fmt.Println("Creating structured logs table...")
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

	_, err := db.Exec(createStructured)
	if err != nil {
		return fmt.Errorf("create structured table: %w", err)
	}

	// Table 2: Raw JSON text
	// Read as plain text, no parsing
	fmt.Println("Creating raw logs table...")
	createRaw := fmt.Sprintf(`
		CREATE TABLE logs_raw AS
		SELECT
			ROW_NUMBER() OVER () as id,
			json_text as raw
		FROM read_json_objects('%s', format='newline_delimited') AS t(json_text)
	`, logFile)

	_, err = db.Exec(createRaw)
	if err != nil {
		return fmt.Errorf("create raw table: %w", err)
	}

	// Create indexes on structured table
	fmt.Println("Creating indexes...")
	_, err = db.Exec("CREATE INDEX idx_timestamp ON logs(timestamp)")
	if err != nil {
		return fmt.Errorf("create timestamp index: %w", err)
	}

	_, err = db.Exec("CREATE INDEX idx_level ON logs(level)")
	if err != nil {
		return fmt.Errorf("create level index: %w", err)
	}

	return nil
}

func showSchema(db *sql.DB) {
	fmt.Println("=== Structured Table Schema ===")
	rows, err := db.Query(`
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE table_name = 'logs'
		ORDER BY ordinal_position
	`)
	if err != nil {
		log.Printf("Failed to query schema: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var colName, dataType string
		if err := rows.Scan(&colName, &dataType); err != nil {
			continue
		}
		fmt.Printf("  %s (%s)\n", colName, dataType)
	}

	fmt.Println("\n=== Raw Table Schema ===")
	rows2, err := db.Query(`
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE table_name = 'logs_raw'
		ORDER BY ordinal_position
	`)
	if err != nil {
		log.Printf("Failed to query schema: %v", err)
		return
	}
	defer rows2.Close()

	for rows2.Next() {
		var colName, dataType string
		if err := rows2.Scan(&colName, &dataType); err != nil {
			continue
		}
		fmt.Printf("  %s (%s)\n", colName, dataType)
	}
	fmt.Println()
}

func showSampleData(db *sql.DB) {
	fmt.Println("=== Sample: Structured + Raw (joined) ===")

	rows, err := db.Query(`
		SELECT
			logs.id,
			logs.timestamp,
			logs.level,
			logs.message,
			logs_raw.raw::VARCHAR as raw
		FROM logs
		JOIN logs_raw ON logs.id = logs_raw.id
		LIMIT 3
	`)
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var ts time.Time
		var level, message, raw string

		if err := rows.Scan(&id, &ts, &level, &message, &raw); err != nil {
			log.Printf("Scan failed: %v", err)
			continue
		}

		fmt.Printf("\nLog #%d:\n", id)
		fmt.Printf("  Time: %s\n", ts.Format("15:04:05"))
		fmt.Printf("  Level: %s\n", level)
		fmt.Printf("  Message: %s\n", message)
		if len(raw) > 100 {
			fmt.Printf("  Raw: %s...\n", raw[:100])
		} else {
			fmt.Printf("  Raw: %s\n", raw)
		}
	}

	fmt.Println()
}

// PromoteField promotes a field from logs_raw to a column in logs table
func PromoteField(db *sql.DB, fieldName string) error {
	fmt.Printf("\n=== Promoting field '%s' ===\n", fieldName)

	// Step 1: Add column to logs table
	fmt.Printf("1. Adding column '%s' to logs table...\n", fieldName)
	start := time.Now()
	_, err := db.Exec(fmt.Sprintf(
		"ALTER TABLE logs ADD COLUMN IF NOT EXISTS %s VARCHAR",
		fieldName))
	if err != nil {
		return fmt.Errorf("add column: %w", err)
	}
	fmt.Printf("   ✓ Column added in %v\n", time.Since(start))

	// Step 2: Backfill from logs_raw (extracting from JSON)
	fmt.Printf("2. Backfilling from logs_raw JSON...\n")
	start = time.Now()
	result, err := db.Exec(fmt.Sprintf(`
		UPDATE logs
		SET %s = json_extract_string(logs_raw.raw, '$.%s')
		FROM logs_raw
		WHERE logs.id = logs_raw.id
	`, fieldName, fieldName))
	if err != nil {
		return fmt.Errorf("backfill: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("   ✓ Backfilled %d rows in %v\n", rowsAffected, time.Since(start))

	// Step 3: Create index
	fmt.Printf("3. Creating index...\n")
	start = time.Now()
	_, err = db.Exec(fmt.Sprintf(
		"CREATE INDEX IF NOT EXISTS idx_%s ON logs(%s)",
		fieldName, fieldName))
	if err != nil {
		return fmt.Errorf("create index: %w", err)
	}
	fmt.Printf("   ✓ Index created in %v\n", time.Since(start))

	fmt.Printf("✓ Field '%s' promoted successfully!\n\n", fieldName)
	return nil
}

func testPromotion(db *sql.DB) error {
	fieldName := "request_id"

	// Query BEFORE promotion (extract from logs_raw)
	fmt.Printf("1. Query BEFORE promotion (extracting from logs_raw):\n")
	queryBefore := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM logs
		JOIN logs_raw ON logs.id = logs_raw.id
		WHERE json_extract_string(logs_raw.raw, '$.%s') IS NOT NULL
	`, fieldName)

	start := time.Now()
	var countBefore int
	err := db.QueryRow(queryBefore).Scan(&countBefore)
	if err != nil {
		return fmt.Errorf("query before: %w", err)
	}
	elapsedBefore := time.Since(start)
	fmt.Printf("   Found %d logs with %s (took %v)\n", countBefore, fieldName, elapsedBefore)

	// Promote the field
	if err := PromoteField(db, fieldName); err != nil {
		return fmt.Errorf("promotion: %w", err)
	}

	// Query AFTER promotion (use indexed column)
	fmt.Printf("2. Query AFTER promotion (indexed column):\n")
	queryAfter := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM logs
		WHERE %s IS NOT NULL
	`, fieldName)

	start = time.Now()
	var countAfter int
	err = db.QueryRow(queryAfter).Scan(&countAfter)
	if err != nil {
		return fmt.Errorf("query after: %w", err)
	}
	elapsedAfter := time.Since(start)
	fmt.Printf("   Found %d logs with %s (took %v)\n", countAfter, fieldName, elapsedAfter)

	// Show speedup
	if elapsedBefore > elapsedAfter && elapsedAfter > 0 {
		speedup := float64(elapsedBefore) / float64(elapsedAfter)
		fmt.Printf("\n✓ Query is %.1fx faster after promotion!\n", speedup)
	}

	// Show the promoted schema
	fmt.Println("\n=== Updated Logs Table Schema ===")
	rows, err := db.Query(`
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE table_name = 'logs'
		ORDER BY ordinal_position
	`)
	if err != nil {
		return fmt.Errorf("query schema: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var colName, dataType string
		if err := rows.Scan(&colName, &dataType); err != nil {
			continue
		}
		fmt.Printf("  %s (%s)\n", colName, dataType)
	}

	fmt.Println()
	return nil
}


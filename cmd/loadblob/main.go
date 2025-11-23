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

	// Verify log file exists
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

	// Load logs using DuckDB's native JSON loader
	start := time.Now()
	if err := loadLogs(db, absPath); err != nil {
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

	// Run example queries
	fmt.Println("=== Example Queries ===")
	runExampleQueries(db)

	// Demonstrate field promotion
	fmt.Println("\n=== Field Promotion Demo ===")
	demoFieldPromotion(db)
}

func loadLogs(db *sql.DB, logFile string) error {
	// Create table with schema mapping per design doc
	// ts → timestamp, msg → message, level stays, raw stores full JSON
	// Use records=false to keep JSON as STRUCT, then cast to JSON
	createSQL := fmt.Sprintf(`
		CREATE TABLE logs AS
		SELECT
			ROW_NUMBER() OVER () as id,
			CAST(json.ts AS TIMESTAMP) as timestamp,
			json.level as level,
			json.msg as message,
			json::JSON as raw
		FROM read_json_auto('%s', maximum_object_size=16777216, records=false) AS t(json)
	`, logFile)

	_, err := db.Exec(createSQL)
	if err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	// Create indexes for fast queries
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

// PromoteField promotes a field from the raw JSON blob to an indexed column
// This dramatically speeds up queries on that field
func PromoteField(db *sql.DB, fieldName string) error {
	fmt.Printf("\n=== Promoting field '%s' ===\n", fieldName)

	// Step 1: Add column
	fmt.Printf("1. Adding column '%s'...\n", fieldName)
	start := time.Now()
	_, err := db.Exec(fmt.Sprintf(
		"ALTER TABLE logs ADD COLUMN IF NOT EXISTS %s VARCHAR",
		fieldName))
	if err != nil {
		return fmt.Errorf("add column: %w", err)
	}
	fmt.Printf("   ✓ Column added in %v\n", time.Since(start))

	// Step 2: Backfill from JSON
	fmt.Printf("2. Backfilling data from raw JSON...\n")
	start = time.Now()
	result, err := db.Exec(fmt.Sprintf(
		"UPDATE logs SET %s = json_extract_string(raw, '$.%s')",
		fieldName, fieldName))
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

func runExampleQueries(db *sql.DB) {
	// Query 1: Count by log level
	fmt.Println("1. Log level breakdown:")
	rows, err := db.Query(`
		SELECT level, COUNT(*) as count
		FROM logs
		GROUP BY level
		ORDER BY count DESC
	`)
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var level string
		var count int
		if err := rows.Scan(&level, &count); err != nil {
			log.Printf("Scan failed: %v", err)
			continue
		}
		fmt.Printf("   %s: %d\n", level, count)
	}

	// Query 2: Recent messages
	fmt.Println("\n2. First 5 log messages:")
	rows2, err := db.Query(`
		SELECT timestamp, level, message
		FROM logs
		ORDER BY timestamp
		LIMIT 5
	`)
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}
	defer rows2.Close()

	for rows2.Next() {
		var ts time.Time
		var level, message string
		if err := rows2.Scan(&ts, &level, &message); err != nil {
			log.Printf("Scan failed: %v", err)
			continue
		}
		fmt.Printf("   [%s] %s: %s\n", ts.Format("15:04:05"), level, message)
	}

	// Query 3: JSON field extraction from raw column
	fmt.Println("\n3. Unique app_id and run_id (from raw JSON):")
	rows3, err := db.Query(`
		SELECT DISTINCT
			json_extract_string(raw, '$.app_id') as app_id,
			json_extract_string(raw, '$.run_id') as run_id
		FROM logs
		WHERE json_extract_string(raw, '$.app_id') IS NOT NULL
		LIMIT 5
	`)
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}
	defer rows3.Close()

	for rows3.Next() {
		var appID, runID sql.NullString
		if err := rows3.Scan(&appID, &runID); err != nil {
			log.Printf("Scan failed: %v", err)
			continue
		}
		fmt.Printf("   app_id=%s, run_id=%s\n", appID.String, runID.String)
	}

	// Query 4: Time range
	fmt.Println("\n4. Time range of logs:")
	var minTime, maxTime time.Time
	err = db.QueryRow(`
		SELECT MIN(timestamp), MAX(timestamp)
		FROM logs
	`).Scan(&minTime, &maxTime)
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}
	fmt.Printf("   From: %s\n", minTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("   To:   %s\n", maxTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("   Duration: %v\n", maxTime.Sub(minTime))

	fmt.Println()
}

func demoFieldPromotion(db *sql.DB) {
	fieldName := "request_id"

	// Query 1: BEFORE promotion (slow - JSON extraction)
	fmt.Printf("1. Query BEFORE promotion (using JSON extraction):\n")
	queryBefore := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM logs
		WHERE json_extract_string(raw, '$.%s') IS NOT NULL
	`, fieldName)

	start := time.Now()
	var countBefore int
	err := db.QueryRow(queryBefore).Scan(&countBefore)
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}
	elapsedBefore := time.Since(start)
	fmt.Printf("   Found %d logs with %s (took %v)\n", countBefore, fieldName, elapsedBefore)

	// Promote the field
	if err := PromoteField(db, fieldName); err != nil {
		log.Printf("Promotion failed: %v", err)
		return
	}

	// Query 2: AFTER promotion (fast - indexed column)
	fmt.Printf("2. Query AFTER promotion (using indexed column):\n")
	queryAfter := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM logs
		WHERE %s IS NOT NULL
	`, fieldName)

	start = time.Now()
	var countAfter int
	err = db.QueryRow(queryAfter).Scan(&countAfter)
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}
	elapsedAfter := time.Since(start)
	fmt.Printf("   Found %d logs with %s (took %v)\n", countAfter, fieldName, elapsedAfter)

	// Show speedup
	if elapsedBefore > elapsedAfter {
		speedup := float64(elapsedBefore) / float64(elapsedAfter)
		fmt.Printf("\n✓ Query is %.1fx faster after promotion!\n", speedup)
	}

	// Show example query with the promoted field
	fmt.Printf("\n3. Example query using promoted field:\n")
	rows, err := db.Query(fmt.Sprintf(`
		SELECT timestamp, level, message, %s
		FROM logs
		WHERE %s IS NOT NULL
		ORDER BY timestamp
		LIMIT 3
	`, fieldName, fieldName))
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var ts time.Time
		var level, message, reqID sql.NullString
		if err := rows.Scan(&ts, &level, &message, &reqID); err != nil {
			log.Printf("Scan failed: %v", err)
			continue
		}
		fmt.Printf("   [%s] %s: %s (request_id=%s)\n",
			ts.Format("15:04:05"), level.String, message.String, reqID.String)
	}

	fmt.Println()
}

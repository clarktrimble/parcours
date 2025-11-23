package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
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

	// Load logs
	start := time.Now()
	if err := LoadLogsFromFile(db, absPath); err != nil {
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

	// Test field promotion
	fmt.Println("=== Testing Field Promotion ===")
	if err := testPromotion(db); err != nil {
		log.Printf("Promotion test failed: %v", err)
	}
}

// Design Decision: Parse JSON in Go, not DuckDB
//
// Why Go parsing instead of DuckDB's read_json_auto?
//
// 1. We need BOTH extracted fields AND original raw JSON text
// 2. DuckDB can't do both in one pass:
//    - read_json_auto with records=false gives STRUCT but reconstructs JSON with union schema (all fields, lots of nils)
//    - read_json_objects gives raw text but requires multiple parses for field extraction
//    - CTE with ::JSON cast might work but unclear if it avoids re-parsing
//
// 3. Go approach is simple and clear:
//    - Parse JSON once per line
//    - Extract fields we want (ts, level, msg)
//    - Keep original line text as raw
//    - INSERT both into DuckDB
//
// 4. Fast enough for our use case (millions of lines)
//
// Schema:
//   CREATE TABLE logs (
//       id INTEGER PRIMARY KEY,
//       timestamp TIMESTAMP,
//       level VARCHAR,
//       message VARCHAR,
//       raw VARCHAR  -- original JSON text for field discovery & promotion
//   )

// LogEntry represents a single log line
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
	Raw       string // original JSON text
}

// LoadLogsFromFile reads a NDJSON log file and loads it into DuckDB
// Parses JSON in Go to extract core fields while preserving raw JSON
func LoadLogsFromFile(db *sql.DB, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	// Create table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS logs (
			id INTEGER PRIMARY KEY,
			timestamp TIMESTAMP NOT NULL,
			level VARCHAR NOT NULL,
			message VARCHAR,
			raw VARCHAR NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	// Create indexes
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_timestamp ON logs(timestamp)")
	if err != nil {
		return fmt.Errorf("create timestamp index: %w", err)
	}

	_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_level ON logs(level)")
	if err != nil {
		return fmt.Errorf("create level index: %w", err)
	}

	// Read and parse logs
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if line == "" {
			continue
		}

		// Parse JSON once
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Skip malformed lines
			continue
		}

		// Extract core fields
		logEntry := LogEntry{
			Raw: line, // preserve original
		}

		// Extract timestamp (field: ts)
		if ts, ok := entry["ts"].(string); ok {
			logEntry.Timestamp, _ = time.Parse(time.RFC3339Nano, ts)
		}

		// Extract level
		if level, ok := entry["level"].(string); ok {
			logEntry.Level = level
		}

		// Extract message (field: msg)
		if msg, ok := entry["msg"].(string); ok {
			logEntry.Message = msg
		}

		// Insert into DuckDB
		_, err = db.Exec(`
			INSERT INTO logs (id, timestamp, level, message, raw)
			VALUES (?, ?, ?, ?, ?)
		`, lineNum, logEntry.Timestamp, logEntry.Level, logEntry.Message, logEntry.Raw)

		if err != nil {
			return fmt.Errorf("insert line %d: %w", lineNum, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan file: %w", err)
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

	// Step 2: Backfill from JSON (raw is now VARCHAR, not JSON type)
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

func testPromotion(db *sql.DB) error {
	fieldName := "request_id"

	// Query BEFORE promotion
	fmt.Printf("1. Query BEFORE promotion (JSON extraction from VARCHAR):\n")
	queryBefore := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM logs
		WHERE json_extract_string(raw, '$.%s') IS NOT NULL
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

	// Query AFTER promotion
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
		fmt.Printf("\n✓ Query is %.1fx faster after promotion!\n\n", speedup)
	}

	return nil
}

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
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

	// Open in-memory DuckDB database
	db, err := sql.Open("duckdb", "")
	if err != nil {
		log.Fatalf("Failed to open DuckDB: %v", err)
	}
	defer db.Close()

	// Load logs
	fmt.Printf("Loading logs from: %s\n\n", absPath)
	start := time.Now()
	if err := loadLogs(db, absPath); err != nil {
		log.Fatalf("Failed to load logs: %v", err)
	}
	fmt.Printf("✓ Loaded in %v\n\n", time.Since(start))

	// Inspect schema
	inspectSchema(db)
}

func loadLogs(db *sql.DB, logFile string) error {
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

	return nil
}

func inspectSchema(db *sql.DB) {
	fmt.Println("=== Schema Inspection ===")

	// Get table columns (promoted fields)
	fmt.Println("1. Promoted columns (fast, indexed):")
	rows, err := db.Query(`
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE table_name = 'logs'
		ORDER BY ordinal_position
	`)
	if err != nil {
		log.Fatalf("Failed to query schema: %v", err)
	}
	defer rows.Close()

	promotedFields := make(map[string]bool)
	for rows.Next() {
		var colName, dataType string
		if err := rows.Scan(&colName, &dataType); err != nil {
			log.Printf("Scan failed: %v", err)
			continue
		}
		fmt.Printf("   %s (%s)\n", colName, dataType)
		promotedFields[colName] = true
	}

	// Discover all JSON fields
	fmt.Println("\n2. Discovering all fields in raw JSON...")
	start := time.Now()

	// Get all unique keys from JSON objects by unnesting them
	rows2, err := db.Query(`
		SELECT DISTINCT unnest(json_keys(raw)) as key
		FROM logs
	`)
	if err != nil {
		log.Fatalf("Failed to discover JSON fields: %v", err)
	}
	defer rows2.Close()

	// Collect all unique keys
	allKeys := make(map[string]bool)
	for rows2.Next() {
		var key string
		if err := rows2.Scan(&key); err != nil {
			log.Printf("Scan failed: %v", err)
			continue
		}
		allKeys[key] = true
	}

	elapsed := time.Since(start)

	// Sort keys for display
	var sortedKeys []string
	for key := range allKeys {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	fmt.Printf("   Found %d unique fields (scanned in %v)\n\n", len(sortedKeys), elapsed)

	// Show fields by status
	fmt.Println("3. Field status:")
	fmt.Println("\n   Core fields (always promoted):")
	coreFields := []string{"id", "timestamp", "level", "message"}
	for _, field := range coreFields {
		fmt.Printf("     ✓ %s\n", field)
	}

	fmt.Println("\n   Fields in raw JSON (not yet promoted):")
	unpromoted := 0
	for _, key := range sortedKeys {
		if !promotedFields[key] {
			// Get count
			var count int
			err := db.QueryRow(fmt.Sprintf(`
				SELECT COUNT(*)
				FROM logs
				WHERE json_extract_string(raw, '$.%s') IS NOT NULL
			`, key)).Scan(&count)

			if err != nil || count == 0 {
				continue
			}

			// Get sample value
			var sampleValue sql.NullString
			db.QueryRow(fmt.Sprintf(`
				SELECT json_extract_string(raw, '$.%s')
				FROM logs
				WHERE json_extract_string(raw, '$.%s') IS NOT NULL
				LIMIT 1
			`, key, key)).Scan(&sampleValue)

			fmt.Printf("     • %s (in %d logs, e.g., %q)\n", key, count, truncate(sampleValue.String, 40))
			unpromoted++
		}
	}

	if unpromoted == 0 {
		fmt.Println("     (none)")
	}

	fmt.Println("\n   Other promoted fields:")
	otherPromoted := 0
	for _, key := range sortedKeys {
		if promotedFields[key] && !contains(coreFields, key) {
			fmt.Printf("     ✓ %s (promoted)\n", key)
			otherPromoted++
		}
	}
	if otherPromoted == 0 {
		fmt.Println("     (none)")
	}

	fmt.Println()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

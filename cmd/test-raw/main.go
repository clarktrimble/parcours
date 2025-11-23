package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/marcboeker/go-duckdb"
)

func main() {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Load a single log
	_, err = db.Exec(`
		CREATE TABLE logs AS
		SELECT
			ROW_NUMBER() OVER () as id,
			CAST(json.ts AS TIMESTAMP) as timestamp,
			json.level as level,
			json.msg as message,
			json::JSON as raw
		FROM read_json_auto('test/data/smar.log', maximum_object_size=16777216, records=false) AS t(json)
	`)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Testing different ways to retrieve raw JSON ===")

	// Test 1: raw directly
	fmt.Println("Test 1: SELECT raw FROM logs WHERE id = 1")
	var raw1 interface{}
	err = db.QueryRow("SELECT raw FROM logs WHERE id = 1").Scan(&raw1)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("  Type: %T\n", raw1)
		fmt.Printf("  Value: %v\n", raw1)
	}

	// Test 2: raw::VARCHAR
	fmt.Println("\nTest 2: SELECT raw::VARCHAR FROM logs WHERE id = 1")
	var raw2 string
	err = db.QueryRow("SELECT raw::VARCHAR FROM logs WHERE id = 1").Scan(&raw2)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("  Type: %T\n", raw2)
		fmt.Printf("  Value: %s\n", raw2[:100]+"...")
	}

	// Test 3: json_extract(raw, '$')
	fmt.Println("\nTest 3: SELECT json_extract(raw, '$') FROM logs WHERE id = 1")
	var raw3 interface{}
	err = db.QueryRow("SELECT json_extract(raw, '$') FROM logs WHERE id = 1").Scan(&raw3)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("  Type: %T\n", raw3)
		fmt.Printf("  Value: %v\n", raw3)
	}

	// Test 4: Try scanning raw into a string
	fmt.Println("\nTest 4: SELECT raw FROM logs (scan into string)")
	var raw4 string
	err = db.QueryRow("SELECT raw FROM logs WHERE id = 1").Scan(&raw4)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("  Type: %T\n", raw4)
		fmt.Printf("  Value: %s\n", raw4[:100]+"...")
	}

	fmt.Println("\n=== Conclusion ===")
	fmt.Println("The go-duckdb driver returns raw JSON as...")
}

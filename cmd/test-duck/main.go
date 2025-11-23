package main

import (
	"context"
	"fmt"
	"log"
	"os"

	_ "github.com/marcboeker/go-duckdb"

	"parcours"
	"parcours/store/duck"
)

// Simple logger that prints to stdout
type simpleLogger struct{}

func (l *simpleLogger) Info(ctx context.Context, msg string, kv ...any) {
	fmt.Printf("[INFO] %s %v\n", msg, kv)
}

func (l *simpleLogger) Error(ctx context.Context, msg string, err error, kv ...any) {
	fmt.Printf("[ERROR] %s: %v %v\n", msg, err, kv)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <log-file.log>\n", os.Args[0])
		os.Exit(1)
	}

	logFile := os.Args[1]

	// Create Duck store
	logger := &simpleLogger{}
	dk, err := duck.New(logger)
	if err != nil {
		log.Fatalf("Failed to create Duck: %v", err)
	}
	defer dk.Close()

	fmt.Printf("Loading logs from: %s\n\n", logFile)

	// Load the log file
	if err := dk.Load(logFile, 0); err != nil {
		log.Fatalf("Failed to load logs: %v", err)
	}

	// Set empty view (no filter, no sort)
	if err := dk.SetView(parcours.Filter{}, nil); err != nil {
		log.Fatalf("Failed to set view: %v", err)
	}

	// Get view info
	fields, count, err := dk.GetView()
	if err != nil {
		log.Fatalf("Failed to get view: %v", err)
	}

	fmt.Printf("✓ Loaded %d log entries\n\n", count)

	// Show schema
	fmt.Println("=== Fields ===")
	for i, field := range fields {
		fmt.Printf("  [%d] %s (%s)\n", i, field.Name, field.Type)
	}
	fmt.Println()

	// Get first page of logs
	pageSize := 5
	lines, err := dk.GetPage(0, pageSize)
	if err != nil {
		log.Fatalf("Failed to get page: %v", err)
	}

	// Display logs
	fmt.Printf("=== First %d Logs ===\n", pageSize)
	for _, line := range lines {
		fmt.Println("\nLog:")
		for i, val := range line {
			if i < len(fields) {
				fmt.Printf("  %s: %s\n", fields[i].Name, val.String())
			}
		}
	}

	fmt.Println()
}

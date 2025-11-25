package main

import (
	"context"
	"fmt"
	"os"

	_ "github.com/marcboeker/go-duckdb"

	tea "charm.land/bubbletea/v2"
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

	logger := &simpleLogger{}
	dk, err := duck.New(logger)
	if err != nil {
		panic(err)
	}
	defer dk.Close()

	//logFile := "test/data/smar.log"
	logFile := "junk/tag2.log"
	if err := dk.Load(logFile, 0); err != nil {
		panic(err)
	}

	if err := dk.SetView(parcours.Filter{}, nil); err != nil {
		panic(err)
	}

	model := parcours.NewModel(dk)
	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

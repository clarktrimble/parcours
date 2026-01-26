package main

import (
	"context"
	"fmt"
	"os"

	"github.com/clarktrimble/sabot"
	_ "github.com/marcboeker/go-duckdb"

	"parcours"
	"parcours/store/duck"
	"parcours/util"
)

const (
	logFile string      = "parcours.log"
	logMax  int         = 999
	mode    os.FileMode = 0600
)

func main() {

	// setup logging

	logCfg := &sabot.Config{MaxLen: logMax, EnableDebug: true} // Todo: cfg
	file := util.OpenLog(logFile, mode)

	lgr := logCfg.New(file)
	ctx := context.Background()

	dk, err := duck.New(lgr)
	if err != nil {
		panic(err)
	}
	defer dk.Close()

	// Todo: dont panic

	cfg := &parcours.Config{}
	app := cfg.New(ctx, dk, lgr)

	err = app.Run(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

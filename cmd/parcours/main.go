package main

import (
	"context"
	"os"

	"github.com/clarktrimble/hondo"
	"github.com/clarktrimble/launch"
	"github.com/clarktrimble/sabot"
	_ "github.com/marcboeker/go-duckdb"

	"parcours"
	"parcours/store/duck"
	"parcours/util"
)

const (
	logFile string      = "parcours.log"
	mode    os.FileMode = 0600

	appId     string = "pcr"
	cfgPrefix string = "pcr"
	blerb     string = "'parcours' is a log viewer"
)

var (
	version string
	release string
)

type config struct {
	Version  string           `json:"version" ignored:"true"`
	Release  string           `json:"release" ignored:"true"`
	Logger   *sabot.Config    `json:"logger"`
	Parcours *parcours.Config `json:"parcours"`
}

func main() {

	// load config and setup logger

	cfg := &config{Version: version, Release: release}
	launch.Load(cfg, cfgPrefix, blerb)

	file := util.Open(logFile, mode) // Todo: sabot wants?
	defer util.Close(file)

	lgr := cfg.Logger.New(file)
	runId := hondo.Rand(7)
	ctx := lgr.WithFields(context.Background(), "app_id", appId, "run_id", runId)
	lgr.Info(ctx, "loaded", "config", cfg)

	// setup store and run the app

	dk, err := duck.New(lgr)
	launch.Check(ctx, lgr, err)
	defer dk.Close()

	app := cfg.Parcours.New(dk, lgr)
	err = app.Run(ctx)
	launch.Check(ctx, lgr, err)
}

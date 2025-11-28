package main

import (
	"context"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/clarktrimble/sabot"
	_ "github.com/marcboeker/go-duckdb"

	"parcours"
	"parcours/store/duck"
	"parcours/util"
)

const (
	logFile string = "parcours.log"
	logMax  int    = 999
	//cfgFile string      = "powercycle.yml"
	mode os.FileMode = 0600
)

func main() {

	// setup logging

	logCfg := &sabot.Config{MaxLen: logMax, EnableDebug: true} // Todo: cfg
	file := util.OpenLog(logFile, mode)

	lgr := logCfg.New(file)
	ctx := context.Background()

	// load cfg

	//cfg := &Config{Version: version, Release: release, PowerCycle: &powercycle.Config{}}
	//cfgErr := util.LoadConfig(cfg, cfgFile)
	//lgr.Info(ctx, "starting", "cfg", cfg)

	dk, err := duck.New(lgr)
	if err != nil {
		panic(err)
	}
	defer dk.Close()

	//logFile := "test/data/smar.log"
	logFile := "junk/tag2.log"

	// Todo: dont panic

	err = dk.Load(logFile, 0)
	if err != nil {
		panic(err)
	}

	//err = dk.SetView(parcours.Filter{}, nil)
	//if err != nil {
	//panic(err)
	//}

	model, err := parcours.NewModel(ctx, dk, lgr)
	if err != nil {
		panic(err)
	}

	_, err = tea.NewProgram(model).Run()
	if err != nil {
		panic(err)
	}
}

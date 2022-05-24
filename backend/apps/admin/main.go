package main

import (
	"fmt"
	"log"
	"os"

	"github.com/trezcool/masomo/core"
	"github.com/trezcool/masomo/services/logger"
	"github.com/trezcool/masomo/storage/database"
	"github.com/trezcool/masomo/storage/database/sqlboiler"
)

func main() {
	defer os.Exit(0)

	conf := core.NewConfig()

	// set up logger
	stdLogger := log.New(os.Stdout, "ADMIN : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	logger := logsvc.NewRollbarLogger(stdLogger, conf)
	logger.Enable(!conf.Debug)

	// set up DB
	db, err := database.Open(conf)
	if err != nil {
		logger.Fatal(fmt.Sprintf("opening database: %v", err), err)
	}
	defer func() { _ = db.Close() }()

	// start CLI
	cli := commandLine{
		db:      db,
		conf:    conf,
		usrRepo: boiledrepos.NewUserRepository(db),
	}
	if err = cli.run(os.Args); err != nil {
		if err != errHelp {
			logger.Info(fmt.Sprintf("\nerror: %v", err), err)
		}
		os.Exit(1)
	}
}

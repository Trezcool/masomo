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

	// set up logger
	stdLogger := log.New(os.Stdout, "ADMIN : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	logger := logsvc.NewRollbarLogger(stdLogger)
	logger.SetEnabled(!core.Conf.Debug)

	// set up DB
	db, err := database.Open()
	if err != nil {
		logger.Fatal(fmt.Sprintf("opening database: %v", err), err)
	}
	defer db.Close()
	if db.Ping() != nil {
		logger.Fatal(fmt.Sprintf("pinging database: %v", err), err)
	}

	// start CLI
	cli := commandLine{
		db:      db,
		usrRepo: boiledrepos.NewUserRepository(db),
	}
	if err = cli.run(os.Args); err != nil {
		if err != errHelp {
			logger.Info(fmt.Sprintf("\nerror: %v", err), err)
		}
		os.Exit(1)
	}
}

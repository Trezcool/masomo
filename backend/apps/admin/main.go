package main

import (
	"log"
	"os"

	"github.com/trezcool/masomo/storage/database"
	"github.com/trezcool/masomo/storage/database/sqlboiler"
)

var logger *log.Logger // todo: logger service

func main() {
	defer os.Exit(0)

	logger = log.New(os.Stdout, "ADMIN : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	// set up DB
	db, err := database.Open()
	errAndDie(err)
	defer db.Close()
	errAndDie(db.Ping())

	// start CLI
	cli := commandLine{
		db:      db,
		usrRepo: boiledrepos.NewUserRepository(db),
	}
	if err := cli.run(os.Args); err != nil {
		if err != errHelp {
			logger.Printf("\nerror: %s\n", err)
		}
		os.Exit(1)
	}
}

func errAndDie(err error) {
	if err != nil {
		logger.Fatal(err)
	}
}

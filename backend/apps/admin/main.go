package main

import (
	"log"
	"os"

	"github.com/trezcool/masomo/backend/core"
	"github.com/trezcool/masomo/backend/core/user"
	"github.com/trezcool/masomo/backend/services/email/dummy"
	"github.com/trezcool/masomo/backend/services/email/sendgrid"
	"github.com/trezcool/masomo/backend/storage/database/dummy"
)

var logger *log.Logger // todo: logger service

func main() {
	defer os.Exit(0)

	debug := core.Conf.GetBool("debug")
	logger = log.New(os.Stdout, "ADMIN : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	// set up DB
	db, err := dummydb.Open()
	errAndDie(err)

	// set up mail service
	var mailSvc core.EmailService
	if debug {
		mailSvc = dummymail.NewService()
	} else {
		mailSvc = sendgridmail.NewService()
	}

	// set up services
	usrSvc := user.NewService(dummydb.NewUserRepository(db), mailSvc)

	// start CLI
	cli := commandLine{
		usrSvc: usrSvc,
	}
	if err := cli.run(os.Args); err != nil {
		if err != errHelp {
			logger.Printf("error: %s", err)
		}
		os.Exit(1)
	}
}

func errAndDie(err error) {
	if err != nil {
		logger.Fatal(err)
	}
}

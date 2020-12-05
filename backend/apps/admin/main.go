package main

import (
	"log"
	"os"
	"time"

	_ "github.com/trezcool/masomo/backend/core"
	"github.com/trezcool/masomo/backend/core/user"
	"github.com/trezcool/masomo/backend/services/email/dummy"
	"github.com/trezcool/masomo/backend/storage/database/dummy"
)

var logger *log.Logger // todo: logger service

func main() {
	defer os.Exit(0)

	logger = log.New(os.Stdout, "ADMIN : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	// todo: load from config
	appName := "Masomo"
	secretKey := []byte("secret")
	serverName := "localhost" // default
	defaultFromEmail := "noreply@" + serverName
	//sendgridApiKey := "${SENDGRID_API_KEY}"
	passwordResetTimeoutDelta := 3 * 24 * time.Hour

	// set up DB
	db, err := dummydb.Open()
	errAndDie(err)

	// set up mail service
	//mailSvc := sendgridmail.NewService(sendgridApiKey, appName, defaultFromEmail)
	mailSvc := dummymail.NewService(appName, defaultFromEmail) // todo: only during dev (config)

	// set up services
	usrSvc := user.NewService(dummydb.NewUserRepository(db), mailSvc, secretKey, passwordResetTimeoutDelta)

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

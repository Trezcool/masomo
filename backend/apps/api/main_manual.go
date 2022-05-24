package main

import (
	"context"
	"database/sql"
	"expvar"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	echoapi "github.com/trezcool/masomo/apps/api/echo"
	"github.com/trezcool/masomo/core"
	"github.com/trezcool/masomo/core/user"
	emailsvc "github.com/trezcool/masomo/services/email"
	logsvc "github.com/trezcool/masomo/services/logger"
	"github.com/trezcool/masomo/storage/database"
	boiledrepos "github.com/trezcool/masomo/storage/database/sqlboiler"
)

func startManual() {
	// =========================================================================
	// Set up Dependencies

	conf := core.NewConfig()

	// set up loggers
	logger := logsvc.NewRollbarLogger(
		log.New(os.Stdout, "API : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile),
		conf,
	)
	logger.Enable(!conf.Debug)

	dbLogger := logsvc.NewRollbarLogger(
		log.New(os.Stdout, "DB : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile),
		conf,
	)
	dbLogger.Enable(!conf.Debug)

	// set up DB
	db, err := setUpDB(conf)
	if err != nil {
		logger.Fatal(fmt.Sprintf("setting up database: %v", err), err)
	}
	defer func() {
		if err = db.Close(); err != nil {
			dbLogger.Fatal("Failed to close", err)
		}
	}()

	// set up services
	var mailSvc core.EmailService
	if conf.Debug {
		mailSvc = emailsvc.NewConsoleService(conf)
	} else {
		mailSvc = emailsvc.NewSendgridService(conf, logger)
	}
	usrSvc := user.NewService(db, boiledrepos.NewUserRepository(db), mailSvc, conf)

	// =========================================================================
	// Initialize App

	logger.Info(fmt.Sprintf("Application initializing : version %q", conf.Build))
	defer logger.Info("Application stopped")

	validate := validator.New()
	translator := newTranslator()
	core.InitValidators(validate, translator)
	user.InitValidators(validate, translator)

	core.ParseEmailTemplates(logger)

	user.LoadCommonPasswords(logger)

	// =========================================================================
	// Start Debug Service
	//
	// /debug/pprof - Added to the default mux by importing the net/http/pprof package.
	// /debug/vars - Added to the default mux by importing the expvar package.

	// Expose important info under /debug/vars.
	expvar.NewString("build").Set(conf.Build)
	expvar.NewString("env").Set(conf.Env)

	go func() {
		if err = http.ListenAndServe(conf.Server.DebugHost, http.DefaultServeMux); err != nil {
			logger.Error(fmt.Sprintf("debug server closed: %v", err), err)
		}
	}()

	// =========================================================================
	// Start API Service

	server := echoapi.NewServer(
		echoapi.ServerDeps{
			Conf:       conf,
			Logger:     logger,
			UserSvc:    usrSvc,
			Validate:   validate,
			Translator: translator,
		},
	)

	go func() {
		server.Start()
	}()

	// =========================================================================
	// Shutdown

	select {
	case err = <-server.Errors():
		logger.Fatal(fmt.Sprintf("server error: %v", err), err)

	case sig := <-server.ShutdownSignal():
		logger.Info(fmt.Sprintf("%v: Start shutdown...", sig))

		// give outstanding requests a deadline for completion
		ctx, cancel := context.WithTimeout(context.Background(), conf.Server.ShutdownTimeout)
		defer cancel()

		// asking listener to shutdown and shed load
		if err = server.Shutdown(ctx); err != nil {
			logger.Error(fmt.Sprintf("could not stop server gracefully: %v", err), err)

			if err = server.Close(); err != nil {
				logger.Fatal(fmt.Sprintf("could not force stop server: %v", err), err)
			}
		}
	}
}

func setUpDB(conf *core.Config) (*sql.DB, error) {
	if err := database.CreateIfNotExist(conf); err != nil {
		return nil, err
	}

	db, err := database.Open(conf)
	if err != nil {
		return nil, err
	}

	if err = database.Migrate(db); err != nil {
		return nil, err
	}
	return db, nil
}

func newTranslator() ut.Translator {
	_en := en.New()
	uni := ut.New(_en, _en)
	translator, _ := uni.GetTranslator("en")
	return translator
}

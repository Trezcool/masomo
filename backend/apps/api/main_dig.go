package main

import (
	"context"
	"database/sql"
	"expvar"
	"fmt"
	"log"
	"net/http"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	dig_container "github.com/trezcool/masomo/apps/api/di/dig"
	echoapi "github.com/trezcool/masomo/apps/api/echo"
	"github.com/trezcool/masomo/core"
	"github.com/trezcool/masomo/core/user"
)

func startWithDig() {
	c := dig_container.New()

	must(c.Invoke(func(
		conf *core.Config,
		apiLogger core.Logger,
		dbLoggerParam dig_container.DBLoggerParam,
		db *sql.DB,
		validate *validator.Validate,
		translator ut.Translator,
		server *echoapi.Server,
	) {
		// =========================================================================
		// Initialize App

		apiLogger.Info(fmt.Sprintf("Application initializing : version %q", conf.Build))

		core.InitValidators(validate, translator)
		user.InitValidators(validate, translator)

		core.ParseEmailTemplates(apiLogger)

		user.LoadCommonPasswords(apiLogger)

		dbLogger := dbLoggerParam.Logger
		defer func() {
			if err := db.Close(); err != nil {
				dbLogger.Fatal("Failed to close", err)
			}
		}()
		defer apiLogger.Info("Application stopped")

		// =========================================================================
		// Start Debug Service
		//
		// /debug/pprof - Added to the default mux by importing the net/http/pprof package.
		// /debug/vars - Added to the default mux by importing the expvar package.

		// Expose important info under /debug/vars.
		expvar.NewString("build").Set(conf.Build)
		expvar.NewString("env").Set(conf.Env)

		go func() {
			if err := http.ListenAndServe(conf.Server.DebugHost, http.DefaultServeMux); err != nil {
				apiLogger.Error(fmt.Sprintf("debug server closed: %v", err), err)
			}
		}()

		// =========================================================================
		// Start API Service

		go func() {
			server.Start()
		}()

		// =========================================================================
		// Shutdown

		select {
		case err := <-server.Errors():
			apiLogger.Fatal(fmt.Sprintf("server error: %v", err), err)

		case sig := <-server.ShutdownSignal():
			apiLogger.Info(fmt.Sprintf("%v: Start shutdown...", sig))

			// give outstanding requests a deadline for completion
			ctx, cancel := context.WithTimeout(context.Background(), conf.Server.ShutdownTimeout)
			defer cancel()

			// asking listener to shut down and shed load
			if err := server.Shutdown(ctx); err != nil {
				apiLogger.Error(fmt.Sprintf("could not stop server gracefully: %v", err), err)

				if err = server.Close(); err != nil {
					apiLogger.Fatal(fmt.Sprintf("could not force stop server: %v", err), err)
				}
			}
		}
	}))
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

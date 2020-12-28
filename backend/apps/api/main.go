package main

import (
	"context"
	"expvar"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/trezcool/masomo/apps/api/echo"
	"github.com/trezcool/masomo/core"
	"github.com/trezcool/masomo/core/user"
	"github.com/trezcool/masomo/services/email"
	"github.com/trezcool/masomo/services/logger"
	"github.com/trezcool/masomo/storage/database"
	"github.com/trezcool/masomo/storage/database/sqlboiler"
)

// TODO:
// - Profiling (Benchmarking) !! https://blog.golang.org/pprof
// - load test:
// - APM/Tracing: New Relic Free :)
// - CSRF !!!
// - Serve static files | Web Server ? (for mailers)
func main() {
	// =========================================================================
	// Initialize App & Set up Dependencies

	// set up logger
	stdLogger := log.New(os.Stdout, "API : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	logger := logsvc.NewRollbarLogger(stdLogger)
	logger.SetEnabled(!core.Conf.Debug)

	logger.Info(fmt.Sprintf("Application initializing : version %q", core.Conf.Build))
	defer logger.Info("Application stopped")

	// set up DB
	db, err := database.Open()
	if err != nil {
		logger.Fatal(fmt.Sprintf("opening database: %v", err), err)
	}
	defer db.Close()

	// set up services
	var mailSvc core.EmailService
	if core.Conf.Debug {
		mailSvc = emailsvc.NewConsoleService()
	} else {
		mailSvc = emailsvc.NewSendgridService()
	}
	usrSvc := user.NewService(db, boiledrepos.NewUserRepository(db), mailSvc)

	// =========================================================================
	// Start Debug Service
	//
	// /debug/pprof - Added to the default mux by importing the net/http/pprof package.
	// /debug/vars - Added to the default mux by importing the expvar package.

	// Expose important info under /debug/vars.
	expvar.NewString("build").Set(core.Conf.Build)
	expvar.NewString("env").Set(core.Conf.Env)

	go func() {
		if err = http.ListenAndServe(core.Conf.Server.DebugHost, http.DefaultServeMux); err != nil {
			logger.Error(fmt.Sprintf("debug server closed: %v", err), err)
		}
	}()

	// =========================================================================
	// Start API Service

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	app := echoapi.NewServer(
		shutdown,
		&echoapi.Deps{
			Logger:  logger,
			UserSvc: usrSvc,
		},
	)

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- app.Start()
	}()

	// shutdown
	select {
	case err = <-serverErrors:
		logger.Fatal(fmt.Sprintf("server error: %v", err), err)

	case sig := <-shutdown:
		logger.Info(fmt.Sprintf("%v: Start shutdown...", sig))

		// give outstanding requests a deadline for completion
		ctx, cancel := context.WithTimeout(context.Background(), core.Conf.Server.ShutdownTimeout)
		defer cancel()

		// asking listener to shutdown and shed load
		if err = app.Shutdown(ctx); err != nil {
			logger.Error(fmt.Sprintf("could not stop server gracefully: %v", err), err)

			if err = app.Close(); err != nil {
				logger.Fatal(fmt.Sprintf("could not force stop server: %v", err), err)
			}
		}
	}
}

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/trezcool/masomo/apps/api/echo"
	"github.com/trezcool/masomo/core"
	"github.com/trezcool/masomo/core/user"
	"github.com/trezcool/masomo/services/email"
	"github.com/trezcool/masomo/storage/database"
	"github.com/trezcool/masomo/storage/database/sqlboiler"
)

var logger *log.Logger // todo: logger service

// TODO:
// - DB & Configs Singleton accessible apis !!!
// - graceful shutdown
// - Profiling (Benchmarking) !! https://blog.golang.org/pprof
// - load test:
// - APM/Tracing: New Relic Free :)
// - Logging: Rollbar!!! | Sentry | LogRocket
// - CSRF !!!
// - Serve static files | Web Server ? (for mailers)
func main() {
	logger = log.New(os.Stdout, "API : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	// set up DB
	db, err := database.Open()
	errAndDie(err, "opening database: %+v")
	defer db.Close()

	// set up services
	var mailSvc core.EmailService
	if core.Conf.Debug {
		mailSvc = emailsvc.NewConsoleService()
	} else {
		mailSvc = emailsvc.NewSendgridService()
	}
	usrSvc := user.NewService(db, boiledrepos.NewUserRepository(db), mailSvc)

	// start API server
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	app := echoapi.NewServer(
		":8000",
		shutdown,
		&echoapi.Deps{
			UserSvc: usrSvc,
		},
	)

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- app.Start()
	}()

	// shutdown
	select {
	case err := <-serverErrors:
		logger.Fatalf("server error: %+v", err)

	case sig := <-shutdown:
		logger.Printf("%v: Start shutdown...", sig)

		// give outstanding requests a deadline for completion
		ctx, cancel := context.WithTimeout(context.Background(), core.Conf.Server.ShutdownTimeout)
		defer cancel()

		// asking listener to shutdown and shed load
		if err := app.Shutdown(ctx); err != nil {
			logger.Printf("could not stop server gracefully: %+v", err)

			errAndDie(app.Close(), "could not force stop server: %+v")
		}
	}
}

func errAndDie(err error, msg string) {
	if err != nil {
		logger.Fatalf(msg, err)
	}
}

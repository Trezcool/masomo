package main

import (
	"log"

	"github.com/trezcool/masomo/apps/api/echo"
	"github.com/trezcool/masomo/core"
	"github.com/trezcool/masomo/core/user"
	"github.com/trezcool/masomo/services/email"
	"github.com/trezcool/masomo/storage/database"
	"github.com/trezcool/masomo/storage/database/sqlboiler"
)

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
	// set up DB
	db, err := database.Open()
	errAndDie(err)
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
	app := echoapi.NewServer(
		&echoapi.Options{
			Addr:    ":8000",
			UserSvc: usrSvc,
		},
	)
	app.Start()
}

func errAndDie(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

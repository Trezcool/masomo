package main

import (
	"log"

	"github.com/trezcool/masomo/backend/apps/api/echo"
	"github.com/trezcool/masomo/backend/core"
	"github.com/trezcool/masomo/backend/core/user"
	"github.com/trezcool/masomo/backend/services/email/dummy"
	"github.com/trezcool/masomo/backend/services/email/sendgrid"
	"github.com/trezcool/masomo/backend/storage/database/dummy"
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
	debug := core.Conf.GetBool("debug")

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

	// TODO: move to script | SQL data migration (dev only?)
	root := user.NewUser{
		Name:     "Root",
		Username: "root",
		Email:    "root@masomo.cd",
		Password: "LolC@t123",
		Roles:    user.AllRoles,
	}
	_, _ = usrSvc.Create(root)

	// start API server
	app := echoapi.NewServer(
		&echoapi.Options{
			Address: ":8000",
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

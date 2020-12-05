package main

import (
	"log"
	"time"

	"github.com/trezcool/masomo/backend/apps/api/echo"
	_ "github.com/trezcool/masomo/backend/core"
	"github.com/trezcool/masomo/backend/core/user"
	"github.com/trezcool/masomo/backend/services/email/dummy"
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
	// todo: load from config
	debug := true
	appName := "Masomo"
	secretKey := []byte("secret")
	serverName := "localhost" // default
	defaultFromEmail := "noreply@" + serverName
	//sendgridApiKey := "${SENDGRID_API_KEY}"
	jwtExpirationDelta := 10 * time.Minute // todo: dev - 7 days
	jwtRefreshExpirationDelta := 4 * time.Hour
	passwordResetTimeoutDelta := 3 * 24 * time.Hour

	// set up DB
	db, err := dummydb.Open()
	errAndDie(err)

	// set up mail service
	//mailSvc := sendgridmail.NewService(sendgridApiKey, appName, defaultFromEmail)
	mailSvc := dummymail.NewService(appName, defaultFromEmail) // todo: only during dev (config)

	// set up services
	usrSvc := user.NewService(dummydb.NewUserRepository(db), mailSvc, secretKey, passwordResetTimeoutDelta)

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
			Address:                   ":8000",
			Debug:                     debug,
			AppName:                   appName,
			SecretKey:                 secretKey,
			JwtExpirationDelta:        jwtExpirationDelta,
			JwtRefreshExpirationDelta: jwtRefreshExpirationDelta,
			UserSvc:                   usrSvc,
		},
	)
	app.Start()
}

func errAndDie(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

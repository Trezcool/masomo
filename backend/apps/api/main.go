package main

import (
	"github.com/trezcool/masomo/backend/apps/api/echo"
	_ "github.com/trezcool/masomo/backend/core"
	"github.com/trezcool/masomo/backend/core/user"
	"github.com/trezcool/masomo/backend/storage/database/dummy"
)

// TODO: DB & Configs Singleton accessible apis !!!
// TODO: graceful shutdown
// TODO: Profiling (Benchmarking) !! https://blog.golang.org/pprof
// TODO: load test:
// TODO: APM/Tracing: New Relic Free :)
// TODO: Logging: Rollbar!!! | Sentry | LogRocket
func main() {
	// set up DB
	db, err := dummydb.Open()
	errAndDie(err)

	// set up services
	usrSvc := user.NewService(dummydb.NewUserRepository(db))

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
	app := echoapi.NewServer(":8080", usrSvc)
	app.Start()
}

func errAndDie(err error) { // TODO: log.Fatal and return instead
	if err != nil {
		panic(err)
	}
}

/*
[{"id":1,"name":"User1","username":"awe","email":"awe@test.cd","is_active":true,"roles":null,"created_at":"2020-11-24T18:28:47.282087Z","updated_at":"2020-11-24T18:28:47.282087Z"},{"id":2,"name":"Admin1","username":"admin1","email":"admin1@test.cd","is_active":true,"roles":["admin:"],"created_at":"2020-11-24T18:28:47.394444Z","updated_at":"2020-11-24T18:28:47.394444Z"}]
[{"id":1,"name":"User1","username":"awe","email":"awe@test.cd","is_active":true,"roles":null,"created_at":"2020-11-24T18:28:47.282087Z","updated_at":"2020-11-24T18:28:47.282087Z"},{"id":2,"name":"Admin1","username":"admin1","email":"admin1@test.cd","is_active":true,"roles":["admin:"],"created_at":"2020-11-24T18:28:47.394444Z","updated_at":"2020-11-24T18:28:47.394444Z"}]
*/

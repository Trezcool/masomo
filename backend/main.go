package main

import (
	"github.com/trezcool/masomo/backend/api"
	_ "github.com/trezcool/masomo/backend/apps/utils"
)

// TODO: graceful shutdown
// TODO: load test:
// TODO: APM/Tracing: New Relic Free :)
// TODO: Logging: Rollbar!!! | Sentry | LogRocket
func main() {
	app := api.NewServer(":8080")
	app.Start()
}

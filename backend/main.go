package main

import (
	"github.com/trezcool/masomo/backend/api"
	_ "github.com/trezcool/masomo/backend/apps/shared"
)

func main() {
	app := api.NewServer(":8080")
	app.Start()
}

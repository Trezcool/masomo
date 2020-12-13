package main

import (
	"path/filepath"

	"github.com/pressly/goose"

	"github.com/trezcool/masomo/core"
)

var (
	gooseRunFunc  = goose.Run // mockable
	migrationsDir = filepath.Join(core.Conf.WorkDir, "storage", "database", "migrations")
)

func (cli *commandLine) migrate(args []string) error {
	arguments := make([]string, 0)
	if len(args) > 1 {
		arguments = append(arguments, args[1:]...)
	}
	return gooseRunFunc(args[0], cli.db, migrationsDir, arguments...)
}

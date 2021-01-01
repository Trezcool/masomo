package main

import (
	"github.com/trezcool/goose"

	"github.com/trezcool/masomo/fs"
)

var gooseRunFunc = goose.RunFS // mockable

func (cli *commandLine) migrate(args []string) error {
	arguments := make([]string, 0)
	if len(args) > 1 {
		arguments = append(arguments, args[1:]...)
	}
	return gooseRunFunc(args[0], cli.db, appfs.FS, "migrations", arguments...)
}

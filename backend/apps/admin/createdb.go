package main

import "github.com/trezcool/masomo/storage/database"

func (cli *commandLine) createDB() error {
	return database.Create()
}

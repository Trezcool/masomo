package goose

import (
	"database/sql"
	"io/fs"
)

// Redo rolls back the most recently applied migration, then runs it again.
func Redo(db *sql.DB, fsys fs.FS, dir string) error {
	currentVersion, err := GetDBVersion(db)
	if err != nil {
		return err
	}

	migrations, err := CollectMigrations(fsys, dir, minVersion, maxVersion)
	if err != nil {
		return err
	}

	current, err := migrations.Current(currentVersion)
	if err != nil {
		return err
	}

	if err := current.Down(db, fsys); err != nil {
		return err
	}

	if err := current.Up(db, fsys); err != nil {
		return err
	}

	return nil
}

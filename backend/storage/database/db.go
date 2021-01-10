package database

import (
	"database/sql"
	"fmt"
	"net/url"
	"time"

	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/trezcool/goose"

	"github.com/trezcool/masomo/core"
	"github.com/trezcool/masomo/fs"
)

func open(dbName string, admin bool) (*sql.DB, error) {
	user := url.UserPassword(core.Conf.Database.User, core.Conf.Database.Password)
	if admin && core.Conf.Database.AdminUser != "" {
		user = url.UserPassword(core.Conf.Database.AdminUser, core.Conf.Database.AdminPassword)
	}

	sslMode := "require"
	if core.Conf.Database.DisableTLS {
		sslMode = "disable"
	}
	q := make(url.Values)
	q.Set("sslmode", sslMode)
	q.Set("timezone", "utc")

	u := url.URL{
		Scheme:   core.Conf.Database.Engine,
		User:     user,
		Host:     core.Conf.Database.Address(),
		Path:     dbName,
		RawQuery: q.Encode(),
	}
	return sql.Open(core.Conf.Database.Engine, u.String())
}

func Open() (*sql.DB, error) {
	return open(core.Conf.Database.Name, false)
}

// ping waits for the database to be ready. Waits 100ms longer between each attempt.
func ping(db *sql.DB) error {
	var err error
	maxAttempts := 30
	for attempts := 1; attempts <= maxAttempts; attempts++ {
		err = db.Ping()
		if err == nil {
			break
		}
		time.Sleep(time.Duration(attempts) * 100 * time.Millisecond)
	}

	if err != nil {
		return errors.Wrap(err, "DB ping timeout")
	}
	return nil
}

func createAppUser(db *sql.DB) error {
	if core.Conf.Database.User == "" {
		return nil
	}

	// check if app user exists
	var exists bool
	rows, err := db.Query(fmt.Sprintf("SELECT true FROM pg_roles WHERE rolname='%s'", core.Conf.Database.User))
	if err != nil {
		return errors.Wrap(err, "checking app user")
	}
	defer rows.Close()
	for rows.Next() {
		if err = rows.Scan(&exists); err != nil {
			return errors.Wrap(err, "checking app user")
		}
	}
	if err = rows.Err(); err != nil {
		return errors.Wrap(err, "checking app user")
	}

	// create app user if not exist
	if !exists {
		q := fmt.Sprintf("CREATE USER %s CREATEDB ENCRYPTED PASSWORD '%s'", core.Conf.Database.User, core.Conf.Database.Password)
		if _, err = db.Exec(q); err != nil {
			return errors.Wrap(err, "creating app user")
		}
	}
	return nil
}

func createDB(db *sql.DB) error {
	// check if DB exists
	var exists bool
	rows, err := db.Query(fmt.Sprintf("SELECT true FROM pg_database WHERE datname='%s'", core.Conf.Database.Name))
	if err != nil {
		return errors.Wrap(err, "checking DB")
	}
	defer rows.Close()
	for rows.Next() {
		if err = rows.Scan(&exists); err != nil {
			return errors.Wrap(err, "checking DB")
		}
	}
	if err = rows.Err(); err != nil {
		return errors.Wrap(err, "checking DB")
	}

	// create DB if not exist
	if !exists {
		if _, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", core.Conf.Database.Name)); err != nil {
			return errors.Wrap(err, "creating database")
		}
	}
	return nil
}

func Create() error {
	// connect as admin
	db, err := open("postgres", true)
	if err != nil {
		return errors.Wrap(err, "opening database")
	}

	if err = ping(db); err != nil {
		return errors.Wrap(err, "pinging database")
	}

	if err = createAppUser(db); err != nil {
		return errors.Wrap(err, "creating app user")
	}
	db.Close()

	// create DB as app user
	db, err = open("postgres", false)
	if err != nil {
		return errors.Wrap(err, "opening database")
	}
	if err = createDB(db); err != nil {
		return errors.Wrap(err, "creating database")
	}
	db.Close()
	return nil
}

func Migrate(db *sql.DB) error {
	if err := goose.RunFS("up", db, appfs.FS, "migrations"); err != nil {
		return errors.Wrap(err, "migrating database")
	}
	return nil
}

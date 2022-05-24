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

func open(dbName string, admin bool, conf *core.Config) (*sql.DB, error) {
	user := url.UserPassword(conf.Database.User, conf.Database.Password)
	if admin && conf.Database.AdminUser != "" {
		user = url.UserPassword(conf.Database.AdminUser, conf.Database.AdminPassword)
	}

	sslMode := "require"
	if conf.Database.DisableTLS {
		sslMode = "disable"
	}
	q := make(url.Values)
	q.Set("sslmode", sslMode)
	q.Set("timezone", "utc")

	u := url.URL{
		Scheme:   conf.Database.Engine,
		User:     user,
		Host:     conf.Database.Address(),
		Path:     dbName,
		RawQuery: q.Encode(),
	}
	return sql.Open(conf.Database.Engine, u.String())
}

func Open(conf *core.Config) (*sql.DB, error) {
	return open(conf.Database.Name, false, conf)
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

func createAppUser(db *sql.DB, conf *core.Config) error {
	if conf.Database.User == "" {
		return nil
	}

	// check if app user exists
	var exists bool
	rows, err := db.Query(fmt.Sprintf("SELECT true FROM pg_roles WHERE rolname='%s'", conf.Database.User))
	if err != nil {
		return errors.Wrap(err, "checking app user")
	}
	defer func() { _ = rows.Close() }()
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
		q := fmt.Sprintf("CREATE USER %s CREATEDB ENCRYPTED PASSWORD '%s'", conf.Database.User, conf.Database.Password)
		if _, err = db.Exec(q); err != nil {
			return errors.Wrap(err, "creating app user")
		}
	}
	return nil
}

func createDB(db *sql.DB, conf *core.Config) error {
	// check if DB exists
	var exists bool
	rows, err := db.Query(fmt.Sprintf("SELECT true FROM pg_database WHERE datname='%s'", conf.Database.Name))
	if err != nil {
		return errors.Wrap(err, "checking DB")
	}
	defer func() { _ = rows.Close() }()
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
		if _, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", conf.Database.Name)); err != nil {
			return errors.Wrap(err, "creating database")
		}
	}
	return nil
}

func CreateIfNotExist(conf *core.Config) error {
	// connect as admin
	db, err := open("postgres", true, conf)
	if err != nil {
		return errors.Wrap(err, "opening database")
	}

	if err = ping(db); err != nil {
		return errors.Wrap(err, "pinging database")
	}

	if err = createAppUser(db, conf); err != nil {
		return errors.Wrap(err, "creating app user")
	}
	defer func() { _ = db.Close() }()

	// create DB as app user
	db, err = open("postgres", false, conf)
	if err != nil {
		return errors.Wrap(err, "opening database")
	}
	if err = createDB(db, conf); err != nil {
		return errors.Wrap(err, "creating database")
	}
	defer func() { _ = db.Close() }()
	return nil
}

func Migrate(db *sql.DB) error {
	if err := goose.RunFS("up", db, appfs.FS, "migrations"); err != nil {
		return errors.Wrap(err, "migrating database")
	}
	return nil
}

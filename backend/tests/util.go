package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/pressly/goose"

	"github.com/trezcool/masomo/core"
	"github.com/trezcool/masomo/core/user"
	"github.com/trezcool/masomo/storage/database"
)

var (
	testDBRegex       = regexp.MustCompile("(?i)test")
	migrationsDir     = filepath.Join(core.Conf.WorkDir, "storage", "database", "migrations")
	truncateTablesSQL = `
DO
$func$
BEGIN
   EXECUTE
   (SELECT 'TRUNCATE TABLE ' || string_agg(oid::regclass::text, ', ') || ' CASCADE'
    FROM   pg_class
    WHERE  relkind = 'r'  -- only tables
    AND    relnamespace = 'public'::regnamespace
    AND    relname <> 'goose_db_version' -- exclude migrations table
   );
END
$func$;
`
)

func OpenDB() *sql.DB {
	db, err := database.Open()
	if err != nil {
		fmt.Printf("OpenDB: %v", err)
		os.Exit(1)
	}
	if err = db.Ping(); err != nil {
		fmt.Printf("db.Ping(): %v", err)
		os.Exit(1)
	}

	// ensure db is a test DB
	var dbName string
	if err = db.QueryRow("SELECT current_database()").Scan(&dbName); err != nil {
		fmt.Printf("\"SELECT current_database()\": %v", err)
		os.Exit(1)
	}
	if !testDBRegex.MatchString(dbName) {
		fmt.Printf("%s is not a test DB", dbName)
		os.Exit(1)
	}
	return db
}

func ResetDB(t *testing.T, db *sql.DB) {
	// migrate
	if err := goose.Run("up", db, migrationsDir); err != nil {
		t.Fatalf("ResetDB: migrate up: %v", err)
	}
	// truncate
	if _, err := db.Exec(truncateTablesSQL); err != nil {
		t.Fatalf("ResetDB: truncate tables: %v", err)
	}
}

func CreateUser(
	t *testing.T,
	repo user.Repository,
	name, uname, email, pwd string,
	roles []string,
	isActive bool,
	createdAt ...time.Time,
) user.User {
	tstamp := time.Now().UTC().Truncate(time.Microsecond)
	if len(createdAt) > 0 {
		tstamp = createdAt[0].UTC().Truncate(time.Microsecond)
	}
	usr := user.User{
		Name:      name,
		Username:  uname,
		Email:     email,
		Roles:     roles,
		CreatedAt: tstamp,
		UpdatedAt: tstamp,
	}
	usr.SetActive(isActive)
	if pwd != "" {
		if err := usr.SetPassword(pwd); err != nil {
			t.Fatalf("CreateUser: usr.SetPassword(): %v", err)
		}
	}
	usr, err := repo.CreateUser(context.Background(), usr)
	if err != nil {
		t.Fatalf("CreateUser: repo.CreateUser(): %v", err)
	}
	return usr
}

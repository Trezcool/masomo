package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/trezcool/goose"
	"github.com/trezcool/masomo/core"

	"github.com/trezcool/masomo/core/user"
	"github.com/trezcool/masomo/fs"
	"github.com/trezcool/masomo/storage/database"
)

var (
	testDBRegex       = regexp.MustCompile("(?i)test")
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

func OpenDB(conf *core.Config) *sql.DB {
	if err := database.CreateIfNotExist(conf); err != nil {
		fmt.Printf("creating DB: %v", err)
		os.Exit(1)
	}
	db, err := database.Open(conf)
	if err != nil {
		fmt.Printf("opening DB: %v", err)
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
	if err := goose.RunFS("up", db, appfs.FS, "migrations"); err != nil {
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
	tstamp := time.Now()
	if len(createdAt) > 0 {
		tstamp = createdAt[0]
	}
	tstamp = tstamp.UTC().Truncate(time.Microsecond)
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

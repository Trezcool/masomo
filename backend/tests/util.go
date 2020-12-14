package testutil

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/pressly/goose"

	"github.com/trezcool/masomo/core"
	"github.com/trezcool/masomo/core/user"
	"github.com/trezcool/masomo/storage/database"
)

var (
	migrationsDir = filepath.Join(core.Conf.WorkDir, "storage", "database", "migrations")

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

func PrepareDB(t *testing.T) *sql.DB {
	db, err := database.Open()
	if err != nil {
		t.Fatalf("PrepareDB: db.Open(): %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("PrepareDB: db.Ping(): %v", err)
	}

	dbMigrateAndTruncate(t, db)
	return db
}

func dbMigrateAndTruncate(t *testing.T, db *sql.DB) {
	if err := goose.Run("up", db, migrationsDir); err != nil {
		t.Fatalf("PrepareDB: migrate up: %v", err)
	}
	if _, err := db.Exec(truncateTablesSQL); err != nil {
		t.Fatalf("PrepareDB: truncate tables: %v", err)
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
	tstamp := time.Now().UTC()
	if len(createdAt) > 0 {
		tstamp = createdAt[0].UTC()
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

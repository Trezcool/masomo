package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"testing"

	"github.com/trezcool/masomo/core/user"
	"github.com/trezcool/masomo/storage/database/sqlboiler"
	"github.com/trezcool/masomo/tests"
)

var usrRepo user.Repository

func setup(t *testing.T) *commandLine {
	// set up DB & repos
	db := testutil.PrepareDB(t)
	usrRepo = boiledrepos.NewUserRepository(db)

	// start CLI
	return &commandLine{
		db:      db,
		usrRepo: usrRepo,
	}
}

type cliTest struct {
	name       string
	args       []string // without program name
	wantErr    error
	wantErrStr string
	extra      interface{}
}

func Test_commandLine_migrate(t *testing.T) {
	cli := setup(t)

	gooseRunFunc = func(command string, db *sql.DB, dir string, args ...string) error {
		switch command {
		case "up", "up-by-one", "down", "fix", "redo", "reset", "status", "version": // pass
		case "up-to":
			if len(args) == 0 {
				return fmt.Errorf("up-to must be of form: goose [OPTIONS] DRIVER DBSTRING up-to VERSION")
			}
			if _, err := strconv.ParseInt(args[0], 10, 64); err != nil {
				return fmt.Errorf("version must be a number (got '%s')", args[0])
			}
		case "create":
			if len(args) == 0 {
				return fmt.Errorf("create must be of form: goose [OPTIONS] DRIVER DBSTRING create NAME [go|sql]")
			}
		case "down-to":
			if len(args) == 0 {
				return fmt.Errorf("down-to must be of form: goose [OPTIONS] DRIVER DBSTRING down-to VERSION")
			}
			if _, err := strconv.ParseInt(args[0], 10, 64); err != nil {
				return fmt.Errorf("version must be a number (got '%s')", args[0])
			}
		default:
			return fmt.Errorf("%q: no such command", command)
		}
		return nil
	}

	tests := []cliTest{
		{name: "no subcommand", args: []string{"migrate"}, wantErr: errHelp},
		{name: "unknown subcommand", args: []string{"migrate", "lol"}, wantErrStr: "\"lol\": no such command"},
		{name: "up-to: no args", args: []string{"migrate", "up-to"}, wantErrStr: "up-to must be of form: goose [OPTIONS] DRIVER DBSTRING up-to VERSION"},
		{name: "up-to: non-int arg", args: []string{"migrate", "up-to", "lol"}, wantErrStr: "version must be a number (got 'lol')"},
		{name: "create: no args", args: []string{"migrate", "create"}, wantErrStr: "create must be of form: goose [OPTIONS] DRIVER DBSTRING create NAME [go|sql]"},
		{name: "down-to: no args", args: []string{"migrate", "down-to"}, wantErrStr: "down-to must be of form: goose [OPTIONS] DRIVER DBSTRING down-to VERSION"},
		{name: "down-to: non-int arg", args: []string{"migrate", "down-to", "lol"}, wantErrStr: "version must be a number (got 'lol')"},
		{name: "up", args: []string{"migrate", "up"}},
		{name: "up-by-one", args: []string{"migrate", "up-by-one"}},
		{name: "up-to", args: []string{"migrate", "up-to", "2"}},
		{name: "down", args: []string{"migrate", "down"}},
		{name: "down-to", args: []string{"migrate", "down-to", "1"}},
		{name: "redo", args: []string{"migrate", "redo"}},
		{name: "reset", args: []string{"migrate", "reset"}},
		{name: "status", args: []string{"migrate", "status"}},
		{name: "version", args: []string{"migrate", "version"}},
		{name: "create", args: []string{"migrate", "create", "course", "sql"}},
		{name: "fix", args: []string{"migrate", "fix"}},
	}
	for _, tt := range tests {
		args := append([]string{"admin"}, tt.args...)

		t.Run(tt.name, func(t *testing.T) {
			if err := cli.run(args); err != nil {
				if tt.wantErr != nil {
					if err != tt.wantErr {
						t.Errorf("cli.run() error = %v, wantErr %v", err, tt.wantErr)
					}
				} else if tt.wantErrStr != "" {
					if err.Error() != tt.wantErrStr {
						t.Errorf("cli.run() error.Error() = %s, wantErrStr %s", err.Error(), tt.wantErrStr)
					}
				} else {
					t.Errorf("cli.run() unexpected error = %v", err)
				}
			}
		})
	}
}

func Test_commandLine_resetPassword(t *testing.T) {
	cli := setup(t)

	usr := testutil.CreateUser(t, usrRepo, "User", "awe", "awe@test.cd", "mdr", nil, true)

	type extra struct {
		pwd string
	}
	tests := []cliTest{
		{name: "no command", wantErr: errHelp},
		{name: "unknown command", args: []string{"lol"}, wantErr: errHelp},
		{name: "no args", args: []string{"resetpassword"}, wantErr: errHelp},
		{name: "username but no password", args: []string{"resetpassword", "-username", "lol"}, wantErr: errHelp},
		{name: "user not found", args: []string{"resetpassword", "-username", "lol"}, extra: extra{pwd: "lol"}, wantErr: user.ErrNotFound},
		{name: "reset with username", args: []string{"resetpassword", "-username", usr.Username}, extra: extra{pwd: "lol"}},
		{name: "reset with email", args: []string{"resetpassword", "-username", usr.Email}, extra: extra{pwd: "lmao"}},
	}
	for _, tt := range tests {
		args := append([]string{"admin"}, tt.args...)

		readPasswordFunc = func(fd int) ([]byte, error) {
			if extra, ok := tt.extra.(extra); ok {
				return []byte(extra.pwd), nil
			}
			return nil, nil
		}

		t.Run(tt.name, func(t *testing.T) {
			err := cli.run(args)
			if err == nil {
				refreshedUsr, err := usrRepo.GetUser(context.Background(), user.GetFilter{ID: usr.ID})
				if err != nil {
					t.Fatalf("GetUserByID() failed, %v", err)
				}
				if bytes.Equal(refreshedUsr.PasswordHash, usr.PasswordHash) {
					t.Error("failed to update new password")
				}
			} else if err != tt.wantErr {
				t.Errorf("cli.run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

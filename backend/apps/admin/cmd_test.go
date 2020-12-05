package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/trezcool/masomo/backend/core/user"
	"github.com/trezcool/masomo/backend/services/email/dummy"
	"github.com/trezcool/masomo/backend/storage/database/dummy"
	"github.com/trezcool/masomo/backend/tests"
)

var (
	// todo: load from test config
	appName                   = "Masomo"
	secretKey                 = []byte("secret")
	serverName                = "localhost"
	defaultFromEmail          = "noreply@" + serverName
	passwordResetTimeoutDelta = 3 * 24 * time.Hour

	usrRepo user.Repository
)

func setup(t *testing.T) *commandLine {
	// set up DB
	db, err := dummydb.Open()
	if err != nil {
		t.Fatalf("setup() failed: %v", err)
	}
	usrRepo = dummydb.NewUserRepository(db)

	// set up services
	mailSvc := dummymail.NewServiceMock(appName, defaultFromEmail)
	usrSvc := user.NewServiceMock(usrRepo, mailSvc, secretKey, passwordResetTimeoutDelta)

	// start CLI
	cli := &commandLine{
		usrSvc: usrSvc,
	}
	return cli
}

type cliTest struct {
	name    string
	args    []string // without program name
	wantErr error
	extra   interface{}
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
				refreshedUsr, err := usrRepo.GetUserByID(usr.ID)
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

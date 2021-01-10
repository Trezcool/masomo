package main

import (
	"context"

	"github.com/trezcool/masomo/core"
	"github.com/trezcool/masomo/core/user"
)

// addUser updates or creates a user.User
func (cli *commandLine) addUser(uname, email, pwd string, isAdmin bool) error {
	var usr user.User
	var err error
	ctx := context.Background()
	uname = core.CleanString(uname, true /* lower */)
	email = core.CleanString(email, true /* lower */)

	if usr, err = cli.usrRepo.GetUser(ctx, user.GetFilter{UsernameOrEmail: []string{uname, email}}); err != nil {
		if err != user.ErrNotFound {
			return err
		}
		usr = user.User{
			Username: uname,
			Email:    email,
		}
	}
	if isAdmin {
		usr.Roles = user.AllRoles
	}
	usr.SetActive(true)
	if err := usr.SetPassword(pwd); err != nil {
		return err
	}
	if _, err := cli.usrRepo.UpdateOrCreateUser(ctx, usr); err != nil {
		return err
	}
	return nil
}

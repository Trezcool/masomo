package main

import (
	"context"

	"github.com/trezcool/masomo/core/user"
)

func (cli *commandLine) addUser(uname, email, pwd string, isAdmin bool) error {
	ctx := context.Background()
	if err := cli.usrRepo.CheckUsernameUniqueness(ctx, uname, email, nil); err != nil {
		return err
	}
	usr := user.User{
		Username: uname,
		Email:    email,
	}
	if isAdmin {
		usr.Roles = user.AllRoles
	}
	usr.SetActive(true)
	if err := usr.SetPassword(pwd); err != nil {
		return err
	}
	if _, err := cli.usrRepo.CreateUser(ctx, usr); err != nil {
		return err
	}
	return nil
}

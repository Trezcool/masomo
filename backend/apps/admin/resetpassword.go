package main

import (
	"context"

	"github.com/trezcool/masomo/core/user"
)

func (cli *commandLine) resetPassword(uname, pwd string) error {
	ctx := context.Background()
	usr, err := cli.usrRepo.GetUser(ctx, user.GetFilter{UsernameOrEmail: uname})
	if err != nil {
		return err
	}
	if err := usr.SetPassword(pwd); err != nil {
		return err
	}
	if _, err := cli.usrRepo.UpdateUser(ctx, usr); err != nil {
		return err
	}
	return nil
}

package main

func (cli *commandLine) resetPassword(uname, pwd string) error {
	return cli.usrSvc.SudoResetPassword(uname, pwd)
}

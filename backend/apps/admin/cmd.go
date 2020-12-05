package main

import (
	"errors"
	"flag"
	"fmt"
	"syscall"

	"golang.org/x/term"

	"github.com/trezcool/masomo/backend/core/user"
)

var (
	readPasswordFunc = term.ReadPassword // mockable

	errHelp = errors.New("help provided")
)

type commandLine struct {
	usrSvc user.Service
}

func (cli *commandLine) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  resetpassword -username USERNAME|EMAIL - reset user's password")
}

func (cli *commandLine) run(args []string) error {
	if len(args) < 2 {
		cli.printUsage()
		return errHelp
	}

	resetPasswordCmd := flag.NewFlagSet("resetpassword", flag.ExitOnError)
	resetPasswordUname := resetPasswordCmd.String("username", "", "The user's username or email. The password will be prompted next.")

	switch args[1] {
	case "resetpassword":
		if err := resetPasswordCmd.Parse(args[2:]); err != nil {
			return err
		}
		if *resetPasswordUname == "" {
			resetPasswordCmd.Usage()
			return errHelp
		}
		fmt.Print("Enter password:")
		pwd, err := readPasswordFunc(syscall.Stdin)
		fmt.Println()
		if err != nil {
			return err
		}
		if len(pwd) == 0 {
			resetPasswordCmd.Usage()
			return errHelp
		}
		return cli.resetPassword(*resetPasswordUname, string(pwd))
	default:
		cli.printUsage()
		return errHelp
	}
}

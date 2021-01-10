package main

import (
	"database/sql"
	"flag"
	"fmt"
	"syscall"

	"github.com/pkg/errors"
	"golang.org/x/term"

	"github.com/trezcool/masomo/core"
	"github.com/trezcool/masomo/core/user"
)

var (
	addUserCmd     = flag.NewFlagSet("adduser", flag.ExitOnError)
	addUserHelpMsg = "The user's %s. One of username or email is required. The password will be prompted next"
	addUserUname   = addUserCmd.String("username", "", fmt.Sprintf(addUserHelpMsg, "username"))
	addUserEmail   = addUserCmd.String("email", "", fmt.Sprintf(addUserHelpMsg, "email"))
	addUserAdmin   = addUserCmd.Bool("admin", false, "Make the user an admin")

	resetPasswordCmd   = flag.NewFlagSet("resetpassword", flag.ExitOnError)
	resetPasswordUname = resetPasswordCmd.String("username", "", "The user's username or email. The password will be prompted next")
	readPasswordFunc   = term.ReadPassword // mockable

	errHelp = errors.New("help provided")
)

type commandLine struct {
	db      *sql.DB
	usrRepo user.Repository
}

func (cli *commandLine) printUsage() {
	if !core.Conf.TestMode {
		fmt.Printf("\n%s\n", usage)
	}
}

func (cli *commandLine) promptPassword() (string, error) {
	fmt.Print("Enter password:")
	pwd, err := readPasswordFunc(syscall.Stdin)
	if err != nil {
		return "", err
	}
	fmt.Println()
	return string(pwd), nil
}

func (cli *commandLine) run(args []string) error {
	if len(args) < 2 {
		cli.printUsage()
		return errHelp
	}

	switch args[1] {
	case "createdb":
		return cli.createDB()

	case "migrate":
		if len(args) < 3 || args[2] == "" {
			cli.printUsage()
			return errHelp
		}
		return cli.migrate(args[2:])

	case "adduser":
		if err := addUserCmd.Parse(args[2:]); err != nil {
			return err
		}
		if *addUserUname == "" && *addUserEmail == "" {
			addUserCmd.Usage()
			return errHelp
		}
		pwd, err := cli.promptPassword()
		if err != nil {
			return err
		}
		if len(pwd) == 0 {
			addUserCmd.Usage()
			return errHelp
		}
		return cli.addUser(*addUserUname, *addUserEmail, pwd, *addUserAdmin)

	case "resetpassword":
		if err := resetPasswordCmd.Parse(args[2:]); err != nil {
			return err
		}
		if *resetPasswordUname == "" {
			resetPasswordCmd.Usage()
			return errHelp
		}
		pwd, err := cli.promptPassword()
		if err != nil {
			return err
		}
		if len(pwd) == 0 {
			resetPasswordCmd.Usage()
			return errHelp
		}
		return cli.resetPassword(*resetPasswordUname, pwd)

	default:
		cli.printUsage()
		return errHelp
	}
}

var (
	usage = `Admin Command Line Interface Usage:

Commands:
  createdb                  Create the app DB

  migrate                   Migrate the DB
    up                      Migrate the DB to the most recent version available
    up-by-one               Migrate the DB to next version
    up-to VERSION           Migrate the DB to a specific VERSION
    down                    Roll back the version by 1
    down-to VERSION         Roll back to a specific VERSION
    redo                    Re-run the latest migration
    reset                   Roll back all migrations
    status                  Dump the migration status for the current DB
    version                 Print the current version of the database
    create NAME [sql|go]    Creates new migration file with the current timestamp
    fix                     Apply sequential ordering to migrations

  adduser [-username USERNAME] [-email EMAIL] [-admin]    Add new user. One of username or email is required.
                                                          Optionally make them an admin

  resetpassword -username USERNAME|EMAIL                  Reset user's password
`
)

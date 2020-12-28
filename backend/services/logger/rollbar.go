package logsvc

import (
	"log"

	"github.com/rollbar/rollbar-go"
	"github.com/rollbar/rollbar-go/errors"

	"github.com/trezcool/masomo/core"
	"github.com/trezcool/masomo/core/user"
)

type rollbarLogger struct {
	std *log.Logger
}

var _ core.Logger = (*rollbarLogger)(nil)

func NewRollbarLogger(std *log.Logger) *rollbarLogger {
	rollbar.SetToken(core.Conf.RollbarToken)
	rollbar.SetEnvironment(core.Conf.Env)
	rollbar.SetServerHost(core.Conf.Server.Host)
	rollbar.SetCodeVersion(core.Conf.Build)
	rollbar.SetStackTracer(errors.StackTracer)
	return &rollbarLogger{std: std}
}

func (l rollbarLogger) SetEnabled(enabled bool) {
	rollbar.SetEnabled(enabled)
}

// expected fmt: msg | error, map[string]interface{}, user.User
func (l rollbarLogger) prepare(msg string, args []interface{}) []interface{} {
	var usrSet bool
	newArgs := make([]interface{}, 0, len(args)+1)
	newArgs = append(newArgs, msg)
	for _, arg := range args {
		// set logged in User
		if usr, ok := arg.(user.User); ok {
			if !usrSet { // only set one User
				rollbar.SetPerson(usr.ID, usr.Username, usr.Email)
				usrSet = true
			}
		} else {
			newArgs = append(newArgs, arg)
		}
	}
	if !usrSet {
		rollbar.ClearPerson()
	}
	return newArgs
}

func (l rollbarLogger) print(msg string, args []interface{}) {
	l.std.Println(msg)
	for _, arg := range args {
		l.std.Printf("%+v\n", arg)
	}
}

func (l rollbarLogger) Debug(msg string, args ...interface{}) {
	rollbar.Debug(l.prepare(msg, args)...)
	l.print(msg, args)
}

func (l rollbarLogger) Info(msg string, args ...interface{}) {
	rollbar.Info(l.prepare(msg, args)...)
	l.print(msg, args)
}

func (l rollbarLogger) Warn(msg string, args ...interface{}) {
	rollbar.Warning(l.prepare(msg, args)...)
	l.print(msg, args)
}

func (l rollbarLogger) Error(msg string, args ...interface{}) {
	rollbar.Error(l.prepare(msg, args)...)
	l.print(msg, args)
}

func (l rollbarLogger) Fatal(msg string, args ...interface{}) {
	rollbar.Critical(l.prepare(msg, args)...)
	l.print(msg, args)
	l.std.Fatal(msg) // todo: delay ?
}

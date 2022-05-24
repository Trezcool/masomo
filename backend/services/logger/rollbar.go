package logsvc

import (
	"log"

	"github.com/rollbar/rollbar-go"
	"github.com/rollbar/rollbar-go/errors"

	"github.com/trezcool/masomo/core"
	"github.com/trezcool/masomo/core/user"
)

type RollbarLogger struct {
	std *log.Logger
}

var _ core.Logger = (*RollbarLogger)(nil)

func NewRollbarLogger(std *log.Logger, conf *core.Config) *RollbarLogger {
	rollbar.SetToken(conf.RollbarToken)
	rollbar.SetEnvironment(conf.Env)
	rollbar.SetServerHost(conf.Server.Host)
	rollbar.SetCodeVersion(conf.Build)
	rollbar.SetStackTracer(errors.StackTracer)
	return &RollbarLogger{std: std}
}

func (l RollbarLogger) Enable(enabled bool) {
	rollbar.SetEnabled(enabled)
}

// expected fmt: msg | error, map[string]interface{}, user.User
func (l RollbarLogger) prepare(msg string, args []interface{}) []interface{} {
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

func (l RollbarLogger) print(msg string, args []interface{}) {
	l.std.Println(msg)
	for _, arg := range args {
		l.std.Printf("%+v\n", arg)
	}
}

func (l RollbarLogger) Debug(msg string, args ...interface{}) {
	rollbar.Debug(l.prepare(msg, args)...)
	l.print(msg, args)
}

func (l RollbarLogger) Info(msg string, args ...interface{}) {
	rollbar.Info(l.prepare(msg, args)...)
	l.print(msg, args)
}

func (l RollbarLogger) Warn(msg string, args ...interface{}) {
	rollbar.Warning(l.prepare(msg, args)...)
	l.print(msg, args)
}

func (l RollbarLogger) Error(msg string, args ...interface{}) {
	rollbar.Error(l.prepare(msg, args)...)
	l.print(msg, args)
}

func (l RollbarLogger) Fatal(msg string, args ...interface{}) {
	rollbar.Critical(l.prepare(msg, args)...)
	l.print(msg, args)
	l.std.Fatal(msg)
}

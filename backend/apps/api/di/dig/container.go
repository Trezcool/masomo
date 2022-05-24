package dig_container

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	echoapi "github.com/trezcool/masomo/apps/api/echo"
	"github.com/trezcool/masomo/core"
	"github.com/trezcool/masomo/core/user"
	emailsvc "github.com/trezcool/masomo/services/email"
	logsvc "github.com/trezcool/masomo/services/logger"
	"github.com/trezcool/masomo/storage/database"
	boiledrepos "github.com/trezcool/masomo/storage/database/sqlboiler"
	"go.uber.org/dig"
)

type DBLoggerParam struct {
	dig.In
	Logger core.Logger `name:"dbLogger"`
}

func newLogger(conf *core.Config) core.Logger {
	stdLogger := log.New(os.Stdout, "API : ", log.LstdFlags)
	logger := logsvc.NewRollbarLogger(stdLogger, conf)
	logger.Enable(!conf.Debug)
	return logger
}

func newDBLogger(conf *core.Config) core.Logger {
	stdLogger := log.New(os.Stdout, "DB : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	logger := logsvc.NewRollbarLogger(stdLogger, conf)
	logger.Enable(!conf.Debug)
	return logger
}

func newDB(conf *core.Config, loggerParam DBLoggerParam) (*sql.DB, core.DB) {
	setUp := func() (*sql.DB, error) {
		if err := database.CreateIfNotExist(conf); err != nil {
			return nil, err
		}

		db, err := database.Open(conf)
		if err != nil {
			return nil, err
		}

		if err = database.Migrate(db); err != nil {
			return nil, err
		}
		return db, nil
	}

	db, err := setUp()
	if err != nil {
		loggerParam.Logger.Fatal(fmt.Sprintf("setting up database: %v", err), err)
	}
	return db, db
}

func newEmailService(conf *core.Config, logger core.Logger) core.EmailService {
	if conf.Debug {
		return emailsvc.NewConsoleService(conf)
	}
	return emailsvc.NewSendgridService(conf, logger)
}

func newTranslator() ut.Translator {
	_en := en.New()
	uni := ut.New(_en, _en)
	translator, _ := uni.GetTranslator("en")
	return translator
}

type NewConfigFunc func() *core.Config

// New returns a new dependency injection dig.Container
func New() *dig.Container {
	c := dig.New()

	must(c.Provide(core.NewConfig))
	must(c.Provide(newLogger))
	must(c.Provide(newDBLogger, dig.Name("dbLogger")))
	must(c.Provide(newDB))
	must(c.Provide(newEmailService))
	must(c.Provide(boiledrepos.NewUserRepository, dig.As(new(user.Repository))))
	must(c.Provide(validator.New))
	must(c.Provide(newTranslator))
	must(c.Provide(user.NewService, dig.As(new(user.ServiceInterface))))
	must(c.Provide(echoapi.NewServer))

	_ = dig.Visualize(c, os.Stdout)

	return c
}

// must exits program if err happened
func must(err error) {
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to provide dependency").Error())
	}
}

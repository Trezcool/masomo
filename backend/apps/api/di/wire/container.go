//+build wireinject

package wire_container

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/google/wire"
	echoapi "github.com/trezcool/masomo/apps/api/echo"
	"github.com/trezcool/masomo/core"
	"github.com/trezcool/masomo/core/user"
	emailsvc "github.com/trezcool/masomo/services/email"
	logsvc "github.com/trezcool/masomo/services/logger"
	"github.com/trezcool/masomo/storage/database"
	boiledrepos "github.com/trezcool/masomo/storage/database/sqlboiler"
)

func newLogger(conf *core.Config) core.Logger {
	stdLogger := log.New(os.Stdout, "API : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
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

func newDB(conf *core.Config, logger core.Logger) *sql.DB {
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
		logger.Fatal(fmt.Sprintf("setting up database: %v", err), err)
	}
	return db
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

var (
	dbSet = wire.NewSet(
		newDB,
		wire.Bind(new(core.DB), new(*sql.DB)))

	// todo: fix !!!
	userRepoSet = wire.NewSet(
		boiledrepos.NewUserRepository,
		wire.Bind(new(user.Repository), new(*boiledrepos.UserRepository)))

	// todo: fix !!!
	userSvcSet = wire.NewSet(
		user.NewService,
		wire.Bind(new(user.ServiceInterface), new(*user.Service)))

	appSet = wire.NewSet(
		core.NewConfig,
		newLogger,
		newEmailService,
		dbSet,
		userRepoSet,
		userSvcSet,
		validator.New,
		newTranslator,
		wire.Struct(new(echoapi.ServerDeps), "*"),
		echoapi.NewServer)
)

func NewConfig() *core.Config {
	wire.Build(appSet)
	return nil
}

func NewLogger() core.Logger {
	wire.Build(appSet)
	return nil
}

func NewDB() *sql.DB {
	wire.Build(appSet)
	return nil
}

func NewValidate() *validator.Validate {
	wire.Build(appSet)
	return nil
}

func NewTranslator() ut.Translator {
	wire.Build(appSet)
	return nil
}

func NewServer() *echoapi.Server {
	wire.Build(appSet)
	return nil
}

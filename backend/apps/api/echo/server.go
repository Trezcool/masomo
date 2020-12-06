package echoapi

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"

	"github.com/trezcool/masomo/backend/core"
	"github.com/trezcool/masomo/backend/core/user"
)

type (
	Options struct {
		Address        string
		DisableReqLogs bool
		UserSvc        user.Service
	}

	Server interface {
		http.Handler
		Start()
		Stop(context.Context) error
	}

	server struct {
		opts *Options
		app  *echo.Echo
	}
)

var _ Server = (*server)(nil)

func NewServer(opts *Options) Server {
	s := &server{
		opts: opts,
		app:  echo.New(),
	}
	s.setup()
	return s
}

func (s *server) setup() {
	debug := core.Conf.GetBool("debug")

	s.app.Pre(middleware.RemoveTrailingSlash())
	if !s.opts.DisableReqLogs {
		s.app.Use(middleware.Logger())
	}
	// do not recover in DEV|TEST mode
	if !(debug || core.Conf.GetBool("testMode")) {
		s.app.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{LogLevel: log.ERROR}))
	}

	s.app.HTTPErrorHandler = appHTTPErrorHandler
	s.app.Debug = debug

	s.app.GET("/", home)

	v1 := s.app.Group("/v1")
	jwt := middleware.JWTWithConfig(appJWTConfig)

	registerUserAPI(v1, jwt, s.opts.UserSvc)

	// TODO: swagger !!
}

func (s *server) Start() {
	s.app.Logger.Fatal(s.app.Start(s.opts.Address))
}

func (s *server) Stop(ctx context.Context) error {
	return s.app.Shutdown(ctx)
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) { // for tests
	s.app.ServeHTTP(w, r)
}

func home(ctx echo.Context) error {
	return ctx.String(http.StatusOK, "Welcome to Masomo API!")
}

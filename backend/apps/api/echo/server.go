package echoapi

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"

	"github.com/trezcool/masomo/core"
	"github.com/trezcool/masomo/core/user"
)

type (
	Options struct {
		Addr    string
		UserSvc user.Service
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
	s.app.Pre(middleware.RemoveTrailingSlash())
	// do not print request logs in TEST mode
	if !core.Conf.TestMode {
		s.app.Use(middleware.Logger())
	}
	// do not recover in DEV|TEST mode
	if !(core.Conf.Debug || core.Conf.TestMode) {
		s.app.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{LogLevel: log.ERROR}))
	}

	s.app.HTTPErrorHandler = appHTTPErrorHandler
	s.app.Debug = core.Conf.Debug

	// todo: health endpoints according to RFC 5785
	// "/.well-known/health-check"
	// "/.well-known/metrics"
	s.app.GET("/", home)

	grp := s.app.Group("/api")
	jwt := middleware.JWTWithConfig(appJWTConfig)

	registerUserAPI(grp, jwt, s.opts.UserSvc)

	// TODO: swagger !!
}

func (s *server) Start() {
	s.app.Logger.Fatal(s.app.Start(s.opts.Addr))
}

func (s *server) Stop(ctx context.Context) error {
	return s.app.Shutdown(ctx)
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) { // for tests
	s.app.ServeHTTP(w, r)
}

// todo: graceful shutdown

func home(ctx echo.Context) error {
	return ctx.String(http.StatusOK, "Welcome to Masomo API!")
}

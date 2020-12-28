package echoapi

import (
	"context"
	"net/http"
	"os"
	"syscall"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"

	"github.com/trezcool/masomo/core"
	"github.com/trezcool/masomo/core/user"
)

type (
	Deps struct {
		Logger  core.Logger
		UserSvc user.Service
	}

	Server struct {
		addr     string
		deps     *Deps
		app      *echo.Echo
		shutdown chan<- os.Signal
	}
)

func NewServer(shutdown chan<- os.Signal, deps *Deps) *Server {
	s := &Server{
		addr:     core.Conf.Server.Address(),
		deps:     deps,
		app:      echo.New(),
		shutdown: shutdown,
	}
	s.setup()
	return s
}

func (s *Server) setup() {
	s.app.Pre(middleware.RemoveTrailingSlash())
	// do not print request logs in TEST mode
	if !core.Conf.TestMode {
		s.app.Use(middleware.Logger())
	}
	// do not recover in DEV|TEST mode
	if !(core.Conf.Debug || core.Conf.TestMode) {
		s.app.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{LogLevel: log.ERROR}))
	}

	s.app.HTTPErrorHandler = newAppHTTPErrorHandler(s.deps.Logger, s.SignalShutdown)
	s.app.Debug = core.Conf.Debug

	// todo: health endpoints according to RFC 5785
	// "/.well-known/health-check"
	// "/.well-known/metrics"
	s.app.GET("/", home) // todo: redirect to "/api" (OpenAPI docs)

	grp := s.app.Group("/api")
	jwt := middleware.JWTWithConfig(appJWTConfig)

	registerUserAPI(grp, jwt, s.deps.UserSvc)

	// TODO: swagger !!
}

func (s *Server) Start() error {
	return s.app.Start(s.addr)
}

func (s *Server) SignalShutdown() {
	s.shutdown <- syscall.SIGTERM
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.app.Shutdown(ctx)
}

func (s *Server) Close() error {
	return s.app.Close()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { // for tests
	s.app.ServeHTTP(w, r)
}

func home(ctx echo.Context) error {
	return ctx.String(http.StatusOK, "Welcome to Masomo API!")
}

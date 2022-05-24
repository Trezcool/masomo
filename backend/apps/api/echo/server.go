package echoapi

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"go.uber.org/dig"

	"github.com/trezcool/masomo/core"
	"github.com/trezcool/masomo/core/user"
)

type (
	ServerDeps struct {
		dig.In     `wire:"-"`
		Conf       *core.Config
		Logger     core.Logger
		UserSvc    user.ServiceInterface
		Validate   *validator.Validate
		Translator ut.Translator
	}

	Server struct {
		deps     ServerDeps
		app      *echo.Echo
		shutdown chan os.Signal
		errors   chan error
	}
)

func NewServer(deps ServerDeps) *Server {
	s := &Server{
		deps:   deps,
		app:    echo.New(),
		errors: make(chan error, 1),
	}
	s.setup()
	return s
}

func (s *Server) setup() {
	// channel of shutdown signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	s.shutdown = shutdown

	s.app.Pre(middleware.RemoveTrailingSlash())
	// do not print request logs in TEST mode
	if !s.deps.Conf.TestMode {
		s.app.Use(middleware.Logger())
	}
	// do not recover in DEV|TEST mode
	if !(s.deps.Conf.Debug || s.deps.Conf.TestMode) {
		s.app.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{LogLevel: log.ERROR}))
	}

	s.app.HTTPErrorHandler = newAppHTTPErrorHandler(s.deps.Logger, s.deps.Translator, s.signalShutdown)
	s.app.Debug = s.deps.Conf.Debug

	// todo: health endpoints according to RFC 5785
	// "/.well-known/health-check"
	// "/.well-known/metrics"
	s.app.GET("/", home) // todo: redirect to "/api" (OpenAPI docs)

	grp := s.app.Group("/api")

	initAuth(s.deps.Conf)
	jwt := middleware.JWTWithConfig(appJWTConfig)

	registerUserAPI(grp, jwt, s.deps.UserSvc, s.deps.Validate, s.deps.Translator)

	// TODO: swagger !!
}

func (s *Server) Start() {
	if err := s.app.Start(s.deps.Conf.Server.Address()); err != nil {
		s.errors <- err
	}
}

func (s *Server) Errors() <-chan error {
	return s.errors
}

func (s *Server) ShutdownSignal() <-chan os.Signal {
	return s.shutdown
}

func (s *Server) signalShutdown() {
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

package echoapi

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"

	"github.com/trezcool/masomo/backend/apps/api/echo/handlers"
	"github.com/trezcool/masomo/backend/apps/api/echo/helpers"
	"github.com/trezcool/masomo/backend/core/user"
)

type Options struct {
	Debug                     bool
	AppName                   string
	SecretKey                 []byte
	JwtExpirationDelta        time.Duration
	JwtRefreshExpirationDelta time.Duration

	UserSvc user.Service
}

type server struct {
	addr string
	opts *Options
	app  *echo.Echo
}

func NewServer(addr string, opts *Options) *server {
	s := &server{
		addr: addr,
		opts: opts,
		app:  echo.New(),
	}
	s.setup()
	return s
}

func (s *server) setup() {
	s.app.Pre(middleware.RemoveTrailingSlash())
	s.app.Use(middleware.Logger())
	s.app.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{LogLevel: log.ERROR}))

	s.app.HTTPErrorHandler = helpers.AppHTTPErrorHandler
	s.app.Debug = s.opts.Debug

	s.app.GET("/", home)

	v1 := s.app.Group("/v1")
	jwt := helpers.ConfigureAuth(
		s.opts.AppName,
		s.opts.SecretKey,
		s.opts.JwtExpirationDelta,
		s.opts.JwtRefreshExpirationDelta,
	)

	handlers.RegisterUserAPI(v1, jwt, s.opts.UserSvc)

	// TODO: swagger !!
}

func (s *server) Start() {
	s.app.Logger.Fatal(s.app.Start(s.addr))
}

func home(ctx echo.Context) error {
	return ctx.String(http.StatusOK, "Welcome to Masomo API!")
}

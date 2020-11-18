package api_echo

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/trezcool/masomo/backend/api/echo/handlers"
	"github.com/trezcool/masomo/backend/api/echo/helpers"
	"github.com/trezcool/masomo/backend/apps/user"
)

type server struct {
	addr       string
	router     *echo.Echo
	usrService *user.Service
}

func NewServer(addr string, usrService *user.Service) *server {
	s := &server{
		addr:       addr,
		router:     echo.New(),
		usrService: usrService,
	}
	s.setup()
	return s
}

func (s *server) setup() {
	s.router.Pre(middleware.RemoveTrailingSlash())
	s.router.Use(middleware.Logger())
	s.router.Use(middleware.Recover())

	s.router.HTTPErrorHandler = helpers.AppHTTPErrorHandler
	//s.router.Debug = true // TODO: load from config

	s.router.GET("/", home)

	v1 := s.router.Group("/v1")
	jwt := middleware.JWTWithConfig(helpers.AppJWTConfig)

	handlers.RegisterUserAPI(v1, jwt, s.usrService)

	// TODO: swagger !!
}

func (s *server) Start() {
	s.router.Logger.Fatal(s.router.Start(s.addr))
}

func home(c echo.Context) error {
	return c.String(http.StatusOK, "Welcome to Masomo API!")
}

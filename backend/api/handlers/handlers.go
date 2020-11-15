package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/trezcool/masomo/backend/api/helpers"
	"github.com/trezcool/masomo/backend/apps/user"
)

func API(e *echo.Echo) {
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.HTTPErrorHandler = helpers.AppHTTPErrorHandler
	//e.Debug = true // TODO: load from config

	e.GET("/", home)

	v1 := e.Group("/v1")
	jwt := middleware.JWTWithConfig(helpers.AppJWTConfig)

	userRepo := user.NewRepository()
	registerUserAPI(v1, jwt, userRepo)

	// TODO: swagger !!

	// TODO: move to script | SQL data migration (dev only?)
	root := user.NewUser{
		Name:     "Root",
		Username: "root",
		Email:    "root@masomo.cd",
		Password: "LolC@t123",
		Roles:    user.AllRoles,
	}
	_, _ = userRepo.Create(root)
}

func home(c echo.Context) error {
	return c.String(http.StatusOK, "Welcome to Masomo API!")
}

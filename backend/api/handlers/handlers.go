package handlers

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/trezcool/masomo/backend/api/helpers"
	"github.com/trezcool/masomo/backend/apps/shared"
	"github.com/trezcool/masomo/backend/apps/user"
)

type appValidator struct {
	validate *validator.Validate
}

func (v appValidator) Validate(i interface{}) error {
	return v.validate.Struct(i) // TODO: StructCtx?
}

func API(e *echo.Echo) {
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Validator = &appValidator{validate: shared.Validate}
	e.HTTPErrorHandler = helpers.AppHTTPErrorHandler
	//e.Debug = true // TODO: load from config

	e.GET("/", home)

	userRepo := user.NewRepository()
	jwt := middleware.JWTWithConfig(helpers.AppJWTConfig)

	v1 := e.Group("/v1")
	registerUserAPI(v1, jwt, userRepo)

	// TODO: swagger !!
}

func home(c echo.Context) error {
	return c.String(http.StatusOK, "Welcome to Masomo Backend!")
}

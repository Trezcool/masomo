package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/trezcool/masomo/backend/api/echo/helpers"
	"github.com/trezcool/masomo/backend/apps/user"
	"github.com/trezcool/masomo/backend/apps/utils"
)

var (
	usrNotFoundInCtxErr  = errors.New("user object not found in echo.Context")
	noPermsToSetRolesErr = "not enough rights to set these roles"
)

type userApi struct {
	service *user.Service
}

func RegisterUserAPI(g *echo.Group, jwt echo.MiddlewareFunc, service *user.Service) {
	a := userApi{service: service}

	ug := g.Group("/users")

	// un-authed endpoints
	ug.POST("/login", a.userLogin)
	ug.POST("/password-reset", a.userResetPassword)
	ug.POST("/password-reset-confirm", a.userConfirmPasswordReset)

	// authed endpoints
	ag := ug.Group("", jwt)
	ag.POST("/register", a.userCreate, helpers.AdminMiddleware())
	ag.GET("", a.userQuery, helpers.AdminMiddleware())
	ag.DELETE("", a.userDestroyMultiple, helpers.AdminMiddleware())
	ag.GET("/roles", a.userQueryRoles, helpers.AdminMiddleware())

	// detail endpoints
	dg := ag.Group("/:id", ctxUserOrAdminMiddleware(a.service))
	dg.GET("", a.userRetrieve)
	dg.PUT("", a.userUpdate)
	dg.DELETE("", a.userDestroy, helpers.AdminMiddleware())
}

// Handlers

func (api *userApi) userCreate(c echo.Context) error {
	data := new(user.NewUser)
	if err := c.Bind(data); err != nil {
		return err
	}
	if err := data.Validate(api.service); err != nil {
		return err
	}

	// ctxUser cannot set a role > their own max role
	ctxUsr, err := helpers.GetContextUser(c, api.service)
	if err != nil {
		return err
	}
	if user.MaxRolePriority(data.Roles) > user.MaxRolePriority(ctxUsr.Roles) {
		return utils.NewValidationError(nil, utils.FieldError{Field: "roles", Error: noPermsToSetRolesErr})
	}

	usr, err := api.service.Create(*data)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, usr)
}

func (api *userApi) userLogin(c echo.Context) error {
	data := new(LoginRequest)
	if err := c.Bind(data); err != nil {
		return err
	}
	if err := data.Validate(); err != nil {
		return err
	}

	claims, err := helpers.Authenticate(data.Username, data.Password, api.service)
	if err != nil {
		return err
	}
	token, err := helpers.GenerateToken(claims)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, LoginResponse{Token: token})
}

func (api *userApi) userResetPassword(c echo.Context) error {
	return c.String(http.StatusOK, "user.userResetPassword")
} // TODO

func (api *userApi) userConfirmPasswordReset(c echo.Context) error {
	return c.String(http.StatusOK, "user.userConfirmPasswordReset")
} // TODO

func (api *userApi) userQuery(c echo.Context) error {
	res, err := api.service.QueryAll()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}

func (api *userApi) userRetrieve(c echo.Context) error {
	usr, ok := c.Get("object").(user.User)
	if !ok {
		return usrNotFoundInCtxErr
	}
	return c.JSON(http.StatusOK, usr)
}

func (api *userApi) userUpdate(c echo.Context) error {
	return c.String(http.StatusOK, "user.userUpdate")
} // TODO

func (api *userApi) userDestroy(c echo.Context) error {
	usr, ok := c.Get("object").(user.User)
	if !ok {
		return usrNotFoundInCtxErr
	}

	// Say No to Suicide! ctxUser cannot delete themselves
	ctxUsr, err := helpers.GetContextUser(c, api.service)
	if err != nil {
		return err
	}
	if usr.ID == ctxUsr.ID {
		return helpers.ForbiddenHttpErr
	}

	if err := api.service.Delete(usr.ID); err != nil {
		return err
	}
	return c.JSON(http.StatusNoContent, nil)
}

func (api *userApi) userDestroyMultiple(c echo.Context) error {
	// TODO: delete selected (ids via Query);
	return c.String(http.StatusOK, "user.userDestroyMultiple")
} // TODO

func (api *userApi) userQueryRoles(c echo.Context) error {
	return c.String(http.StatusOK, "user.userQueryRoles")
} // TODO

func ctxUserOrAdminMiddleware(service *user.Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if id, err := strconv.Atoi(c.Param("id")); err == nil {
				ctxUsr, err := helpers.GetContextUser(c, service)
				if err != nil {
					return err
				}

				if id == ctxUsr.ID || ctxUsr.IsAdmin() {
					usr, err := service.GetByID(id)
					if err == nil {
						c.Set("object", usr)
						return next(c)
					} else if err != user.NotFoundErr {
						return err
					}
				}
			}
			return helpers.NotFoundHttpErr
		}
	}
}

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

func (lr *LoginRequest) Validate() error {
	lr.Username = utils.CleanString(lr.Username, true)
	return utils.Validate.Struct(lr)
}

type LoginResponse struct {
	Token string `json:"token"`
}
